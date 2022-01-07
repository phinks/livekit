package types

import (
	"time"

	"github.com/livekit/protocol/livekit"
	"github.com/pion/webrtc/v3"

	"github.com/livekit/livekit-server/pkg/routing"
	"github.com/livekit/livekit-server/pkg/sfu"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate . WebsocketClient
type WebsocketClient interface {
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
	WriteControl(messageType int, data []byte, deadline time.Time) error
}

type AddSubscriberParams struct {
	AllTracks bool
	TrackIDs  []livekit.TrackID
}

//counterfeiter:generate . Participant
type Participant interface {
	ID() livekit.ParticipantID
	Identity() livekit.ParticipantIdentity
	State() livekit.ParticipantInfo_State
	ProtocolVersion() ProtocolVersion
	IsReady() bool
	ConnectedAt() time.Time
	ToProto() *livekit.ParticipantInfo
	SetMetadata(metadata string)
	SetPermission(permission *livekit.ParticipantPermission)
	GetResponseSink() routing.MessageSink
	SetResponseSink(sink routing.MessageSink)
	SubscriberMediaEngine() *webrtc.MediaEngine
	Negotiate()
	ICERestart() error

	AddTrack(req *livekit.AddTrackRequest)
	GetPublishedTrack(sid livekit.TrackID) PublishedTrack
	GetPublishedTracks() []PublishedTrack
	GetSubscribedTrack(sid livekit.TrackID) SubscribedTrack
	GetSubscribedTracks() []SubscribedTrack
	HandleOffer(sdp webrtc.SessionDescription) (answer webrtc.SessionDescription, err error)
	HandleAnswer(sdp webrtc.SessionDescription) error
	AddICECandidate(candidate webrtc.ICECandidateInit, target livekit.SignalTarget) error
	AddSubscriber(op Participant, params AddSubscriberParams) (int, error)
	RemoveSubscriber(op Participant, trackID livekit.TrackID)
	SendJoinResponse(info *livekit.Room, otherParticipants []*livekit.ParticipantInfo, iceServers []*livekit.ICEServer) error
	SendParticipantUpdate(participants []*livekit.ParticipantInfo, updatedAt time.Time) error
	SendSpeakerUpdate(speakers []*livekit.SpeakerInfo) error
	SendDataPacket(packet *livekit.DataPacket) error
	SendRoomUpdate(room *livekit.Room) error
	SendConnectionQualityUpdate(update *livekit.ConnectionQualityUpdate) error
	SetTrackMuted(trackID livekit.TrackID, muted bool, fromAdmin bool)
	GetAudioLevel() (level uint8, active bool)
	GetConnectionQuality() *livekit.ConnectionQualityInfo
	IsSubscribedTo(participantID livekit.ParticipantID) bool
	// returns list of participant identities that the current participant is subscribed to
	GetSubscribedParticipants() []livekit.ParticipantID

	// permissions
	CanPublish() bool
	CanSubscribe() bool
	CanPublishData() bool
	Hidden() bool
	IsRecorder() bool
	SubscriberAsPrimary() bool

	Start()
	Close() error

	// callbacks
	OnStateChange(func(p Participant, oldState livekit.ParticipantInfo_State))
	// OnTrackPublished - remote added a remoteTrack
	OnTrackPublished(func(Participant, PublishedTrack))
	// OnTrackUpdated - one of its publishedTracks changed in status
	OnTrackUpdated(callback func(Participant, PublishedTrack))
	OnMetadataUpdate(callback func(Participant))
	OnDataPacket(callback func(Participant, *livekit.DataPacket))
	OnClose(func(Participant, map[livekit.TrackID]livekit.ParticipantID))

	// package methods
	AddSubscribedTrack(st SubscribedTrack)
	RemoveSubscribedTrack(st SubscribedTrack)
	SubscriberPC() *webrtc.PeerConnection

	UpdateSubscriptionPermissions(permissions *livekit.UpdateSubscriptionPermissions, resolver func(participantID livekit.ParticipantID) Participant) error
	SubscriptionPermissionUpdate(publisherID livekit.ParticipantID, trackID livekit.TrackID, allowed bool)

	UpdateVideoLayers(updateVideoLayers *livekit.UpdateVideoLayers) error

	UpdateSubscribedQuality(nodeID string, trackID livekit.TrackID, maxQuality livekit.VideoQuality) error

	UpdateMediaLoss(nodeID string, trackID livekit.TrackID, fractionalLoss uint32) error

	DebugInfo() map[string]interface{}
}

// Room is a container of participants, and can provide room level actions
//counterfeiter:generate . Room
type Room interface {
	Name() livekit.RoomName
	UpdateSubscriptions(participant Participant, trackIDs []livekit.TrackID, participantTracks []*livekit.ParticipantTracks, subscribe bool) error
	UpdateSubscriptionPermissions(participant Participant, permissions *livekit.UpdateSubscriptionPermissions) error

	UpdateVideoLayers(participant Participant, updateVideoLayers *livekit.UpdateVideoLayers) error
}

// MediaTrack represents a media track
//counterfeiter:generate . MediaTrack
type MediaTrack interface {
	ID() livekit.TrackID
	Kind() livekit.TrackType
	Name() string
	IsMuted() bool
	SetMuted(muted bool)
	UpdateVideoLayers(layers []*livekit.VideoLayer)
	Source() livekit.TrackSource
	IsSimulcast() bool

	PublisherID() livekit.ParticipantID
	PublisherIdentity() livekit.ParticipantIdentity

	ToProto() *livekit.TrackInfo

	// subscribers
	AddSubscriber(participant Participant) error
	RemoveSubscriber(participantID livekit.ParticipantID)
	IsSubscriber(subID livekit.ParticipantID) bool
	RemoveAllSubscribers()
	RevokeDisallowedSubscribers(allowedSubscriberIDs []livekit.ParticipantID) []livekit.ParticipantID

	// returns quality information that's appropriate for width & height
	GetQualityForDimension(width, height uint32) livekit.VideoQuality

	NotifySubscriberMaxQuality(subscriberID livekit.ParticipantID, quality livekit.VideoQuality)
	NotifySubscriberNodeMaxQuality(nodeID string, quality livekit.VideoQuality)

	NotifySubscriberNodeMediaLoss(nodeID string, fractionalLoss uint8)
}

// PublishedTrack is the main interface representing a track published to the room
// it's responsible for managing subscribers and forwarding data from the input track to all subscribers
//counterfeiter:generate . PublishedTrack
type PublishedTrack interface {
	MediaTrack

	SignalCid() string
	SdpCid() string
	ToProto() *livekit.TrackInfo

	PublishLossPercentage() uint32
	Receiver() sfu.TrackReceiver
	GetConnectionScore() float64

	GetAudioLevel() (level uint8, active bool)

	UpdateVideoLayers(layers []*livekit.VideoLayer)

	OnSubscribedMaxQualityChange(f func(trackID livekit.TrackID, subscribedQualities []*livekit.SubscribedQuality, maxQuality livekit.VideoQuality) error)

	// callbacks
	AddOnClose(func())
}

//counterfeiter:generate . SubscribedTrack
type SubscribedTrack interface {
	OnBind(f func())
	ID() livekit.TrackID
	PublisherID() livekit.ParticipantID
	PublisherIdentity() livekit.ParticipantIdentity
	DownTrack() *sfu.DownTrack
	MediaTrack() MediaTrack
	IsMuted() bool
	SetPublisherMuted(muted bool)
	UpdateSubscriberSettings(settings *livekit.UpdateTrackSettings)
	// selects appropriate video layer according to subscriber preferences
	UpdateVideoLayer()
}

// interface for properties of webrtc.TrackRemote
//counterfeiter:generate . TrackRemote
type TrackRemote interface {
	SSRC() webrtc.SSRC
	StreamID() string
	Kind() webrtc.RTPCodecType
	Codec() webrtc.RTPCodecParameters
}
