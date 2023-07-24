package rtc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/pion/webrtc/v3"

	"go.uber.org/atomic"
	"google.golang.org/protobuf/proto"

	"github.com/livekit/livekit-server/pkg/p2p"
	"github.com/livekit/livekit-server/pkg/rtc/relay"
	"github.com/livekit/livekit-server/pkg/rtc/relay/pc"

	"github.com/livekit/protocol/livekit"
	"github.com/livekit/protocol/logger"
	"github.com/livekit/protocol/utils"

	"github.com/livekit/livekit-server/pkg/sfu/connectionquality"

	"github.com/livekit/livekit-server/pkg/config"
	"github.com/livekit/livekit-server/pkg/routing"
	"github.com/livekit/livekit-server/pkg/rtc/types"
	"github.com/livekit/livekit-server/pkg/sfu/buffer"
	"github.com/livekit/livekit-server/pkg/telemetry"
	"github.com/livekit/livekit-server/pkg/telemetry/prometheus"

	cfg "github.com/livekit/livekit-server/pkg/config"
)

const (
	DefaultEmptyTimeout       = 5 * 60 // 5m
	AudioLevelQuantization    = 8      // ideally power of 2 to minimize float decimal
	invAudioLevelQuantization = 1.0 / AudioLevelQuantization
	subscriberUpdateInterval  = 3 * time.Second

	dataForwardLoadBalanceThreshold = 20
)

var (
	// var to allow unit test override
	RoomDepartureGrace uint32 = 20
)

type broadcastOptions struct {
	skipSource bool
	immediate  bool
}

type Room struct {
	lock sync.RWMutex

	protoRoom *livekit.Room
	internal  *livekit.RoomInternal
	Logger    logger.Logger

	Config         WebRTCConfig
	audioConfig    *config.AudioConfig
	serverInfo     *livekit.ServerInfo
	telemetry      telemetry.TelemetryService
	egressLauncher EgressLauncher
	trackManager   *RoomTrackManager

	// map of identity -> Participant
	participants              map[livekit.ParticipantIdentity]types.LocalParticipant
	participantOpts           map[livekit.ParticipantIdentity]*ParticipantOptions
	participantRequestSources map[livekit.ParticipantIdentity]routing.MessageSource
	bufferFactory             *buffer.FactoryOfBufferFactory

	// batch update participant info for non-publishers
	batchedUpdates   map[livekit.ParticipantIdentity]*livekit.ParticipantInfo
	batchedUpdatesMu sync.Mutex

	relayedParticipants   map[livekit.ParticipantIdentity]*RelayedParticipantImpl
	relayedParticipantsMu sync.Mutex
	outRelayCollection    *relay.Collection

	// time the first participant joined the room
	joinedAt atomic.Int64
	holds    atomic.Int32
	// time that the last participant left the room
	leftAt atomic.Int64
	closed chan struct{}

	onParticipantChanged func(p types.LocalParticipant)
	onMetadataUpdate     func(metadata string)
	onClose              func()
}

type ParticipantOptions struct {
	AutoSubscribe bool
}

type signalPeerMessage struct {
	ReplyTo string `json:"replyTo"`
	Signal  string `json:"signal"`
}

type relayMessage struct {
	Updates    []*livekit.ParticipantInfo `json:"updates,omitempty"`
	DataPacket []byte
}

func packSignalPeerMessage(replyTo string, signal []byte) interface{} {
	return &signalPeerMessage{
		ReplyTo: replyTo,
		Signal:  base64.StdEncoding.EncodeToString(signal),
	}
}

func unpackSignalPeerMessage(message interface{}) (replyTo string, signal []byte, err error) {
	messageMap, ok := message.(map[string]interface{})
	if !ok {
		err = errors.New("cannot cast")
		return
	}

	replyToValue, ok := messageMap["replyTo"]
	if !ok {
		err = errors.New("ReplyTo undefined")
		return
	}
	replyTo, ok = replyToValue.(string)
	if !ok {
		err = errors.New("cannot cast ReplyTo to string")
		return
	}

	signalBase64Value, ok := messageMap["signal"]
	if !ok {
		err = errors.New("Signal undefined")
		return
	}
	signalBase64, ok := signalBase64Value.(string)
	if !ok {
		err = errors.New("cannot cast Signal to string")
		return
	}
	signal, err = base64.StdEncoding.DecodeString(signalBase64)

	return
}

func NewRoom(
	room *livekit.Room,
	internal *livekit.RoomInternal,
	config WebRTCConfig,
	audioConfig *config.AudioConfig,
	serverInfo *livekit.ServerInfo,
	telemetry telemetry.TelemetryService,
	egressLauncher EgressLauncher,
	roomP2PCommunicator p2p.RoomCommunicator,
) *Room {
	logger := LoggerWithRoom(logger.GetLogger(), livekit.RoomKey(room.Key), livekit.RoomID(room.Sid))

	r := &Room{
		protoRoom:                 proto.Clone(room).(*livekit.Room),
		internal:                  internal,
		Logger:                    logger,
		Config:                    config,
		audioConfig:               audioConfig,
		telemetry:                 telemetry,
		egressLauncher:            egressLauncher,
		trackManager:              NewRoomTrackManager(),
		serverInfo:                serverInfo,
		participants:              make(map[livekit.ParticipantIdentity]types.LocalParticipant),
		participantOpts:           make(map[livekit.ParticipantIdentity]*ParticipantOptions),
		participantRequestSources: make(map[livekit.ParticipantIdentity]routing.MessageSource),
		bufferFactory:             buffer.NewFactoryOfBufferFactory(config.Receiver.PacketBufferSize),
		batchedUpdates:            make(map[livekit.ParticipantIdentity]*livekit.ParticipantInfo),
		closed:                    make(chan struct{}),

		relayedParticipants: make(map[livekit.ParticipantIdentity]*RelayedParticipantImpl),
		outRelayCollection:  relay.NewCollection(),
	}
	if r.protoRoom.EmptyTimeout == 0 {
		r.protoRoom.EmptyTimeout = DefaultEmptyTimeout
	}
	if r.protoRoom.CreationTime == 0 {
		r.protoRoom.CreationTime = time.Now().Unix()
	}

	go r.audioUpdateWorker()
	go r.connectionQualityWorker()
	go r.subscriberBroadcastWorker()

	pendingAnswers := map[string]chan []byte{}
	pendingAnswersMu := sync.Mutex{}

	roomP2PCommunicator.ForEachPeer(func(peerId string) {
		logger.Infow("New p2p peer", "peerId", peerId)
		rel, err := pc.NewRelay(logger, &relay.RelayConfig{
			ID:            peerId,
			BufferFactory: r.GetBufferFactory(),
			SettingEngine: config.SettingEngine,
			ICEServers:    config.Configuration.ICEServers,
		})
		if err != nil {
			logger.Errorw("New out relay", err)
			return
		}

		rel.OnReady(func() {
			logger.Infow("Out relay is ready")
			updates := ToProtoParticipants(r.GetParticipants())
			if len(updates) > 0 {
				if updatesForRelay, err := r.getUpdatesPayloadForRelay(updates); err != nil {
					r.Logger.Errorw("could not create participant update for relay", err)
				} else if err := rel.SendMessage(updatesForRelay); err != nil {
					r.Logger.Errorw("could not send participant updates to relay", err)
				}
			}
			r.outRelayCollection.AddRelay(rel)
		})

		rel.OnConnectionStateChange(func(state webrtc.ICEConnectionState) {
			logger.Infow("Out relay connection state changed", "state", state)
		})

		signalFn := func(offer []byte) ([]byte, error) {
			answer := make(chan []byte, 1)

			pendingAnswersMu.Lock()
			msgId, sendErr := roomP2PCommunicator.SendMessage(peerId, packSignalPeerMessage("", offer))
			if sendErr != nil {
				pendingAnswersMu.Unlock()
				return nil, err
			}
			logger.Infow("offer sent")
			pendingAnswers[msgId] = answer
			pendingAnswersMu.Unlock()

			defer func() {
				pendingAnswersMu.Lock()
				delete(pendingAnswers, msgId)
				pendingAnswersMu.Unlock()
			}()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			select {
			case a := <-answer:
				return a, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		if err := rel.Offer(signalFn); err != nil {
			logger.Errorw("Relay Offer", err)
		}
	})

	roomP2PCommunicator.OnMessage(func(message interface{}, fromPeerId string, eventId string) {
		replyTo, signal, err := unpackSignalPeerMessage(message)
		if err != nil {
			logger.Errorw("Unmarshal signal peer message", err)
			return
		}
		if len(replyTo) > 0 {
			// Answer
			pendingAnswersMu.Lock()
			if answer, ok := pendingAnswers[replyTo]; ok {
				answer <- signal
			}
			pendingAnswersMu.Unlock()
		} else {
			// Offer
			rel, err := pc.NewRelay(logger, &relay.RelayConfig{
				ID:            fromPeerId,
				BufferFactory: r.GetBufferFactory(),
				SettingEngine: config.SettingEngine,
				ICEServers:    config.Configuration.ICEServers,
			})
			if err != nil {
				logger.Errorw("New in-relay", err)
				return
			}

			rel.OnReady(func() {
				logger.Infow("In-relay is ready")
				// TODO
			})

			rel.OnConnectionStateChange(func(state webrtc.ICEConnectionState) {
				logger.Infow("In-relay connection state changed", "state", state)
			})

			answer, answerErr := rel.Answer(signal)
			if answerErr != nil {
				logger.Errorw("In-relay answer", answerErr)
				return
			}

			if _, err := roomP2PCommunicator.SendMessage(fromPeerId, packSignalPeerMessage(eventId, answer)); err != nil {
				logger.Errorw("can not send answer", err)
				return
			}

			logger.Infow("answer sent")

			rel.OnMessage(func(id uint64, payload []byte) {
				logger.Infow("Relay message received")
				var msg relayMessage

				if err := json.Unmarshal(payload, &msg); err != nil {
					r.Logger.Errorw("could not unmarshal relay message", err)
					return
				}
				if len(msg.Updates) > 0 {
					for _, update := range msg.Updates {
						r.onRelayParticipantUpdate(rel, update)
					}
					r.sendParticipantUpdates(msg.Updates)
				}
				if len(msg.DataPacket) > 0 {
					dp := livekit.DataPacket{}
					if err := proto.Unmarshal(msg.DataPacket, &dp); err != nil {
						r.Logger.Errorw("could not unmarshal relay data packet", err)
						return
					} else {
						BroadcastDataPacketForRoom(r, nil, &dp, r.Logger)
					}
				}
			})

			rel.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver, mid string, rid string, meta []byte) {
				logger.Infow("Relay track published", "mid", mid, "rid", rid)
				var addTrackSignal AddTrackSignal
				if err := json.Unmarshal(meta, &addTrackSignal); err != nil {
					r.Logger.Errorw("unmarshal err", err)
					return
				}
				r.onRelayAddTrack(rel, track, receiver, mid, rid, addTrackSignal)
			})
		}
	})

	return r
}

func (r *Room) onRelayParticipantUpdate(rel relay.Relay, pi *livekit.ParticipantInfo) {
	r.relayedParticipantsMu.Lock()
	defer r.relayedParticipantsMu.Unlock()

	participantIdentity := livekit.ParticipantIdentity(pi.Identity)

	remoteParticipant, exists := r.relayedParticipants[participantIdentity]
	if !exists {
		rtcConfig := r.Config
		rtcConfig.SetBufferFactory(rel.GetBufferFactory())

		remoteParticipant, _ = NewRelayedParticipant(RelayedParticipantParams{
			Identity: participantIdentity,
			Name:     livekit.ParticipantName(pi.Name),
			SID:      livekit.ParticipantID(pi.Sid),
			Config:   &rtcConfig,
			AudioConfig: cfg.AudioConfig{
				ActiveLevel:     35, // -35dBov
				MinPercentile:   40,
				UpdateInterval:  400,
				SmoothIntervals: 2,
			},
			VideoConfig: cfg.VideoConfig{
				DynacastPauseDelay: 5 * time.Second,
				StreamTracker: cfg.StreamTrackersConfig{
					Video: cfg.StreamTrackerConfig{
						StreamTrackerType: cfg.StreamTrackerTypePacket,
						BitrateReportInterval: map[int32]time.Duration{
							0: 1 * time.Second,
							1: 1 * time.Second,
							2: 1 * time.Second,
						},
						PacketTracker: map[int32]cfg.StreamTrackerPacketConfig{
							0: {
								SamplesRequired: 1,
								CyclesRequired:  4,
								CycleDuration:   500 * time.Millisecond,
							},
							1: {
								SamplesRequired: 5,
								CyclesRequired:  20,
								CycleDuration:   500 * time.Millisecond,
							},
							2: {
								SamplesRequired: 5,
								CyclesRequired:  20,
								CycleDuration:   500 * time.Millisecond,
							},
						},
						FrameTracker: map[int32]cfg.StreamTrackerFrameConfig{
							0: {
								MinFPS: 5.0,
							},
							1: {
								MinFPS: 5.0,
							},
							2: {
								MinFPS: 5.0,
							},
						},
					},
					Screenshare: cfg.StreamTrackerConfig{
						StreamTrackerType: cfg.StreamTrackerTypePacket,
						BitrateReportInterval: map[int32]time.Duration{
							0: 4 * time.Second,
							1: 4 * time.Second,
							2: 4 * time.Second,
						},
						PacketTracker: map[int32]cfg.StreamTrackerPacketConfig{
							0: {
								SamplesRequired: 1,
								CyclesRequired:  1,
								CycleDuration:   2 * time.Second,
							},
							1: {
								SamplesRequired: 1,
								CyclesRequired:  1,
								CycleDuration:   2 * time.Second,
							},
							2: {
								SamplesRequired: 1,
								CyclesRequired:  1,
								CycleDuration:   2 * time.Second,
							},
						},
						FrameTracker: map[int32]cfg.StreamTrackerFrameConfig{
							0: {
								MinFPS: 0.5,
							},
							1: {
								MinFPS: 0.5,
							},
							2: {
								MinFPS: 0.5,
							},
						},
					},
				},
			},
			Logger: LoggerWithParticipant(r.Logger, livekit.ParticipantIdentity(pi.Identity), livekit.ParticipantID(pi.Sid), true),
			// SimTracks: nil,
			// InitialVersion: 0,
			Telemetry: r.telemetry,
			PLIThrottleConfig: cfg.PLIThrottleConfig{
				LowQuality:  500 * time.Millisecond,
				MidQuality:  time.Second,
				HighQuality: time.Second,
			},
			VersionGenerator: utils.NewDefaultTimedVersionGenerator(),
			Relay:            rel,
		})
		opts := ParticipantOptions{
			AutoSubscribe: false,
		}
		if err := r.Join(remoteParticipant, nil, &opts, nil); err != nil {
			logger.Errorw("Can not join remote participant", err, "Identity", remoteParticipant.Identity())
			return
		}
		logger.Infow("Remote participant joined", "Identity", remoteParticipant.Identity())
		r.relayedParticipants[participantIdentity] = remoteParticipant
	}
	remoteParticipant.SetName(pi.Name)
	remoteParticipant.SetState(pi.State)
	remoteParticipant.SetMetadata(pi.Metadata)
}

func (r *Room) onRelayAddTrack(rel relay.Relay, track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver, mid string, rid string, addTrackSignal AddTrackSignal) {
	participantIdentity := livekit.ParticipantIdentity(addTrackSignal.Identity)

	r.relayedParticipantsMu.Lock()
	defer r.relayedParticipantsMu.Unlock()

	if remoteParticipant, exists := r.relayedParticipants[participantIdentity]; exists {
		remoteParticipant.OnMediaTrack(track, receiver, mid, rid, addTrackSignal.Track)
	} else {
		r.Logger.Errorw("unknown relayed participant", nil)
	}
}

func (r *Room) ToProto() *livekit.Room {
	r.lock.RLock()
	defer r.lock.RUnlock()

	return proto.Clone(r.protoRoom).(*livekit.Room)
}

func (r *Room) Name() livekit.RoomName {
	return livekit.RoomName(r.protoRoom.Name)
}

func (r *Room) Key() livekit.RoomKey {
	return livekit.RoomKey(r.protoRoom.Key)
}

func (r *Room) ID() livekit.RoomID {
	return livekit.RoomID(r.protoRoom.Sid)
}

func (r *Room) GetParticipant(identity livekit.ParticipantIdentity) types.LocalParticipant {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.participants[identity]
}

func (r *Room) GetParticipantByID(participantID livekit.ParticipantID) types.LocalParticipant {
	r.lock.RLock()
	defer r.lock.RUnlock()

	for _, p := range r.participants {
		if p.ID() == participantID {
			return p
		}
	}

	return nil
}

func (r *Room) GetParticipants() []types.LocalParticipant {
	r.lock.RLock()
	defer r.lock.RUnlock()
	participants := make([]types.LocalParticipant, 0, len(r.participants))
	for _, p := range r.participants {
		participants = append(participants, p)
	}
	return participants
}

func (r *Room) GetLocalParticipants() []types.LocalParticipant {
	return r.GetParticipants()
}

func (r *Room) GetActiveSpeakers() []*livekit.SpeakerInfo {
	participants := r.GetParticipants()
	speakers := make([]*livekit.SpeakerInfo, 0, len(participants))
	for _, p := range participants {
		level, active := p.GetAudioLevel()
		if !active {
			continue
		}
		speakers = append(speakers, &livekit.SpeakerInfo{
			Sid:    string(p.ID()),
			Level:  float32(level),
			Active: active,
		})
	}

	sort.Slice(speakers, func(i, j int) bool {
		return speakers[i].Level > speakers[j].Level
	})

	// quantize to smooth out small changes
	for _, speaker := range speakers {
		speaker.Level = float32(math.Ceil(float64(speaker.Level*AudioLevelQuantization)) * invAudioLevelQuantization)
	}

	return speakers
}

func (r *Room) GetBufferFactory() *buffer.Factory {
	return r.bufferFactory.CreateBufferFactory()
}

func (r *Room) FirstJoinedAt() int64 {
	return r.joinedAt.Load()
}

func (r *Room) LastLeftAt() int64 {
	return r.leftAt.Load()
}

func (r *Room) Internal() *livekit.RoomInternal {
	return r.internal
}

func (r *Room) Hold() bool {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.IsClosed() {
		return false
	}

	r.holds.Inc()
	return true
}

func (r *Room) Release() {
	r.holds.Dec()
}

func (r *Room) Join(participant types.LocalParticipant, requestSource routing.MessageSource, opts *ParticipantOptions, iceServers []*livekit.ICEServer) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.IsClosed() {
		return ErrRoomClosed
	}

	if r.participants[participant.Identity()] != nil {
		return ErrAlreadyJoined
	}

	if r.protoRoom.MaxParticipants > 0 && !participant.IsRecorder() {
		participantCount := 0
		for _, p := range r.participants {
			if !p.IsRecorder() {
				participantCount++
			}
		}

		if participantCount >= int(r.protoRoom.MaxParticipants) {
			return ErrMaxParticipantsExceeded
		}
	}

	if r.FirstJoinedAt() == 0 {
		r.joinedAt.Store(time.Now().Unix())
	}
	if !participant.Hidden() {
		r.protoRoom.NumParticipants++
	}

	// it's important to set this before connection, we don't want to miss out on any published tracks
	participant.OnTrackPublished(r.onTrackPublished)
	participant.OnStateChange(func(p types.LocalParticipant, oldState livekit.ParticipantInfo_State) {
		r.Logger.Infow("participant state changed",
			"state", p.State(),
			"participant", p.Identity(),
			"pID", p.ID(),
			"oldState", oldState)
		if r.onParticipantChanged != nil {
			r.onParticipantChanged(participant)
		}
		r.broadcastParticipantState(p, broadcastOptions{skipSource: true})

		state := p.State()
		if state == livekit.ParticipantInfo_ACTIVE {
			// subscribe participant to existing published tracks
			r.subscribeToExistingTracks(p)

			// start the workers once connectivity is established
			p.Start()

			r.telemetry.ParticipantActive(context.Background(), r.ToProto(), p.ToProto(), &livekit.AnalyticsClientMeta{
				ClientConnectTime: uint32(time.Since(p.ConnectedAt()).Milliseconds()),
				ConnectionType:    string(p.GetICEConnectionType()),
			}, p.ClaimGrants().WebHookURL)
		} else if state == livekit.ParticipantInfo_DISCONNECTED {
			// remove participant from room
			go r.RemoveParticipant(p.Identity(), p.ID(), types.ParticipantCloseReasonStateDisconnected)
		}
	})
	participant.OnTrackUpdated(r.onTrackUpdated)
	participant.OnTrackUnpublished(r.onTrackUnpublished)
	participant.OnParticipantUpdate(r.onParticipantUpdate)
	participant.OnDataPacket(r.onDataPacket)
	participant.OnSubscribeStatusChanged(func(publisherID livekit.ParticipantID, subscribed bool) {
		if subscribed {
			pub := r.GetParticipantByID(publisherID)
			if pub != nil && pub.State() == livekit.ParticipantInfo_ACTIVE {
				// when a participant subscribes to another participant,
				// send speaker update if the subscribed to participant is active.
				level, active := pub.GetAudioLevel()
				if active {
					_ = participant.SendSpeakerUpdate([]*livekit.SpeakerInfo{
						{
							Sid:    string(pub.ID()),
							Level:  float32(level),
							Active: active,
						},
					}, false)
				}

				if cq := pub.GetConnectionQuality(); cq != nil {
					update := &livekit.ConnectionQualityUpdate{}
					update.Updates = append(update.Updates, cq)
					_ = participant.SendConnectionQualityUpdate(update)
				}
			}
		} else {
			// no longer subscribed to the publisher, clear speaker status
			_ = participant.SendSpeakerUpdate([]*livekit.SpeakerInfo{
				{
					Sid:    string(publisherID),
					Level:  0,
					Active: false,
				},
			}, true)
		}

	})
	r.Logger.Infow("new participant joined",
		"pID", participant.ID(),
		"participant", participant.Identity(),
		"protocol", participant.ProtocolVersion(),
		"options", opts)

	if participant.IsRecorder() && !r.protoRoom.ActiveRecording {
		r.protoRoom.ActiveRecording = true
		r.sendRoomUpdateLocked()
	}

	r.participants[participant.Identity()] = participant
	r.participantOpts[participant.Identity()] = opts
	r.participantRequestSources[participant.Identity()] = requestSource

	if r.onParticipantChanged != nil {
		r.onParticipantChanged(participant)
	}

	time.AfterFunc(time.Minute, func() {
		state := participant.State()
		if state == livekit.ParticipantInfo_JOINING || state == livekit.ParticipantInfo_JOINED {
			r.RemoveParticipant(participant.Identity(), participant.ID(), types.ParticipantCloseReasonJoinTimeout)
		}
	})

	joinResponse := r.createJoinResponseLocked(participant, iceServers)
	if err := participant.SendJoinResponse(joinResponse); err != nil {
		prometheus.ServiceOperationCounter.WithLabelValues("participant_join", "error", "send_response").Add(1)
		return err
	}

	participant.SetMigrateState(types.MigrateStateComplete)

	if participant.SubscriberAsPrimary() {
		// initiates sub connection as primary
		if participant.ProtocolVersion().SupportFastStart() {
			go func() {
				r.subscribeToExistingTracks(participant)
				participant.Negotiate(true)
			}()
		} else {
			participant.Negotiate(true)
		}
	}

	prometheus.ServiceOperationCounter.WithLabelValues("participant_join", "success", "").Add(1)

	return nil
}

func (r *Room) ReplaceParticipantRequestSource(identity livekit.ParticipantIdentity, reqSource routing.MessageSource) {
	r.lock.Lock()
	if rs, ok := r.participantRequestSources[identity]; ok {
		rs.Close()
	}
	r.participantRequestSources[identity] = reqSource

	r.lock.Unlock()
}

func (r *Room) GetParticipantRequestSource(identity livekit.ParticipantIdentity) routing.MessageSource {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.participantRequestSources[identity]
}

func (r *Room) ResumeParticipant(p types.LocalParticipant, requestSource routing.MessageSource, responseSink routing.MessageSink, iceServers []*livekit.ICEServer, reason livekit.ReconnectReason) error {
	r.ReplaceParticipantRequestSource(p.Identity(), requestSource)
	// close previous sink, and link to new one
	p.CloseSignalConnection()
	p.SetResponseSink(responseSink)

	p.SetSignalSourceValid(true)

	if err := p.SendReconnectResponse(&livekit.ReconnectResponse{
		IceServers:          iceServers,
		ClientConfiguration: p.GetClientConfiguration(),
	}); err != nil {
		return err
	}

	updates := ToProtoParticipants(r.GetParticipants())
	if err := p.SendParticipantUpdate(updates); err != nil {
		return err
	}

	r.lock.RLock()
	p.SendRoomUpdate(r.protoRoom)
	r.lock.RUnlock()

	p.ICERestart(nil, reason)
	return nil
}

func (r *Room) RemoveParticipant(identity livekit.ParticipantIdentity, pID livekit.ParticipantID, reason types.ParticipantCloseReason) {
	r.lock.Lock()
	p, ok := r.participants[identity]
	if ok {
		if pID != "" && p.ID() != pID {
			// participant session has been replaced
			r.lock.Unlock()
			return
		}

		delete(r.participants, identity)
		delete(r.participantOpts, identity)
		delete(r.participantRequestSources, identity)

		r.relayedParticipantsMu.Lock()
		delete(r.relayedParticipants, identity)
		r.relayedParticipantsMu.Unlock()

		if !p.Hidden() {
			r.protoRoom.NumParticipants--
		}
	}

	if (p != nil && p.IsRecorder()) || r.protoRoom.ActiveRecording {
		activeRecording := false
		for _, op := range r.participants {
			if op.IsRecorder() {
				activeRecording = true
				break
			}
		}

		if r.protoRoom.ActiveRecording != activeRecording {
			r.protoRoom.ActiveRecording = activeRecording
			r.sendRoomUpdateLocked()
		}
	}
	r.lock.Unlock()

	if !ok {
		return
	}

	// send broadcast only if it's not already closed
	sendUpdates := !p.IsDisconnected()

	// remove all published tracks
	for _, t := range p.GetPublishedTracks() {
		r.trackManager.RemoveTrack(t)
	}

	p.OnTrackUpdated(nil)
	p.OnTrackPublished(nil)
	p.OnTrackUnpublished(nil)
	p.OnStateChange(nil)
	p.OnParticipantUpdate(nil)
	p.OnDataPacket(nil)
	p.OnSubscribeStatusChanged(nil)

	// close participant as well
	r.Logger.Debugw("closing participant for removal", "pID", p.ID(), "participant", p.Identity())
	_ = p.Close(true, reason)

	r.leftAt.Store(time.Now().Unix())

	if sendUpdates {
		if r.onParticipantChanged != nil {
			r.onParticipantChanged(p)
		}
		r.broadcastParticipantState(p, broadcastOptions{skipSource: true})
	}
}

func (r *Room) UpdateSubscriptions(
	participant types.LocalParticipant,
	trackIDs []livekit.TrackID,
	participantTracks []*livekit.ParticipantTracks,
	subscribe bool,
) {
	// handle subscription changes
	for _, trackID := range trackIDs {
		if subscribe {
			participant.SubscribeToTrack(trackID)
		} else {
			participant.UnsubscribeFromTrack(trackID)
		}
	}

	for _, pt := range participantTracks {
		for _, trackID := range livekit.StringsAsTrackIDs(pt.TrackSids) {
			if subscribe {
				participant.SubscribeToTrack(trackID)
			} else {
				participant.UnsubscribeFromTrack(trackID)
			}
		}
	}
}

func (r *Room) SyncState(participant types.LocalParticipant, state *livekit.SyncState) error {
	return nil
}

func (r *Room) UpdateSubscriptionPermission(participant types.LocalParticipant, subscriptionPermission *livekit.SubscriptionPermission) error {
	if err := participant.UpdateSubscriptionPermission(subscriptionPermission, nil, r.GetParticipant, r.GetParticipantByID); err != nil {
		return err
	}
	for _, track := range participant.GetPublishedTracks() {
		r.trackManager.NotifyTrackChanged(track.ID())
	}
	return nil
}

func (r *Room) RemoveDisallowedSubscriptions(sub types.LocalParticipant, disallowedSubscriptions map[livekit.TrackID]livekit.ParticipantID) {
	for trackID, publisherID := range disallowedSubscriptions {
		pub := r.GetParticipantByID(publisherID)
		if pub == nil {
			continue
		}

		track := pub.GetPublishedTrack(trackID)
		if track != nil {
			track.RemoveSubscriber(sub.ID(), false)
		}
	}
}

func (r *Room) UpdateVideoLayers(participant types.Participant, updateVideoLayers *livekit.UpdateVideoLayers) error {
	return participant.UpdateVideoLayers(updateVideoLayers)
}

func (r *Room) ResolveMediaTrackForSubscriber(subIdentity livekit.ParticipantIdentity, trackID livekit.TrackID) types.MediaResolverResult {
	res := types.MediaResolverResult{}

	info := r.trackManager.GetTrackInfo(trackID)
	res.TrackChangedNotifier = r.trackManager.GetOrCreateTrackChangeNotifier(trackID)

	if info == nil {
		return res
	}

	res.Track = info.Track
	res.TrackRemovedNotifier = r.trackManager.GetOrCreateTrackRemoveNotifier(trackID)
	res.PublisherIdentity = info.PublisherIdentity
	res.PublisherID = info.PublisherID

	pub := r.GetParticipantByID(info.PublisherID)
	// when publisher is not found, we will assume it doesn't have permission to access
	if pub != nil {
		res.HasPermission = pub.HasPermission(trackID, subIdentity)
	}

	return res
}

func (r *Room) IsClosed() bool {
	select {
	case <-r.closed:
		return true
	default:
		return false
	}
}

// CloseIfEmpty closes the room if all participants had left, or it's still empty past timeout
func (r *Room) CloseIfEmpty() {
	r.lock.Lock()

	if r.IsClosed() || r.holds.Load() > 0 {
		r.lock.Unlock()
		return
	}

	for _, p := range r.participants {
		if !p.IsRecorder() {
			r.lock.Unlock()
			return
		}
	}

	var timeout uint32
	var elapsed int64
	if r.FirstJoinedAt() > 0 && r.LastLeftAt() > 0 {
		elapsed = time.Now().Unix() - r.LastLeftAt()
		// need to give time in case participant is reconnecting
		timeout = RoomDepartureGrace
	} else {
		elapsed = time.Now().Unix() - r.protoRoom.CreationTime
		timeout = r.protoRoom.EmptyTimeout
	}
	r.lock.Unlock()

	if elapsed >= int64(timeout) {
		r.Close()
	}
}

func (r *Room) Close() {
	r.lock.Lock()
	select {
	case <-r.closed:
		r.lock.Unlock()
		return
	default:
		// fall through
	}
	close(r.closed)
	r.lock.Unlock()
	r.Logger.Infow("closing room")
	for _, p := range r.GetParticipants() {
		_ = p.Close(true, types.ParticipantCloseReasonRoomClose)
	}
	if r.onClose != nil {
		r.onClose()
	}
}

func (r *Room) OnClose(f func()) {
	r.onClose = f
}

func (r *Room) OnParticipantChanged(f func(participant types.LocalParticipant)) {
	r.onParticipantChanged = f
}

func (r *Room) SendDataPacket(up *livekit.UserPacket, kind livekit.DataPacket_Kind) {
	dp := &livekit.DataPacket{
		Kind: kind,
		Value: &livekit.DataPacket_User{
			User: up,
		},
	}
	r.onDataPacket(nil, dp)
}

func (r *Room) SetMetadata(metadata string) {
	r.lock.Lock()
	r.protoRoom.Metadata = metadata
	r.lock.Unlock()

	r.lock.RLock()
	r.sendRoomUpdateLocked()
	r.lock.RUnlock()

	if r.onMetadataUpdate != nil {
		r.onMetadataUpdate(metadata)
	}
}

func (r *Room) sendRoomUpdateLocked() {
	// Send update to participants
	for _, p := range r.participants {
		if !p.IsReady() {
			continue
		}

		err := p.SendRoomUpdate(r.protoRoom)
		if err != nil {
			r.Logger.Warnw("failed to send room update", err, "participant", p.Identity())
		}
	}
}

func (r *Room) OnMetadataUpdate(f func(metadata string)) {
	r.onMetadataUpdate = f
}

func (r *Room) SimulateScenario(participant types.LocalParticipant, simulateScenario *livekit.SimulateScenario) error {
	switch scenario := simulateScenario.Scenario.(type) {
	case *livekit.SimulateScenario_SpeakerUpdate:
		r.Logger.Infow("simulating speaker update", "participant", participant.Identity())
		go func() {
			<-time.After(time.Duration(scenario.SpeakerUpdate) * time.Second)
			r.sendSpeakerChanges([]*livekit.SpeakerInfo{{
				Sid:    string(participant.ID()),
				Active: false,
				Level:  0,
			}})
		}()
		r.sendSpeakerChanges([]*livekit.SpeakerInfo{{
			Sid:    string(participant.ID()),
			Active: true,
			Level:  0.9,
		}})
	case *livekit.SimulateScenario_Migration:
		r.Logger.Infow("simulating migration", "participant", participant.Identity())
		// drop participant without necessarily cleaning up
		if err := participant.Close(false, types.ParticipantCloseReasonSimulateMigration); err != nil {
			return err
		}
	case *livekit.SimulateScenario_NodeFailure:
		r.Logger.Infow("simulating node failure", "participant", participant.Identity())
		// drop participant without necessarily cleaning up
		if err := participant.Close(false, types.ParticipantCloseReasonSimulateNodeFailure); err != nil {
			return err
		}
	case *livekit.SimulateScenario_ServerLeave:
		r.Logger.Infow("simulating server leave", "participant", participant.Identity())
		if err := participant.Close(true, types.ParticipantCloseReasonSimulateServerLeave); err != nil {
			return err
		}

	case *livekit.SimulateScenario_SwitchCandidateProtocol:
		r.Logger.Infow("simulating switch candidate protocol", "participant", participant.Identity())
		participant.ICERestart(&livekit.ICEConfig{
			PreferenceSubscriber: livekit.ICECandidateType(scenario.SwitchCandidateProtocol),
			PreferencePublisher:  livekit.ICECandidateType(scenario.SwitchCandidateProtocol),
		}, livekit.ReconnectReason_RR_SWITCH_CANDIDATE)
	}
	return nil
}

// checks if participant should be autosubscribed to new tracks, assumes lock is already acquired
func (r *Room) autoSubscribe(participant types.LocalParticipant) bool {
	opts := r.participantOpts[participant.Identity()]
	// default to true if no options are set
	if opts != nil && !opts.AutoSubscribe {
		return false
	}
	return true
}

func (r *Room) createJoinResponseLocked(participant types.LocalParticipant, iceServers []*livekit.ICEServer) *livekit.JoinResponse {
	// gather other participants and send join response
	otherParticipants := make([]*livekit.ParticipantInfo, 0, len(r.participants))
	for _, p := range r.participants {
		if p.ID() != participant.ID() && !p.Hidden() {
			otherParticipants = append(otherParticipants, p.ToProto())
		}
	}

	return &livekit.JoinResponse{
		Room:              r.protoRoom,
		Participant:       participant.ToProto(),
		OtherParticipants: otherParticipants,
		ServerVersion:     r.serverInfo.Version,
		ServerRegion:      r.serverInfo.Region,
		IceServers:        iceServers,
		// indicates both server and client support subscriber as primary
		SubscriberPrimary:   participant.SubscriberAsPrimary(),
		ClientConfiguration: participant.GetClientConfiguration(),
		// sane defaults for ping interval & timeout
		PingInterval: 10,
		PingTimeout:  20,
		ServerInfo:   r.serverInfo,
	}
}

// a ParticipantImpl in the room added a new track, subscribe other participants to it
func (r *Room) onTrackPublished(participant types.LocalParticipant, track types.MediaTrack) {
	// publish participant update, since track state is changed
	r.broadcastParticipantState(participant, broadcastOptions{skipSource: true})

	r.lock.RLock()
	// subscribe all existing participants to this MediaTrack
	for _, existingParticipant := range r.participants {
		if existingParticipant == participant {
			// skip publishing participant
			continue
		}
		if existingParticipant.State() != livekit.ParticipantInfo_ACTIVE {
			// not fully joined. don't subscribe yet
			continue
		}
		if !r.autoSubscribe(existingParticipant) {
			continue
		}

		r.Logger.Debugw("subscribing to new track",
			"participant", existingParticipant.Identity(),
			"pID", existingParticipant.ID(),
			"publisher", participant.Identity(),
			"publisherID", participant.ID(),
			"trackID", track.ID())
		existingParticipant.SubscribeToTrack(track.ID())
	}
	onParticipantChanged := r.onParticipantChanged
	r.lock.RUnlock()

	if onParticipantChanged != nil {
		onParticipantChanged(participant)
	}

	r.trackManager.AddTrack(track, participant.Identity(), participant.ID())

	// auto track egress
	if r.internal != nil && r.internal.TrackEgress != nil {
		if err := StartTrackEgress(
			context.Background(),
			r.egressLauncher,
			r.telemetry,
			r.internal.TrackEgress,
			track,
			r.Name(),
			r.ID(),
		); err != nil {
			r.Logger.Errorw("failed to launch track egress", err)
		}
	}
}

func (r *Room) onTrackUpdated(p types.LocalParticipant, _ types.MediaTrack) {
	// send track updates to everyone, especially if track was updated by admin
	r.broadcastParticipantState(p, broadcastOptions{})
	if r.onParticipantChanged != nil {
		r.onParticipantChanged(p)
	}
}

func (r *Room) onTrackUnpublished(p types.LocalParticipant, track types.MediaTrack) {
	r.trackManager.RemoveTrack(track)
	if !p.IsClosed() {
		r.broadcastParticipantState(p, broadcastOptions{skipSource: true})
	}
	if r.onParticipantChanged != nil {
		r.onParticipantChanged(p)
	}
}

func (r *Room) onParticipantUpdate(p types.LocalParticipant) {
	// immediately notify when permissions or metadata changed
	r.broadcastParticipantState(p, broadcastOptions{immediate: true})
	if r.onParticipantChanged != nil {
		r.onParticipantChanged(p)
	}
}

func (r *Room) onDataPacket(source types.LocalParticipant, dp *livekit.DataPacket) {
	r.sendDataPacketToRelays(dp)
	BroadcastDataPacketForRoom(r, source, dp, r.Logger)
}

func (r *Room) subscribeToExistingTracks(p types.LocalParticipant) {
	r.lock.RLock()
	shouldSubscribe := r.autoSubscribe(p)
	r.lock.RUnlock()
	if !shouldSubscribe {
		return
	}

	var trackIDs []livekit.TrackID
	for _, op := range r.GetParticipants() {
		if p.ID() == op.ID() {
			// don't send to itself
			continue
		}

		// subscribe to all
		for _, track := range op.GetPublishedTracks() {
			trackIDs = append(trackIDs, track.ID())
			p.SubscribeToTrack(track.ID())
		}
	}
	if len(trackIDs) > 0 {
		r.Logger.Debugw("subscribed participant to existing tracks", "trackID", trackIDs)
	}
}

// broadcast an update about participant p
func (r *Room) broadcastParticipantState(p types.LocalParticipant, opts broadcastOptions) {
	if _, ok := p.(*RelayedParticipantImpl); ok {
		return
	}

	pi := p.ToProto()

	if p.Hidden() {
		if !opts.skipSource {
			// send update only to hidden participant
			err := p.SendParticipantUpdate([]*livekit.ParticipantInfo{pi})
			if err != nil {
				r.Logger.Errorw("could not send update to participant", err,
					"participant", p.Identity(), "pID", p.ID())
			}
		}
		return
	}

	updates := r.pushAndDequeueUpdates(pi, opts.immediate)
	r.sendParticipantUpdates(updates)
	r.sendParticipantUpdatesToRelays(updates)
}

func (r *Room) sendParticipantUpdates(updates []*livekit.ParticipantInfo) {
	if len(updates) == 0 {
		return
	}

	for _, op := range r.GetParticipants() {
		err := op.SendParticipantUpdate(updates)
		if err != nil {
			r.Logger.Errorw("could not send update to participant", err,
				"participant", op.Identity(), "pID", op.ID())
		}
	}
}

func (r *Room) getUpdatesPayloadForRelay(updates []*livekit.ParticipantInfo) ([]byte, error) {
	updatesForRelay := make([]*livekit.ParticipantInfo, 0, len(updates))
	for _, update := range updates {
		if _, ok := r.relayedParticipants[livekit.ParticipantIdentity(update.Identity)]; ok {
			continue
		}
		updatesForRelay = append(updatesForRelay, update)
	}

	if updatesPayload, err := json.Marshal(relayMessage{Updates: updatesForRelay}); err != nil {
		return nil, fmt.Errorf("could not marshal participant updates: %w", err)
	} else {
		return updatesPayload, nil
	}
}

func (r *Room) sendParticipantUpdatesToRelays(updates []*livekit.ParticipantInfo) {
	if len(updates) == 0 {
		return
	}

	if updatesForRelay, err := r.getUpdatesPayloadForRelay(updates); err != nil {
		r.Logger.Errorw("could not create participant update for relay", err)
	} else if len(updatesForRelay) > 0 {
		r.outRelayCollection.ForEach(func(relay relay.Relay) {
			if err := relay.SendMessage(updatesForRelay); err != nil {
				r.Logger.Errorw("could not send participant updates to relay", err)
			}
		})
	}
}

func (r *Room) sendDataPacketToRelays(dp *livekit.DataPacket) {
	dpData, err := proto.Marshal(dp)
	if err != nil {
		r.Logger.Errorw("could not marshal data packet", err)
		return
	}
	if messageData, err := json.Marshal(relayMessage{DataPacket: dpData}); err != nil {
		r.Logger.Errorw("could not create data packet message for relay", err)
	} else {
		r.outRelayCollection.ForEach(func(relay relay.Relay) {
			if err := relay.SendMessage(messageData); err != nil {
				r.Logger.Errorw("could not send data packet to relay", err, "relayId", relay.ID())
			}
		})
	}
}

// for protocol 2, send all active speakers
func (r *Room) sendActiveSpeakers(speakers []*livekit.SpeakerInfo) {
	dp := &livekit.DataPacket{
		Kind: livekit.DataPacket_LOSSY,
		Value: &livekit.DataPacket_Speaker{
			Speaker: &livekit.ActiveSpeakerUpdate{
				Speakers: speakers,
			},
		},
	}

	var dpData []byte
	for _, p := range r.GetParticipants() {
		if p.ProtocolVersion().HandlesDataPackets() && !p.ProtocolVersion().SupportsSpeakerChanged() {
			if dpData == nil {
				var err error
				dpData, err = proto.Marshal(dp)
				if err != nil {
					r.Logger.Errorw("failed to marshal ActiveSpeaker data packet", err)
					return
				}
			}
			_ = p.SendDataPacket(dp, dpData)
		}
	}
}

// for protocol 3, send only changed updates
func (r *Room) sendSpeakerChanges(speakers []*livekit.SpeakerInfo) {
	for _, p := range r.GetParticipants() {
		if p.ProtocolVersion().SupportsSpeakerChanged() {
			_ = p.SendSpeakerUpdate(speakers, false)
		}
	}
}

// push a participant update for batched broadcast, optionally returning immediate updates to broadcast.
// it handles the following scenarios
// * subscriber-only updates will be queued for batch updates
// * publisher & immediate updates will be returned without queuing
// * when the SID changes, it will return both updates, with the earlier participant set to disconnected
func (r *Room) pushAndDequeueUpdates(pi *livekit.ParticipantInfo, isImmediate bool) []*livekit.ParticipantInfo {
	r.batchedUpdatesMu.Lock()
	defer r.batchedUpdatesMu.Unlock()

	var updates []*livekit.ParticipantInfo
	identity := livekit.ParticipantIdentity(pi.Identity)
	existing := r.batchedUpdates[identity]
	shouldSend := isImmediate || pi.IsPublisher

	if existing != nil {
		if pi.Sid == existing.Sid {
			// same participant session
			if pi.Version < existing.Version {
				// out of order update
				return nil
			}
		} else {
			// different participant sessions
			if existing.JoinedAt < pi.JoinedAt {
				// existing is older, synthesize a DISCONNECT for older and
				// send immediately along with newer session to signal switch
				shouldSend = true
				existing.State = livekit.ParticipantInfo_DISCONNECTED
				updates = append(updates, existing)
			} else {
				// older session update, newer session has already become active, so nothing to do
				return nil
			}
		}
	}

	if shouldSend {
		// include any queued update, and return
		delete(r.batchedUpdates, identity)
		updates = append(updates, pi)
	} else {
		// enqueue for batch
		r.batchedUpdates[identity] = pi
	}

	return updates
}

func (r *Room) subscriberBroadcastWorker() {
	ticker := time.NewTicker(subscriberUpdateInterval)
	defer ticker.Stop()

	for !r.IsClosed() {
		select {
		case <-r.closed:
			return
		case <-ticker.C:
			r.batchedUpdatesMu.Lock()
			updatesMap := r.batchedUpdates
			r.batchedUpdates = make(map[livekit.ParticipantIdentity]*livekit.ParticipantInfo)
			r.batchedUpdatesMu.Unlock()

			if len(updatesMap) == 0 {
				continue
			}

			updates := make([]*livekit.ParticipantInfo, 0, len(updatesMap))
			for _, pi := range updatesMap {
				updates = append(updates, pi)
			}
			r.sendParticipantUpdates(updates)
		}
	}
}

func (r *Room) audioUpdateWorker() {
	lastActiveMap := make(map[livekit.ParticipantID]*livekit.SpeakerInfo)
	for {
		if r.IsClosed() {
			return
		}

		activeSpeakers := r.GetActiveSpeakers()
		changedSpeakers := make([]*livekit.SpeakerInfo, 0, len(activeSpeakers))
		nextActiveMap := make(map[livekit.ParticipantID]*livekit.SpeakerInfo, len(activeSpeakers))
		for _, speaker := range activeSpeakers {
			prev := lastActiveMap[livekit.ParticipantID(speaker.Sid)]
			if prev == nil || prev.Level != speaker.Level {
				changedSpeakers = append(changedSpeakers, speaker)
			}
			nextActiveMap[livekit.ParticipantID(speaker.Sid)] = speaker
		}

		// changedSpeakers need to include previous speakers that are no longer speaking
		for sid, speaker := range lastActiveMap {
			if nextActiveMap[sid] == nil {
				inactiveSpeaker := proto.Clone(speaker).(*livekit.SpeakerInfo)
				inactiveSpeaker.Level = 0
				inactiveSpeaker.Active = false
				changedSpeakers = append(changedSpeakers, inactiveSpeaker)
			}
		}

		// see if an update is needed
		if len(changedSpeakers) > 0 {
			r.sendActiveSpeakers(activeSpeakers)
			r.sendSpeakerChanges(changedSpeakers)
		}

		lastActiveMap = nextActiveMap

		time.Sleep(time.Duration(r.audioConfig.UpdateInterval) * time.Millisecond)
	}
}

func (r *Room) connectionQualityWorker() {
	ticker := time.NewTicker(connectionquality.UpdateInterval)
	defer ticker.Stop()

	prevConnectionInfos := make(map[livekit.ParticipantID]*livekit.ConnectionQualityInfo)
	// send updates to only users that are subscribed to each other
	for !r.IsClosed() {
		<-ticker.C

		participants := r.GetParticipants()
		nowConnectionInfos := make(map[livekit.ParticipantID]*livekit.ConnectionQualityInfo, len(participants))

		for _, p := range participants {
			if p.State() != livekit.ParticipantInfo_ACTIVE {
				continue
			}

			if q := p.GetConnectionQuality(); q != nil {
				nowConnectionInfos[p.ID()] = q
			}
		}

		// send an update if there is a change
		//   - new participant
		//   - quality change
		// NOTE: participant leaving is explicitly omitted as `leave` signal notifies that a participant is not in the room anymore
		sendUpdate := false
		for _, p := range participants {
			pID := p.ID()
			prevInfo, prevOk := prevConnectionInfos[pID]
			nowInfo, nowOk := nowConnectionInfos[pID]
			if !nowOk {
				// participant is not ACTIVE any more
				continue
			}
			if !prevOk || nowInfo.Quality != prevInfo.Quality {
				// new entrant OR change in quality
				sendUpdate = true
				break
			}
		}

		if !sendUpdate {
			prevConnectionInfos = nowConnectionInfos
			continue
		}

		maybeAddToUpdate := func(pID livekit.ParticipantID, update *livekit.ConnectionQualityUpdate) {
			if nowInfo, nowOk := nowConnectionInfos[pID]; nowOk {
				update.Updates = append(update.Updates, nowInfo)
			}
		}

		for _, op := range participants {
			if !op.ProtocolVersion().SupportsConnectionQuality() || op.State() != livekit.ParticipantInfo_ACTIVE {
				continue
			}
			update := &livekit.ConnectionQualityUpdate{}

			// send to user itself
			maybeAddToUpdate(op.ID(), update)

			// add connection quality of other participants its subscribed to
			for _, sid := range op.GetSubscribedParticipants() {
				maybeAddToUpdate(sid, update)
			}
			if len(update.Updates) == 0 {
				// no change
				continue
			}
			if err := op.SendConnectionQualityUpdate(update); err != nil {
				r.Logger.Warnw("could not send connection quality update", err,
					"participant", op.Identity())
			}
		}

		prevConnectionInfos = nowConnectionInfos
	}
}

func (r *Room) DebugInfo() map[string]interface{} {
	info := map[string]interface{}{
		"Name":      r.protoRoom.Name,
		"Sid":       r.protoRoom.Sid,
		"CreatedAt": r.protoRoom.CreationTime,
	}

	participants := r.GetParticipants()
	participantInfo := make(map[string]interface{})
	for _, p := range participants {
		participantInfo[string(p.Identity())] = p.DebugInfo()
	}
	info["Participants"] = participantInfo

	outRelaysInfo := make(map[string]interface{})
	i := 0
	r.outRelayCollection.ForEach(func(relay relay.Relay) {
		outRelaysInfo[strconv.Itoa(i)] = relay.DebugInfo()
		i++
	})
	info["OutRelays"] = outRelaysInfo

	return info
}

func (r *Room) GetOutRelayCollection() *relay.Collection {
	return r.outRelayCollection
}

func BroadcastDataPacketForRoom(r types.Room, source types.LocalParticipant, dp *livekit.DataPacket, logger logger.Logger) {
	dest := dp.GetUser().GetDestinationSids()
	var dpData []byte

	participants := r.GetLocalParticipants()
	capacity := len(dest)
	if capacity == 0 {
		capacity = len(participants)
	}
	destParticipants := make([]types.LocalParticipant, 0, capacity)

	for _, op := range participants {
		if op.State() != livekit.ParticipantInfo_ACTIVE {
			continue
		}
		if source != nil && op.ID() == source.ID() {
			continue
		}
		if len(dest) > 0 {
			found := false
			for _, dID := range dest {
				if op.ID() == livekit.ParticipantID(dID) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if dpData == nil {
			var err error
			dpData, err = proto.Marshal(dp)
			if err != nil {
				logger.Errorw("failed to marshal data packet", err)
				return
			}
		}
		destParticipants = append(destParticipants, op)
	}

	utils.ParallelExec(destParticipants, dataForwardLoadBalanceThreshold, 1, func(op types.LocalParticipant) {
		err := op.SendDataPacket(dp, dpData)
		if err != nil && !errors.Is(err, io.ErrClosedPipe) {
			op.GetLogger().Infow("send data packet error", "error", err)
		}
	})
}
