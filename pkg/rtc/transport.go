// Copyright 2023 LiveKit, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rtc

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/bep/debounce"
	"github.com/pion/dtls/v2/pkg/crypto/elliptic"
	"github.com/pion/interceptor"
	"github.com/pion/interceptor/pkg/cc"
	"github.com/pion/interceptor/pkg/gcc"
	"github.com/pion/interceptor/pkg/twcc"
	"github.com/pion/rtcp"
	"github.com/pion/sctp"
	"github.com/pion/sdp/v3"
	"github.com/pion/webrtc/v3"
	"github.com/pkg/errors"
	"go.uber.org/atomic"

	"github.com/livekit/livekit-server/pkg/config"
	"github.com/livekit/livekit-server/pkg/rtc/types"
	"github.com/livekit/livekit-server/pkg/sfu/pacer"
	"github.com/livekit/livekit-server/pkg/sfu/rtpextension"
	"github.com/livekit/livekit-server/pkg/sfu/streamallocator"
	"github.com/livekit/livekit-server/pkg/telemetry/prometheus"
	"github.com/livekit/livekit-server/pkg/utils"
	sutils "github.com/livekit/livekit-server/pkg/utils"
	"github.com/livekit/protocol/livekit"
	"github.com/livekit/protocol/logger"
	"github.com/livekit/protocol/logger/pionlogger"
	lksdp "github.com/livekit/protocol/sdp"
)

const (
	LossyDataChannel    = "_lossy"
	ReliableDataChannel = "_reliable"

	negotiationFrequency       = 150 * time.Millisecond
	negotiationFailedTimeout   = 15 * time.Second
	dtlsRetransmissionInterval = 100 * time.Millisecond

	iceDisconnectedTimeout = 10 * time.Second                          // compatible for ice-lite with firefox client
	iceFailedTimeout       = 5 * time.Second                           // time between disconnected and failed
	iceFailedTimeoutTotal  = iceFailedTimeout + iceDisconnectedTimeout // total time between connecting and failure
	iceKeepaliveInterval   = 2 * time.Second                           // pion's default

	minTcpICEConnectTimeout = 5 * time.Second
	maxTcpICEConnectTimeout = 12 * time.Second // js-sdk has a default 15s timeout for first connection, let server detect failure earlier before that

	minConnectTimeoutAfterICE = 10 * time.Second
	maxConnectTimeoutAfterICE = 20 * time.Second // max duration for waiting pc to connect after ICE is connected

	maxICECandidates = 20

	shortConnectionThreshold = 90 * time.Second
)

var (
	ErrIceRestartWithoutLocalSDP        = errors.New("ICE restart without local SDP settled")
	ErrIceRestartOnClosedPeerConnection = errors.New("ICE restart on closed peer connection")
	ErrNoTransceiver                    = errors.New("no transceiver")
	ErrNoSender                         = errors.New("no sender")
	ErrNoICECandidateHandler            = errors.New("no ICE candidate handler")
	ErrNoOfferHandler                   = errors.New("no offer handler")
	ErrNoAnswerHandler                  = errors.New("no answer handler")
	ErrMidNotFound                      = errors.New("mid not found")
)

// -------------------------------------------------------------------------

type signal int

const (
	signalICEGatheringComplete signal = iota
	signalLocalICECandidate
	signalRemoteICECandidate
	signalSendOffer
	signalRemoteDescriptionReceived
	signalICERestart
)

func (s signal) String() string {
	switch s {
	case signalICEGatheringComplete:
		return "ICE_GATHERING_COMPLETE"
	case signalLocalICECandidate:
		return "LOCAL_ICE_CANDIDATE"
	case signalRemoteICECandidate:
		return "REMOTE_ICE_CANDIDATE"
	case signalSendOffer:
		return "SEND_OFFER"
	case signalRemoteDescriptionReceived:
		return "REMOTE_DESCRIPTION_RECEIVED"
	case signalICERestart:
		return "ICE_RESTART"
	default:
		return fmt.Sprintf("%d", int(s))
	}
}

// -------------------------------------------------------

type event struct {
	signal signal
	data   interface{}
}

func (e event) String() string {
	return fmt.Sprintf("PCTransport:Event{signal: %s, data: %+v}", e.signal, e.data)
}

// -------------------------------------------------------

type NegotiationState int

const (
	NegotiationStateNone NegotiationState = iota
	// waiting for remote description
	NegotiationStateRemote
	// need to Negotiate again
	NegotiationStateRetry
)

func (n NegotiationState) String() string {
	switch n {
	case NegotiationStateNone:
		return "NONE"
	case NegotiationStateRemote:
		return "WAITING_FOR_REMOTE"
	case NegotiationStateRetry:
		return "RETRY"
	default:
		return fmt.Sprintf("%d", int(n))
	}
}

// -------------------------------------------------------

type SimulcastTrackInfo struct {
	Mid string
	Rid string
}

type trackDescription struct {
	mid    string
	sender *webrtc.RTPSender
}

// PCTransport is a wrapper around PeerConnection, with some helper methods
type PCTransport struct {
	params TransportParams
	pc     *webrtc.PeerConnection
	me     *webrtc.MediaEngine

	lock sync.RWMutex

	reliableDC       *webrtc.DataChannel
	reliableDCOpened bool
	lossyDC          *webrtc.DataChannel
	lossyDCOpened    bool
	onDataPacket     func(kind livekit.DataPacket_Kind, data []byte)

	iceStartedAt               time.Time
	iceConnectedAt             time.Time
	firstConnectedAt           time.Time
	connectedAt                time.Time
	tcpICETimer                *time.Timer
	connectAfterICETimer       *time.Timer // timer to wait for pc to connect after ice connected
	resetShortConnOnICERestart atomic.Bool
	signalingRTT               atomic.Uint32 // milliseconds

	onFullyEstablished func()

	debouncedNegotiate func(func())
	debouncePending    bool

	onICECandidate            func(c *webrtc.ICECandidate) error
	onOffer                   func(offer webrtc.SessionDescription) error
	onAnswer                  func(answer webrtc.SessionDescription) error
	onInitialConnected        func()
	onFailed                  func(isShortLived bool)
	onNegotiationStateChanged func(state NegotiationState)
	onNegotiationFailed       func()

	// stream allocator for subscriber PC
	streamAllocator *streamallocator.StreamAllocator

	// only for subscriber PC
	pacer pacer.Pacer

	previousAnswer *webrtc.SessionDescription
	// track id -> description map in previous offer sdp
	previousTrackDescription map[string]*trackDescription
	canReuseTransceiver      bool

	preferTCP atomic.Bool
	isClosed  atomic.Bool

	eventsQueue *utils.OpsQueue

	// the following should be accessed only in event processing go routine
	cacheLocalCandidates      bool
	cachedLocalCandidates     []*webrtc.ICECandidate
	pendingRemoteCandidates   []*webrtc.ICECandidateInit
	restartAfterGathering     bool
	restartAtNextOffer        bool
	negotiationState          NegotiationState
	negotiateCounter          atomic.Int32
	signalStateCheckTimer     *time.Timer
	currentOfferIceCredential string // ice user:pwd, for publish side ice restart checking
	pendingRestartIceOffer    *webrtc.SessionDescription

	connectionDetails *types.ICEConnectionDetails
}

type TransportParams struct {
	ParticipantID                livekit.ParticipantID
	ParticipantIdentity          livekit.ParticipantIdentity
	ProtocolVersion              types.ProtocolVersion
	Config                       *WebRTCConfig
	DirectionConfig              DirectionConfig
	CongestionControlConfig      config.CongestionControlConfig
	EnabledCodecs                []*livekit.Codec
	Logger                       logger.Logger
	Transport                    livekit.SignalTarget
	SimTracks                    map[uint32]SimulcastTrackInfo
	ClientInfo                   ClientInfo
	IsOfferer                    bool
	IsSendSide                   bool
	AllowPlayoutDelay            bool
	DataChannelMaxBufferedAmount uint64
}

func newPeerConnection(params TransportParams, onBandwidthEstimator func(estimator cc.BandwidthEstimator)) (*webrtc.PeerConnection, *webrtc.MediaEngine, error) {
	directionConfig := params.DirectionConfig

	if params.AllowPlayoutDelay {
		directionConfig.RTPHeaderExtension.Video = append(directionConfig.RTPHeaderExtension.Video, rtpextension.PlayoutDelayURI)
	}

	// Some of the browser clients do not handle H.264 High Profile in signalling properly.
	// They still decode if the actual stream is H.264 High Profile, but do not handle it well in signalling.
	// So, disable H.264 High Profile for SUBSCRIBER peer connection to ensure it is not offered.
	me, err := createMediaEngine(params.EnabledCodecs, directionConfig, params.IsOfferer)
	if err != nil {
		return nil, nil, err
	}

	se := params.Config.SettingEngine
	se.DisableMediaEngineCopy(true)

	// Change elliptic curve to improve connectivity
	// https://github.com/pion/dtls/pull/474
	se.SetDTLSEllipticCurves(elliptic.X25519, elliptic.P384, elliptic.P256)

	//
	// Disable SRTP replay protection (https://datatracker.ietf.org/doc/html/rfc3711#page-15).
	// Needed due to lack of RTX stream support in Pion.
	//
	// When clients probe for bandwidth, there are several possible approaches
	//   1. Use padding packet (Chrome uses this)
	//   2. Use an older packet (Firefox uses this)
	// Typically, these are sent over the RTX stream and hence SRTP replay protection will not
	// trigger. As Pion does not support RTX, when firefox uses older packet for probing, they
	// trigger the replay protection.
	//
	// That results in two issues
	//   - Firefox bandwidth probing is not successful
	//   - Pion runs out of read buffer capacity - this potentially looks like a Pion issue
	//
	// NOTE: It is not required to disable RTCP replay protection, but doing it to be symmetric.
	//
	se.DisableSRTPReplayProtection(true)
	se.DisableSRTCPReplayProtection(true)
	if !params.ProtocolVersion.SupportsICELite() {
		se.SetLite(false)
	}
	se.SetDTLSRetransmissionInterval(dtlsRetransmissionInterval)
	se.SetICETimeouts(iceDisconnectedTimeout, iceFailedTimeout, iceKeepaliveInterval)

	// if client don't support prflx over relay, we should not expose private address to it, use single external ip as host candidate
	if !params.ClientInfo.SupportPrflxOverRelay() && len(params.Config.NAT1To1IPs) > 0 {
		var nat1to1Ips []string
		var includeIps []string
		for _, mapping := range params.Config.NAT1To1IPs {
			if ips := strings.Split(mapping, "/"); len(ips) == 2 {
				if ips[0] != ips[1] {
					nat1to1Ips = append(nat1to1Ips, mapping)
					includeIps = append(includeIps, ips[1])
				}
			}
		}
		if len(nat1to1Ips) > 0 {
			params.Logger.Infow("client doesn't support prflx over relay, use external ip only as host candidate", "ips", nat1to1Ips)
			se.SetNAT1To1IPs(nat1to1Ips, webrtc.ICECandidateTypeHost)
			se.SetIPFilter(func(ip net.IP) bool {
				if ip.To4() == nil {
					return true
				}
				ipstr := ip.String()
				for _, inc := range includeIps {
					if inc == ipstr {
						return true
					}
				}
				return false
			})
		}
	}

	lf := pionlogger.NewLoggerFactory(params.Logger)
	if lf != nil {
		se.LoggerFactory = lf
	}

	ir := &interceptor.Registry{}
	if params.IsSendSide {
		if params.CongestionControlConfig.UseSendSideBWE {
			gf, err := cc.NewInterceptor(func() (cc.BandwidthEstimator, error) {
				return gcc.NewSendSideBWE(
					gcc.SendSideBWEInitialBitrate(1*1000*1000),
					gcc.SendSideBWEPacer(gcc.NewNoOpPacer()),
				)
			})
			if err == nil {
				gf.OnNewPeerConnection(func(id string, estimator cc.BandwidthEstimator) {
					if onBandwidthEstimator != nil {
						onBandwidthEstimator(estimator)
					}
				})
				ir.Add(gf)

				tf, err := twcc.NewHeaderExtensionInterceptor()
				if err == nil {
					ir.Add(tf)
				}
			}
		}
	}
	if len(params.SimTracks) > 0 {
		f, err := NewUnhandleSimulcastInterceptorFactory(UnhandleSimulcastTracks(params.SimTracks))
		if err != nil {
			params.Logger.Errorw("NewUnhandleSimulcastInterceptorFactory failed", err)
		} else {
			ir.Add(f)
		}
	}
	api := webrtc.NewAPI(
		webrtc.WithMediaEngine(me),
		webrtc.WithSettingEngine(se),
		webrtc.WithInterceptorRegistry(ir),
	)
	pc, err := api.NewPeerConnection(params.Config.Configuration)
	return pc, me, err
}

func NewPCTransport(params TransportParams) (*PCTransport, error) {
	if params.Logger == nil {
		params.Logger = logger.GetLogger()
	}
	t := &PCTransport{
		params:                   params,
		debouncedNegotiate:       debounce.New(negotiationFrequency),
		negotiationState:         NegotiationStateNone,
		eventsQueue:              utils.NewOpsQueue("transport", 64, false),
		previousTrackDescription: make(map[string]*trackDescription),
		canReuseTransceiver:      true,
		connectionDetails:        types.NewICEConnectionDetails(params.Transport, params.Logger),
	}
	if params.IsSendSide {
		t.streamAllocator = streamallocator.NewStreamAllocator(streamallocator.StreamAllocatorParams{
			Config: params.CongestionControlConfig,
			Logger: params.Logger.WithComponent(sutils.ComponentCongestionControl),
		})
		t.streamAllocator.Start()
		t.pacer = pacer.NewPassThrough(params.Logger)
	}

	if err := t.createPeerConnection(); err != nil {
		return nil, err
	}

	t.eventsQueue.Start()

	return t, nil
}

func (t *PCTransport) createPeerConnection() error {
	var bwe cc.BandwidthEstimator
	pc, me, err := newPeerConnection(t.params, func(estimator cc.BandwidthEstimator) {
		bwe = estimator
	})
	if err != nil {
		return err
	}

	t.pc = pc
	t.pc.OnICEGatheringStateChange(t.onICEGatheringStateChange)
	t.pc.OnICEConnectionStateChange(t.onICEConnectionStateChange)
	t.pc.OnICECandidate(t.onICECandidateTrickle)

	t.pc.OnConnectionStateChange(t.onPeerConnectionStateChange)

	t.pc.OnDataChannel(t.onDataChannel)

	t.me = me

	if bwe != nil && t.streamAllocator != nil {
		t.streamAllocator.SetBandwidthEstimator(bwe)
	}

	return nil
}

func (t *PCTransport) GetPacer() pacer.Pacer {
	return t.pacer
}

func (t *PCTransport) SetSignalingRTT(rtt uint32) {
	t.signalingRTT.Store(rtt)
}

func (t *PCTransport) setICEStartedAt(at time.Time) {
	t.lock.Lock()
	if t.iceStartedAt.IsZero() {
		t.iceStartedAt = at

		// set failure timer for tcp ice connection based on signaling RTT
		if t.preferTCP.Load() {
			signalingRTT := t.signalingRTT.Load()
			if signalingRTT < 1000 {
				tcpICETimeout := time.Duration(signalingRTT*8) * time.Millisecond
				if tcpICETimeout < minTcpICEConnectTimeout {
					tcpICETimeout = minTcpICEConnectTimeout
				} else if tcpICETimeout > maxTcpICEConnectTimeout {
					tcpICETimeout = maxTcpICEConnectTimeout
				}
				t.params.Logger.Debugw("set TCP ICE connect timer", "timeout", tcpICETimeout, "signalRTT", signalingRTT)
				t.tcpICETimer = time.AfterFunc(tcpICETimeout, func() {
					if t.pc.ICEConnectionState() == webrtc.ICEConnectionStateChecking {
						t.params.Logger.Infow("TCP ICE connect timeout", "timeout", tcpICETimeout, "signalRTT", signalingRTT)
						t.handleConnectionFailed(true)
					}
				})
			}
		}
	}
	t.lock.Unlock()
}

func (t *PCTransport) setICEConnectedAt(at time.Time) {
	t.lock.Lock()
	if t.iceConnectedAt.IsZero() {
		//
		// Record initial connection time.
		// This prevents reset of connected at time if ICE goes `Connected` -> `Disconnected` -> `Connected`.
		//
		t.iceConnectedAt = at

		// set failure timer for dtls handshake
		iceDuration := at.Sub(t.iceStartedAt)
		connTimeoutAfterICE := minConnectTimeoutAfterICE
		if connTimeoutAfterICE < 3*iceDuration {
			connTimeoutAfterICE = 3 * iceDuration
		}
		if connTimeoutAfterICE > maxConnectTimeoutAfterICE {
			connTimeoutAfterICE = maxConnectTimeoutAfterICE
		}
		t.params.Logger.Debugw("setting connection timer after ICE connected", "timeout", connTimeoutAfterICE, "iceDuration", iceDuration)
		t.connectAfterICETimer = time.AfterFunc(connTimeoutAfterICE, func() {
			state := t.pc.ConnectionState()
			// if pc is still checking or connected but not fully established after timeout, then fire connection fail
			if state != webrtc.PeerConnectionStateClosed && state != webrtc.PeerConnectionStateFailed && !t.isFullyEstablished() {
				t.params.Logger.Infow("connect timeout after ICE connected", "timeout", connTimeoutAfterICE, "iceDuration", iceDuration)
				t.handleConnectionFailed(false)
			}
		})

		// clear tcp ice connect timer
		if t.tcpICETimer != nil {
			t.tcpICETimer.Stop()
			t.tcpICETimer = nil
		}
	}
	t.lock.Unlock()
}

func (t *PCTransport) resetShortConn() {
	t.params.Logger.Infow("resetting short connection on ICE restart")
	t.lock.Lock()
	t.iceStartedAt = time.Time{}
	t.iceConnectedAt = time.Time{}
	t.connectedAt = time.Time{}
	if t.connectAfterICETimer != nil {
		t.connectAfterICETimer.Stop()
		t.connectAfterICETimer = nil
	}
	if t.tcpICETimer != nil {
		t.tcpICETimer.Stop()
		t.tcpICETimer = nil
	}
	t.lock.Unlock()
}

func (t *PCTransport) IsShortConnection(at time.Time) (bool, time.Duration) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	if t.iceConnectedAt.IsZero() {
		return false, 0
	}

	duration := at.Sub(t.iceConnectedAt)
	return duration < shortConnectionThreshold, duration
}

func (t *PCTransport) getSelectedPair() (*webrtc.ICECandidatePair, error) {
	s := t.pc.SCTP()
	if s == nil {
		return nil, errors.New("no SCTP")
	}

	dtlsTransport := s.Transport()
	if dtlsTransport == nil {
		return nil, errors.New("no DTLS transport")
	}

	iceTransport := dtlsTransport.ICETransport()
	if iceTransport == nil {
		return nil, errors.New("no ICE transport")
	}

	pair, err := iceTransport.GetSelectedCandidatePair()
	if err != nil {
		return nil, err
	}

	if pair == nil {
		return nil, errors.New("no selected pair")
	}

	return pair, err
}

func (t *PCTransport) setConnectedAt(at time.Time) bool {
	t.lock.Lock()
	t.connectedAt = at
	if !t.firstConnectedAt.IsZero() {
		t.lock.Unlock()
		return false
	}

	t.firstConnectedAt = at
	prometheus.ServiceOperationCounter.WithLabelValues("peer_connection", "success", "").Add(1)
	t.lock.Unlock()
	return true
}

func (t *PCTransport) onICEGatheringStateChange(state webrtc.ICEGathererState) {
	t.params.Logger.Debugw("ice gathering state change", "state", state.String())
	if state != webrtc.ICEGathererStateComplete {
		return
	}

	t.postEvent(event{
		signal: signalICEGatheringComplete,
	})
}

func (t *PCTransport) onICECandidateTrickle(c *webrtc.ICECandidate) {
	t.postEvent(event{
		signal: signalLocalICECandidate,
		data:   c,
	})
}

func (t *PCTransport) handleConnectionFailed(forceShortConn bool) {
	isShort := forceShortConn
	if !isShort {
		var duration time.Duration
		isShort, duration = t.IsShortConnection(time.Now())
		if isShort {
			pair, err := t.getSelectedPair()
			if err != nil {
				t.params.Logger.Errorw("short ICE connection", err, "duration", duration)
			} else {
				t.params.Logger.Infow("short ICE connection", "pair", pair, "duration", duration)
			}
		}
	} else {
		t.params.Logger.Infow("force short ICE connection")
	}

	if onFailed := t.getOnFailed(); onFailed != nil {
		onFailed(isShort)
	}
}

func (t *PCTransport) onICEConnectionStateChange(state webrtc.ICEConnectionState) {
	t.params.Logger.Debugw("ice connection state change", "state", state.String())
	switch state {
	case webrtc.ICEConnectionStateConnected:
		t.setICEConnectedAt(time.Now())
		go func() {
			pair, err := t.getSelectedPair()
			if err != nil {
				t.params.Logger.Warnw("failed to get selected candidate pair", err)
				return
			}
			t.connectionDetails.SetSelectedPair(pair)
		}()

	case webrtc.ICEConnectionStateChecking:
		t.setICEStartedAt(time.Now())

	case webrtc.ICEConnectionStateDisconnected:
		t.params.Logger.Infow("ice connection state change unexpected", "state", state.String())
	case webrtc.ICEConnectionStateFailed:
		t.params.Logger.Debugw("ice connection state change unexpected", "state", state.String())
	}
}

func (t *PCTransport) onPeerConnectionStateChange(state webrtc.PeerConnectionState) {
	t.params.Logger.Debugw("peer connection state change", "state", state.String())
	switch state {
	case webrtc.PeerConnectionStateConnected:
		t.clearConnTimer()
		isInitialConnection := t.setConnectedAt(time.Now())
		if isInitialConnection {
			if onInitialConnected := t.getOnInitialConnected(); onInitialConnected != nil {
				onInitialConnected()
			}

			t.maybeNotifyFullyEstablished()
		}
	case webrtc.PeerConnectionStateFailed:
		t.params.Logger.Infow("peer connection failed")
		t.clearConnTimer()
		t.handleConnectionFailed(false)
	}
}

func (t *PCTransport) onDataChannel(dc *webrtc.DataChannel) {
	t.params.Logger.Debugw(dc.Label() + " data channel open")
	switch dc.Label() {
	case ReliableDataChannel:
		t.lock.Lock()
		t.reliableDC = dc
		t.reliableDCOpened = true
		t.lock.Unlock()
		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			if onDataPacket := t.getOnDataPacket(); onDataPacket != nil {
				onDataPacket(livekit.DataPacket_RELIABLE, msg.Data)
			}
		})

		t.maybeNotifyFullyEstablished()
	case LossyDataChannel:
		t.lock.Lock()
		t.lossyDC = dc
		t.lossyDCOpened = true
		t.lock.Unlock()
		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			if onDataPacket := t.getOnDataPacket(); onDataPacket != nil {
				onDataPacket(livekit.DataPacket_LOSSY, msg.Data)
			}
		})

		t.maybeNotifyFullyEstablished()
	default:
		t.params.Logger.Warnw("unsupported datachannel added", nil, "label", dc.Label())
	}
}

func (t *PCTransport) maybeNotifyFullyEstablished() {
	if t.isFullyEstablished() {
		if onFullyEstablished := t.getOnFullyEstablished(); onFullyEstablished != nil {
			onFullyEstablished()
		}
	}
}

func (t *PCTransport) isFullyEstablished() bool {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.reliableDCOpened && t.lossyDCOpened && !t.connectedAt.IsZero()
}

func (t *PCTransport) SetPreferTCP(preferTCP bool) {
	t.preferTCP.Store(preferTCP)
}

func (t *PCTransport) AddICECandidate(candidate webrtc.ICECandidateInit) {
	t.postEvent(event{
		signal: signalRemoteICECandidate,
		data:   &candidate,
	})
}

func (t *PCTransport) AddTrack(trackLocal webrtc.TrackLocal, params types.AddTrackParams) (sender *webrtc.RTPSender, transceiver *webrtc.RTPTransceiver, err error) {
	t.lock.Lock()
	canReuse := t.canReuseTransceiver
	td, ok := t.previousTrackDescription[trackLocal.ID()]
	if ok {
		delete(t.previousTrackDescription, trackLocal.ID())
	}
	t.lock.Unlock()

	// keep track use same mid after migration if possible
	if td != nil && td.sender != nil {
		for _, tr := range t.pc.GetTransceivers() {
			if tr.Mid() == td.mid {
				return td.sender, tr, tr.SetSender(td.sender, trackLocal)
			}
		}
	}

	// if never negotiated with client, can't reuse transceiver for track not subscribed before migration
	if !canReuse {
		return t.AddTransceiverFromTrack(trackLocal, params)
	}

	sender, err = t.pc.AddTrack(trackLocal)
	if err != nil {
		return
	}

	// as there is no way to get transceiver from sender, search
	for _, tr := range t.pc.GetTransceivers() {
		if tr.Sender() == sender {
			transceiver = tr
			break
		}
	}

	if transceiver == nil {
		err = ErrNoTransceiver
		return
	}

	configureAudioTransceiver(transceiver, params.Stereo, !params.Red || !t.params.ClientInfo.SupportsAudioRED())
	return
}

func (t *PCTransport) AddTransceiverFromTrack(trackLocal webrtc.TrackLocal, params types.AddTrackParams) (sender *webrtc.RTPSender, transceiver *webrtc.RTPTransceiver, err error) {
	transceiver, err = t.pc.AddTransceiverFromTrack(trackLocal)
	if err != nil {
		return
	}

	sender = transceiver.Sender()
	if sender == nil {
		err = ErrNoSender
		return
	}

	configureAudioTransceiver(transceiver, params.Stereo, !params.Red || !t.params.ClientInfo.SupportsAudioRED())

	return
}

func (t *PCTransport) RemoveTrack(sender *webrtc.RTPSender) error {
	return t.pc.RemoveTrack(sender)
}

func (t *PCTransport) GetMid(rtpReceiver *webrtc.RTPReceiver) string {
	for _, tr := range t.pc.GetTransceivers() {
		if tr.Receiver() == rtpReceiver {
			return tr.Mid()
		}
	}

	return ""
}

func (t *PCTransport) GetRTPReceiver(mid string) *webrtc.RTPReceiver {
	for _, tr := range t.pc.GetTransceivers() {
		if tr.Mid() == mid {
			return tr.Receiver()
		}
	}

	return nil
}

func (t *PCTransport) CreateDataChannel(label string, dci *webrtc.DataChannelInit) error {
	dc, err := t.pc.CreateDataChannel(label, dci)
	if err != nil {
		return err
	}

	dcReadyHandler := func() {
		t.lock.Lock()
		switch dc.Label() {
		case ReliableDataChannel:
			t.reliableDCOpened = true

		case LossyDataChannel:
			t.lossyDCOpened = true
		}
		t.lock.Unlock()
		t.params.Logger.Debugw(dc.Label() + " data channel open")

		t.maybeNotifyFullyEstablished()
	}

	dcCloseHandler := func() {
		t.params.Logger.Debugw(dc.Label() + " data channel close")
	}

	dcErrorHandler := func(err error) {
		if !errors.Is(err, sctp.ErrResetPacketInStateNotExist) && !errors.Is(err, sctp.ErrChunk) {
			t.params.Logger.Errorw(dc.Label()+" data channel error", err)
		}
	}

	t.lock.Lock()
	switch dc.Label() {
	case ReliableDataChannel:
		t.reliableDC = dc
		if t.params.DirectionConfig.StrictACKs {
			t.reliableDC.OnOpen(dcReadyHandler)
		} else {
			t.reliableDC.OnDial(dcReadyHandler)
		}
		t.reliableDC.OnClose(dcCloseHandler)
		t.reliableDC.OnError(dcErrorHandler)
	case LossyDataChannel:
		t.lossyDC = dc
		if t.params.DirectionConfig.StrictACKs {
			t.lossyDC.OnOpen(dcReadyHandler)
		} else {
			t.lossyDC.OnDial(dcReadyHandler)
		}
		t.lossyDC.OnClose(dcCloseHandler)
		t.lossyDC.OnError(dcErrorHandler)
	default:
		t.params.Logger.Errorw("unknown data channel label", nil, "label", dc.Label())
	}
	t.lock.Unlock()

	return nil
}

func (t *PCTransport) CreateDataChannelIfEmpty(dcLabel string, dci *webrtc.DataChannelInit) (label string, id uint16, existing bool, err error) {
	t.lock.RLock()
	var dc *webrtc.DataChannel
	switch dcLabel {
	case ReliableDataChannel:
		dc = t.reliableDC
	case LossyDataChannel:
		dc = t.lossyDC
	default:
		t.params.Logger.Errorw("unknown data channel label", nil, "label", label)
		err = errors.New("unknown data channel label")
	}
	t.lock.RUnlock()
	if err != nil {
		return
	}

	if dc != nil {
		return dc.Label(), *dc.ID(), true, nil
	}

	dc, err = t.pc.CreateDataChannel(dcLabel, dci)
	if err != nil {
		return
	}

	t.onDataChannel(dc)
	return dc.Label(), *dc.ID(), false, nil
}

// IsEstablished returns true if the PeerConnection has been established
func (t *PCTransport) IsEstablished() bool {
	return t.pc.ConnectionState() != webrtc.PeerConnectionStateNew
}

func (t *PCTransport) HasEverConnected() bool {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return !t.firstConnectedAt.IsZero()
}

func (t *PCTransport) GetICEConnectionDetails() *types.ICEConnectionDetails {
	return t.connectionDetails
}

func (t *PCTransport) WriteRTCP(pkts []rtcp.Packet) error {
	return t.pc.WriteRTCP(pkts)
}

func (t *PCTransport) SendDataPacket(dp *livekit.DataPacket, data []byte) error {
	var dc *webrtc.DataChannel
	t.lock.RLock()
	if dp.Kind == livekit.DataPacket_RELIABLE {
		dc = t.reliableDC
	} else {
		dc = t.lossyDC
	}
	t.lock.RUnlock()

	if dc == nil {
		return ErrDataChannelUnavailable
	}

	if t.pc.ConnectionState() == webrtc.PeerConnectionStateFailed {
		return ErrTransportFailure
	}

	if t.params.DataChannelMaxBufferedAmount > 0 && dc.BufferedAmount() > t.params.DataChannelMaxBufferedAmount {
		return ErrDataChannelBufferFull
	}

	return dc.Send(data)
}

func (t *PCTransport) Close() {
	if t.isClosed.Swap(true) {
		return
	}

	<-t.eventsQueue.Stop()
	t.clearSignalStateCheckTimer()

	if t.streamAllocator != nil {
		t.streamAllocator.Stop()
	}
	if t.pacer != nil {
		t.pacer.Stop()
	}

	_ = t.pc.Close()

	t.clearConnTimer()
}

func (t *PCTransport) clearConnTimer() {
	t.lock.Lock()
	defer t.lock.Unlock()
	if t.connectAfterICETimer != nil {
		t.connectAfterICETimer.Stop()
		t.connectAfterICETimer = nil
	}
	if t.tcpICETimer != nil {
		t.tcpICETimer.Stop()
		t.tcpICETimer = nil
	}
}

func (t *PCTransport) HandleRemoteDescription(sd webrtc.SessionDescription) {
	t.postEvent(event{
		signal: signalRemoteDescriptionReceived,
		data:   &sd,
	})
}

func (t *PCTransport) OnICECandidate(f func(c *webrtc.ICECandidate) error) {
	t.lock.Lock()
	t.onICECandidate = f
	t.lock.Unlock()
}

func (t *PCTransport) getOnICECandidate() func(c *webrtc.ICECandidate) error {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.onICECandidate
}

func (t *PCTransport) OnInitialConnected(f func()) {
	t.lock.Lock()
	t.onInitialConnected = f
	t.lock.Unlock()

	if f != nil {
		if t.pc.ConnectionState() == webrtc.PeerConnectionStateConnected {
			go f()
		}
	}
}

func (t *PCTransport) getOnInitialConnected() func() {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.onInitialConnected
}

func (t *PCTransport) OnFullyEstablished(f func()) {
	t.lock.Lock()
	t.onFullyEstablished = f
	t.lock.Unlock()
}

func (t *PCTransport) getOnFullyEstablished() func() {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.onFullyEstablished
}

func (t *PCTransport) OnFailed(f func(isShortLived bool)) {
	t.lock.Lock()
	t.onFailed = f
	t.lock.Unlock()
}

func (t *PCTransport) getOnFailed() func(isShortLived bool) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.onFailed
}

func (t *PCTransport) OnTrack(f func(track *webrtc.TrackRemote, rtpReceiver *webrtc.RTPReceiver)) {
	t.pc.OnTrack(f)
}

func (t *PCTransport) OnDataPacket(f func(kind livekit.DataPacket_Kind, data []byte)) {
	t.lock.Lock()
	t.onDataPacket = f
	t.lock.Unlock()
}

func (t *PCTransport) getOnDataPacket() func(kind livekit.DataPacket_Kind, data []byte) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.onDataPacket
}

// OnOffer is called when the PeerConnection starts negotiation and prepares an offer
func (t *PCTransport) OnOffer(f func(sd webrtc.SessionDescription) error) {
	t.lock.Lock()
	t.onOffer = f
	t.lock.Unlock()
}

func (t *PCTransport) getOnOffer() func(sd webrtc.SessionDescription) error {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.onOffer
}

func (t *PCTransport) OnAnswer(f func(sd webrtc.SessionDescription) error) {
	t.lock.Lock()
	t.onAnswer = f
	t.lock.Unlock()
}

func (t *PCTransport) getOnAnswer() func(sd webrtc.SessionDescription) error {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.onAnswer
}

func (t *PCTransport) OnNegotiationStateChanged(f func(state NegotiationState)) {
	t.lock.Lock()
	t.onNegotiationStateChanged = f
	t.lock.Unlock()
}

func (t *PCTransport) getOnNegotiationStateChanged() func(state NegotiationState) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.onNegotiationStateChanged
}

func (t *PCTransport) OnNegotiationFailed(f func()) {
	t.lock.Lock()
	t.onNegotiationFailed = f
	t.lock.Unlock()
}

func (t *PCTransport) getOnNegotiationFailed() func() {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.onNegotiationFailed
}

func (t *PCTransport) Negotiate(force bool) {
	if t.isClosed.Load() {
		return
	}

	if force {
		t.lock.Lock()
		t.debouncedNegotiate(func() {
			// no op to cancel pending negotiation
		})
		t.debouncePending = false
		t.lock.Unlock()

		t.postEvent(event{
			signal: signalSendOffer,
		})
	} else {
		t.lock.Lock()
		if !t.debouncePending {
			t.debouncedNegotiate(func() {
				t.lock.Lock()
				t.debouncePending = false
				t.lock.Unlock()

				t.postEvent(event{
					signal: signalSendOffer,
				})
			})
			t.debouncePending = true
		}
		t.lock.Unlock()
	}
}

func (t *PCTransport) ICERestart() error {
	if t.pc.ConnectionState() == webrtc.PeerConnectionStateClosed {
		t.params.Logger.Warnw("trying to restart ICE on closed peer connection", nil)
		return ErrIceRestartOnClosedPeerConnection
	}

	t.postEvent(event{
		signal: signalICERestart,
	})
	return nil
}

func (t *PCTransport) ResetShortConnOnICERestart() {
	t.resetShortConnOnICERestart.Store(true)
}

func (t *PCTransport) OnStreamStateChange(f func(update *streamallocator.StreamStateUpdate) error) {
	if t.streamAllocator == nil {
		return
	}

	t.streamAllocator.OnStreamStateChange(f)
}

func (t *PCTransport) AddTrackToStreamAllocator(subTrack types.SubscribedTrack) {
	if t.streamAllocator == nil {
		return
	}

	t.streamAllocator.AddTrack(subTrack.DownTrack(), streamallocator.AddTrackParams{
		Source:      subTrack.MediaTrack().Source(),
		IsSimulcast: subTrack.MediaTrack().IsSimulcast(),
		PublisherID: subTrack.MediaTrack().PublisherID(),
	})
}

func (t *PCTransport) RemoveTrackFromStreamAllocator(subTrack types.SubscribedTrack) {
	if t.streamAllocator == nil {
		return
	}

	t.streamAllocator.RemoveTrack(subTrack.DownTrack())
}

func (t *PCTransport) SetAllowPauseOfStreamAllocator(allowPause bool) {
	if t.streamAllocator == nil {
		return
	}

	t.streamAllocator.SetAllowPause(allowPause)
}

func (t *PCTransport) SetChannelCapacityOfStreamAllocator(channelCapacity int64) {
	if t.streamAllocator == nil {
		return
	}

	t.streamAllocator.SetChannelCapacity(channelCapacity)
}

func (t *PCTransport) preparePC(previousAnswer webrtc.SessionDescription) error {
	// sticky data channel to first m-lines, if someday we don't send sdp without media streams to
	// client's subscribe pc after joining, should change this step
	parsed, err := previousAnswer.Unmarshal()
	if err != nil {
		return err
	}
	fp, fpHahs, err := lksdp.ExtractFingerprint(parsed)
	if err != nil {
		return err
	}

	offer, err := t.pc.CreateOffer(nil)
	if err != nil {
		return err
	}
	if err := t.pc.SetLocalDescription(offer); err != nil {
		return err
	}

	//
	// Simulate client side peer connection and set DTLS role from previous answer.
	// Role needs to be set properly (one side needs to be server and the other side
	// needs to be the client) for DTLS connection to form properly. As this is
	// trying to replicate previous setup, read from previous answer and use that role.
	//
	se := webrtc.SettingEngine{}
	_ = se.SetAnsweringDTLSRole(lksdp.ExtractDTLSRole(parsed))
	api := webrtc.NewAPI(
		webrtc.WithSettingEngine(se),
		webrtc.WithMediaEngine(t.me),
	)
	pc2, err := api.NewPeerConnection(webrtc.Configuration{
		SDPSemantics: webrtc.SDPSemanticsUnifiedPlan,
	})
	if err != nil {
		return err
	}
	defer pc2.Close()

	if err := pc2.SetRemoteDescription(offer); err != nil {
		return err
	}
	ans, err := pc2.CreateAnswer(nil)
	if err != nil {
		return err
	}

	// replace client's fingerprint into dump pc's answer, for pion's dtls process, it will
	// keep the fingerprint at first call of SetRemoteDescription, if dummy pc and client pc use
	// different fingerprint, that will cause pion denied dtls data after handshake with client
	// complete (can't pass fingerprint change).
	// in this step, we don't established connection with dump pc(no candidate swap), just use
	// sdp negotiation to sticky data channel and keep client's fingerprint
	parsedAns, _ := ans.Unmarshal()
	fpLine := fpHahs + " " + fp
	replaceFP := func(attrs []sdp.Attribute, fpLine string) {
		for k := range attrs {
			if attrs[k].Key == "fingerprint" {
				attrs[k].Value = fpLine
			}
		}
	}
	replaceFP(parsedAns.Attributes, fpLine)
	for _, m := range parsedAns.MediaDescriptions {
		replaceFP(m.Attributes, fpLine)
	}
	bytes, err := parsedAns.Marshal()
	if err != nil {
		return err
	}
	ans.SDP = string(bytes)

	return t.pc.SetRemoteDescription(ans)
}

func (t *PCTransport) initPCWithPreviousAnswer(previousAnswer webrtc.SessionDescription) (map[string]*webrtc.RTPSender, error) {
	senders := make(map[string]*webrtc.RTPSender)
	parsed, err := previousAnswer.Unmarshal()
	if err != nil {
		return senders, err
	}
	for _, m := range parsed.MediaDescriptions {
		var codecType webrtc.RTPCodecType
		switch m.MediaName.Media {
		case "video":
			codecType = webrtc.RTPCodecTypeVideo
		case "audio":
			codecType = webrtc.RTPCodecTypeAudio
		case "application":
			// for pion generate unmatched sdp, it always appends data channel to last m-lines,
			// that not consistent with our previous answer that data channel might at middle-line
			// because sdp can negotiate multi times before migration.(it will sticky to the last m-line atfirst negotiate)
			// so use a dummy pc to negotiate sdp to fixed the datachannel's mid at same position with previous answer
			if err := t.preparePC(previousAnswer); err != nil {
				t.params.Logger.Errorw("prepare pc for migration failed", err)
				return senders, err
			}
			continue
		default:
			continue
		}
		tr, err := t.pc.AddTransceiverFromKind(codecType, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionSendonly})
		if err != nil {
			return senders, err
		}
		mid := lksdp.GetMidValue(m)
		if mid == "" {
			return senders, ErrMidNotFound
		}
		tr.SetMid(mid)

		// save mid -> senders for migration reuse
		sender := tr.Sender()
		senders[mid] = sender

		// set transceiver to inactive
		tr.SetSender(tr.Sender(), nil)
	}
	return senders, nil
}

func (t *PCTransport) SetPreviousSdp(offer, answer *webrtc.SessionDescription) {
	// when there is no previous answer, cannot migrate, force a full reconnect
	if answer == nil {
		if onNegotiationFailed := t.getOnNegotiationFailed(); onNegotiationFailed != nil {
			onNegotiationFailed()
		}
		return
	}

	t.lock.Lock()
	if t.pc.RemoteDescription() == nil && t.previousAnswer == nil {
		t.previousAnswer = answer
		if senders, err := t.initPCWithPreviousAnswer(*t.previousAnswer); err != nil {
			t.params.Logger.Errorw("initPCWithPreviousAnswer failed", err)
			t.lock.Unlock()

			if onNegotiationFailed := t.getOnNegotiationFailed(); onNegotiationFailed != nil {
				onNegotiationFailed()
			}
			return
		} else if offer != nil {
			// in migration case, can't reuse transceiver before negotiated except track subscribed at previous node
			t.canReuseTransceiver = false
			if err := t.parseTrackMid(*offer, senders); err != nil {
				t.params.Logger.Errorw("parse previous offer failed", err, "offer", offer.SDP)
			}
		}
	}
	t.lock.Unlock()
}

func (t *PCTransport) parseTrackMid(offer webrtc.SessionDescription, senders map[string]*webrtc.RTPSender) error {
	parsed, err := offer.Unmarshal()
	if err != nil {
		return err
	}

	t.previousTrackDescription = make(map[string]*trackDescription)
	for _, m := range parsed.MediaDescriptions {
		msid, ok := m.Attribute(sdp.AttrKeyMsid)
		if !ok {
			continue
		}

		if split := strings.Split(msid, " "); len(split) == 2 {
			trackID := split[1]
			mid := lksdp.GetMidValue(m)
			if mid == "" {
				return ErrMidNotFound
			}
			t.previousTrackDescription[trackID] = &trackDescription{
				mid:    mid,
				sender: senders[mid],
			}
		}
	}
	return nil
}

func (t *PCTransport) postEvent(event event) {
	t.eventsQueue.Enqueue(func() {
		err := t.handleEvent(&event)
		if err != nil {
			if !t.isClosed.Load() {
				t.params.Logger.Errorw("error handling event", err, "event", event.String())
				if onNegotiationFailed := t.getOnNegotiationFailed(); onNegotiationFailed != nil {
					onNegotiationFailed()
				}
			}
		}
	})
}

func (t *PCTransport) handleEvent(e *event) error {
	switch e.signal {
	case signalICEGatheringComplete:
		return t.handleICEGatheringComplete(e)
	case signalLocalICECandidate:
		return t.handleLocalICECandidate(e)
	case signalRemoteICECandidate:
		return t.handleRemoteICECandidate(e)
	case signalSendOffer:
		return t.handleSendOffer(e)
	case signalRemoteDescriptionReceived:
		return t.handleRemoteDescriptionReceived(e)
	case signalICERestart:
		return t.handleICERestart(e)
	}

	return nil
}

func (t *PCTransport) handleICEGatheringComplete(_ *event) error {
	if t.params.IsOfferer {
		return t.handleICEGatheringCompleteOfferer()
	} else {
		return t.handleICEGatheringCompleteAnswerer()
	}
}

func (t *PCTransport) handleICEGatheringCompleteOfferer() error {
	if !t.restartAfterGathering {
		return nil
	}

	t.params.Logger.Debugw("restarting ICE after ICE gathering")
	t.restartAfterGathering = false
	return t.doICERestart()
}

func (t *PCTransport) handleICEGatheringCompleteAnswerer() error {
	if t.pendingRestartIceOffer == nil {
		return nil
	}

	offer := *t.pendingRestartIceOffer
	t.pendingRestartIceOffer = nil

	t.params.Logger.Debugw("accept remote restart ice offer after ICE gathering")
	if err := t.setRemoteDescription(offer); err != nil {
		return err
	}

	return t.createAndSendAnswer()
}

func (t *PCTransport) localDescriptionSent() error {
	if !t.cacheLocalCandidates {
		return nil
	}

	t.cacheLocalCandidates = false

	cachedLocalCandidates := t.cachedLocalCandidates
	t.cachedLocalCandidates = nil

	if onICECandidate := t.getOnICECandidate(); onICECandidate != nil {
		for _, c := range cachedLocalCandidates {
			if err := onICECandidate(c); err != nil {
				return err
			}
		}

		return nil
	}

	return ErrNoICECandidateHandler
}

func (t *PCTransport) clearLocalDescriptionSent() {
	t.cacheLocalCandidates = true
	t.cachedLocalCandidates = nil
	t.connectionDetails.Clear()
}

func (t *PCTransport) handleLocalICECandidate(e *event) error {
	c := e.data.(*webrtc.ICECandidate)

	filtered := false
	if c != nil {
		if t.preferTCP.Load() && c.Protocol != webrtc.ICEProtocolTCP {
			t.params.Logger.Debugw("filtering out local candidate",
				"candidate", func() interface{} {
					return c.String()
				})
			filtered = true
		}
		t.connectionDetails.AddLocalCandidate(c, filtered)
	}

	if filtered {
		return nil
	}

	if t.cacheLocalCandidates {
		t.cachedLocalCandidates = append(t.cachedLocalCandidates, c)
		return nil
	}

	if onICECandidate := t.getOnICECandidate(); onICECandidate != nil {
		return onICECandidate(c)
	}

	return ErrNoICECandidateHandler
}

func (t *PCTransport) handleRemoteICECandidate(e *event) error {
	c := e.data.(*webrtc.ICECandidateInit)

	filtered := false
	if t.preferTCP.Load() && !strings.Contains(c.Candidate, "tcp") {
		t.params.Logger.Debugw("filtering out remote candidate", "candidate", c.Candidate)
		filtered = true
	}

	t.connectionDetails.AddRemoteCandidate(*c, filtered)
	if filtered {
		return nil
	}

	if t.pc.RemoteDescription() == nil {
		t.pendingRemoteCandidates = append(t.pendingRemoteCandidates, c)
		return nil
	}

	if err := t.pc.AddICECandidate(*c); err != nil {
		return errors.Wrap(err, "add ice candidate failed")
	}

	return nil
}

func (t *PCTransport) setNegotiationState(state NegotiationState) {
	t.negotiationState = state
	if onNegotiationStateChanged := t.getOnNegotiationStateChanged(); onNegotiationStateChanged != nil {
		onNegotiationStateChanged(t.negotiationState)
	}
}

func (t *PCTransport) filterCandidates(sd webrtc.SessionDescription, preferTCP bool) webrtc.SessionDescription {
	parsed, err := sd.Unmarshal()
	if err != nil {
		t.params.Logger.Errorw("could not unmarshal SDP to filter candidates", err)
		return sd
	}

	filterAttributes := func(attrs []sdp.Attribute) []sdp.Attribute {
		filteredAttrs := make([]sdp.Attribute, 0, len(attrs))
		for _, a := range attrs {
			if a.Key == sdp.AttrKeyCandidate {
				if preferTCP {
					if strings.Contains(a.Value, "tcp") {
						filteredAttrs = append(filteredAttrs, a)
					}
				} else {
					filteredAttrs = append(filteredAttrs, a)
				}
			} else {
				filteredAttrs = append(filteredAttrs, a)
			}
		}

		return filteredAttrs
	}

	parsed.Attributes = filterAttributes(parsed.Attributes)
	for _, m := range parsed.MediaDescriptions {
		m.Attributes = filterAttributes(m.Attributes)
	}

	bytes, err := parsed.Marshal()
	if err != nil {
		t.params.Logger.Errorw("could not marshal SDP to filter candidates", err)
		return sd
	}
	sd.SDP = string(bytes)
	return sd
}

func (t *PCTransport) clearSignalStateCheckTimer() {
	if t.signalStateCheckTimer != nil {
		t.signalStateCheckTimer.Stop()
		t.signalStateCheckTimer = nil
	}
}

func (t *PCTransport) setupSignalStateCheckTimer() {
	t.clearSignalStateCheckTimer()

	negotiateVersion := t.negotiateCounter.Inc()
	t.signalStateCheckTimer = time.AfterFunc(negotiationFailedTimeout, func() {
		t.clearSignalStateCheckTimer()

		failed := t.negotiationState != NegotiationStateNone

		if t.negotiateCounter.Load() == negotiateVersion && failed && t.pc.ConnectionState() == webrtc.PeerConnectionStateConnected {
			t.params.Logger.Infow(
				"negotiation timed out",
				"localCurrent", t.pc.CurrentLocalDescription(),
				"localPending", t.pc.PendingLocalDescription(),
				"remoteCurrent", t.pc.CurrentRemoteDescription(),
				"remotePending", t.pc.PendingRemoteDescription(),
			)
			if onNegotiationFailed := t.getOnNegotiationFailed(); onNegotiationFailed != nil {
				onNegotiationFailed()
			}
		}
	})
}

func (t *PCTransport) createAndSendOffer(options *webrtc.OfferOptions) error {
	if t.pc.ConnectionState() == webrtc.PeerConnectionStateClosed {
		t.params.Logger.Warnw("trying to send offer on closed peer connection", nil)
		return nil
	}

	// when there's an ongoing negotiation, let it finish and not disrupt its state
	if t.negotiationState == NegotiationStateRemote {
		t.params.Logger.Debugw("skipping negotiation, trying again later")
		t.setNegotiationState(NegotiationStateRetry)
		return nil
	} else if t.negotiationState == NegotiationStateRetry {
		// already set to retry, we can safely skip this attempt
		return nil
	}

	ensureICERestart := func(options *webrtc.OfferOptions) *webrtc.OfferOptions {
		if options == nil {
			options = &webrtc.OfferOptions{}
		}
		options.ICERestart = true
		return options
	}

	t.lock.Lock()
	if t.previousAnswer != nil {
		t.previousAnswer = nil
		options = ensureICERestart(options)
		t.params.Logger.Infow("ice restart due to previous answer")
	}
	t.lock.Unlock()

	if t.restartAtNextOffer {
		t.restartAtNextOffer = false
		options = ensureICERestart(options)
		t.params.Logger.Infow("ice restart at next offer")
	}

	if options != nil && options.ICERestart {
		t.clearLocalDescriptionSent()
	}

	offer, err := t.pc.CreateOffer(options)
	if err != nil {
		if errors.Is(err, webrtc.ErrConnectionClosed) {
			t.params.Logger.Warnw("trying to create offer on closed peer connection", nil)
			return nil
		}

		prometheus.ServiceOperationCounter.WithLabelValues("offer", "error", "create").Add(1)
		return errors.Wrap(err, "create offer failed")
	}

	preferTCP := t.preferTCP.Load()
	if preferTCP {
		t.params.Logger.Debugw("local offer (unfiltered)", "sdp", offer.SDP)
	}

	err = t.pc.SetLocalDescription(offer)
	if err != nil {
		if errors.Is(err, webrtc.ErrConnectionClosed) {
			t.params.Logger.Warnw("trying to set local description on closed peer connection", nil)
			return nil
		}

		prometheus.ServiceOperationCounter.WithLabelValues("offer", "error", "local_description").Add(1)
		return errors.Wrap(err, "setting local description failed")
	}

	//
	// Filter after setting local description as pion expects the offer
	// to match between CreateOffer and SetLocalDescription.
	// Filtered offer is sent to remote so that remote does not
	// see filtered candidates.
	//
	offer = t.filterCandidates(offer, preferTCP)
	if preferTCP {
		t.params.Logger.Debugw("local offer (filtered)", "sdp", offer.SDP)
	}

	// indicate waiting for remote
	t.setNegotiationState(NegotiationStateRemote)

	t.setupSignalStateCheckTimer()

	if onOffer := t.getOnOffer(); onOffer != nil {
		if err := onOffer(offer); err != nil {
			prometheus.ServiceOperationCounter.WithLabelValues("offer", "error", "write_message").Add(1)
			return errors.Wrap(err, "could not send offer")
		}

		prometheus.ServiceOperationCounter.WithLabelValues("offer", "success", "").Add(1)
		return t.localDescriptionSent()
	}

	return ErrNoOfferHandler
}

func (t *PCTransport) handleSendOffer(_ *event) error {
	return t.createAndSendOffer(nil)
}

func (t *PCTransport) handleRemoteDescriptionReceived(e *event) error {
	sd := e.data.(*webrtc.SessionDescription)
	if sd.Type == webrtc.SDPTypeOffer {
		return t.handleRemoteOfferReceived(sd)
	} else {
		return t.handleRemoteAnswerReceived(sd)
	}
}

func (t *PCTransport) isRemoteOfferRestartICE(sd *webrtc.SessionDescription) (string, bool, error) {
	parsed, err := sd.Unmarshal()
	if err != nil {
		return "", false, err
	}
	user, pwd, err := lksdp.ExtractICECredential(parsed)
	if err != nil {
		return "", false, err
	}

	credential := fmt.Sprintf("%s:%s", user, pwd)
	// ice credential changed, remote offer restart ice
	restartICE := t.currentOfferIceCredential != "" && t.currentOfferIceCredential != credential
	return credential, restartICE, nil
}

func (t *PCTransport) setRemoteDescription(sd webrtc.SessionDescription) error {
	// filter before setting remote description so that pion does not see filtered remote candidates
	preferTCP := t.preferTCP.Load()
	if preferTCP {
		t.params.Logger.Debugw("remote description (unfiltered)", "type", sd.Type, "sdp", sd.SDP)
	}
	sd = t.filterCandidates(sd, preferTCP)
	if preferTCP {
		t.params.Logger.Debugw("remote description (filtered)", "type", sd.Type, "sdp", sd.SDP)
	}

	if err := t.pc.SetRemoteDescription(sd); err != nil {
		if errors.Is(err, webrtc.ErrConnectionClosed) {
			t.params.Logger.Warnw("trying to set remote description on closed peer connection", nil)
			return nil
		}

		sdpType := "offer"
		if sd.Type == webrtc.SDPTypeAnswer {
			sdpType = "answer"
		}
		prometheus.ServiceOperationCounter.WithLabelValues(sdpType, "error", "remote_description").Add(1)
		return errors.Wrap(err, "setting remote description failed")
	} else if sd.Type == webrtc.SDPTypeAnswer {
		t.lock.Lock()
		if !t.canReuseTransceiver {
			t.canReuseTransceiver = true
			t.previousTrackDescription = make(map[string]*trackDescription)
		}
		t.lock.Unlock()
	}

	for _, c := range t.pendingRemoteCandidates {
		if err := t.pc.AddICECandidate(*c); err != nil {
			return errors.Wrap(err, "add ice candidate failed")
		}
	}
	t.pendingRemoteCandidates = nil

	return nil
}

func (t *PCTransport) createAndSendAnswer() error {
	answer, err := t.pc.CreateAnswer(nil)
	if err != nil {
		if errors.Is(err, webrtc.ErrConnectionClosed) {
			t.params.Logger.Warnw("trying to create answer on closed peer connection", nil)
			return nil
		}

		prometheus.ServiceOperationCounter.WithLabelValues("answer", "error", "create").Add(1)
		return errors.Wrap(err, "create answer failed")
	}

	preferTCP := t.preferTCP.Load()
	if preferTCP {
		t.params.Logger.Debugw("local answer (unfiltered)", "sdp", answer.SDP)
	}

	if err = t.pc.SetLocalDescription(answer); err != nil {
		prometheus.ServiceOperationCounter.WithLabelValues("answer", "error", "local_description").Add(1)
		return errors.Wrap(err, "setting local description failed")
	}

	//
	// Filter after setting local description as pion expects the answer
	// to match between CreateAnswer and SetLocalDescription.
	// Filtered answer is sent to remote so that remote does not
	// see filtered candidates.
	//
	answer = t.filterCandidates(answer, preferTCP)
	if preferTCP {
		t.params.Logger.Debugw("local answer (filtered)", "sdp", answer.SDP)
	}

	if onAnswer := t.getOnAnswer(); onAnswer != nil {
		if err := onAnswer(answer); err != nil {
			prometheus.ServiceOperationCounter.WithLabelValues("answer", "error", "write_message").Add(1)
			return errors.Wrap(err, "could not send answer")
		}

		prometheus.ServiceOperationCounter.WithLabelValues("answer", "success", "").Add(1)
		return t.localDescriptionSent()
	}

	return ErrNoAnswerHandler
}

func (t *PCTransport) handleRemoteOfferReceived(sd *webrtc.SessionDescription) error {
	iceCredential, offerRestartICE, err := t.isRemoteOfferRestartICE(sd)
	if err != nil {
		return errors.Wrap(err, "check remote offer restart ice failed")
	}

	if offerRestartICE && t.pendingRestartIceOffer == nil {
		t.clearLocalDescriptionSent()
	}

	if offerRestartICE && t.pc.ICEGatheringState() == webrtc.ICEGatheringStateGathering {
		t.params.Logger.Debugw("remote offer restart ice while ice gathering")
		t.pendingRestartIceOffer = sd
		return nil
	}

	if offerRestartICE && t.resetShortConnOnICERestart.CompareAndSwap(true, false) {
		t.resetShortConn()
	}

	if err := t.setRemoteDescription(*sd); err != nil {
		return err
	}

	if t.currentOfferIceCredential == "" || offerRestartICE {
		t.currentOfferIceCredential = iceCredential
	}

	return t.createAndSendAnswer()
}

func (t *PCTransport) handleRemoteAnswerReceived(sd *webrtc.SessionDescription) error {
	t.clearSignalStateCheckTimer()

	if err := t.setRemoteDescription(*sd); err != nil {
		// Pion will call RTPSender.Send method for each new added Downtrack, and return error if the DownTrack.Bind
		// returns error. In case of Downtrack.Bind returns ErrUnsupportedCodec, the signal state will be stable as negotiation is aleady compelted
		// before startRTPSenders, and the peerconnection state can be recovered by next negotiation which will be triggered
		// by the SubscriptionManager unsubscribe the failure DownTrack. So don't treat this error as negotiation failure.
		if !errors.Is(err, webrtc.ErrUnsupportedCodec) {
			return err
		}
	}

	if t.negotiationState == NegotiationStateRetry {
		t.setNegotiationState(NegotiationStateNone)

		t.params.Logger.Debugw("re-negotiate after receiving answer")
		return t.createAndSendOffer(nil)
	}

	t.setNegotiationState(NegotiationStateNone)
	return nil
}

func (t *PCTransport) doICERestart() error {
	if t.pc.ConnectionState() == webrtc.PeerConnectionStateClosed {
		t.params.Logger.Warnw("trying to restart ICE on closed peer connection", nil)
		return nil
	}

	// if restart is requested, and we are not ready, then continue afterwards
	if t.pc.ICEGatheringState() == webrtc.ICEGatheringStateGathering {
		t.params.Logger.Debugw("deferring ICE restart to after gathering")
		t.restartAfterGathering = true
		return nil
	}

	if t.resetShortConnOnICERestart.CompareAndSwap(true, false) {
		t.resetShortConn()
	}

	if t.negotiationState == NegotiationStateNone {
		return t.createAndSendOffer(&webrtc.OfferOptions{ICERestart: true})
	}

	currentRemoteDescription := t.pc.CurrentRemoteDescription()
	if currentRemoteDescription == nil {
		// restart without current remote description, send current local description again to try recover
		offer := t.pc.LocalDescription()
		if offer == nil {
			// it should not happen, log just in case
			t.params.Logger.Warnw("ice restart without local offer", nil)
			return ErrIceRestartWithoutLocalSDP
		} else {
			t.params.Logger.Infow("deferring ice restart to next offer")
			t.setNegotiationState(NegotiationStateRetry)
			t.restartAtNextOffer = true
			if onOffer := t.getOnOffer(); onOffer != nil {
				err := onOffer(*offer)
				if err != nil {
					prometheus.ServiceOperationCounter.WithLabelValues("offer", "error", "write_message").Add(1)
				} else {
					prometheus.ServiceOperationCounter.WithLabelValues("offer", "success", "").Add(1)
				}
				return err
			}
			return ErrNoOfferHandler
		}
	} else {
		// recover by re-applying the last answer
		t.params.Logger.Infow("recovering from client negotiation state on ICE restart")
		if err := t.pc.SetRemoteDescription(*currentRemoteDescription); err != nil {
			prometheus.ServiceOperationCounter.WithLabelValues("offer", "error", "remote_description").Add(1)
			return errors.Wrap(err, "set remote description failed")
		} else {
			t.setNegotiationState(NegotiationStateNone)
			return t.createAndSendOffer(&webrtc.OfferOptions{ICERestart: true})
		}
	}
}

func (t *PCTransport) handleICERestart(_ *event) error {
	return t.doICERestart()
}

// configure subscriber transceiver for audio stereo and nack
func configureAudioTransceiver(tr *webrtc.RTPTransceiver, stereo bool, nack bool) {
	sender := tr.Sender()
	if sender == nil {
		return
	}
	// enable stereo
	codecs := sender.GetParameters().Codecs
	configCodecs := make([]webrtc.RTPCodecParameters, 0, len(codecs))
	for _, c := range codecs {
		if strings.EqualFold(c.MimeType, webrtc.MimeTypeOpus) {
			c.SDPFmtpLine = strings.ReplaceAll(c.SDPFmtpLine, ";sprop-stereo=1", "")
			if stereo {
				c.SDPFmtpLine += ";sprop-stereo=1"
			}
			if nack {
				var nackFound bool
				for _, fb := range c.RTCPFeedback {
					if fb.Type == webrtc.TypeRTCPFBNACK {
						nackFound = true
						break
					}
				}
				if !nackFound {
					c.RTCPFeedback = append(c.RTCPFeedback, webrtc.RTCPFeedback{Type: webrtc.TypeRTCPFBNACK})
				}
			}
		}
		configCodecs = append(configCodecs, c)
	}

	tr.SetCodecPreferences(configCodecs)
}
