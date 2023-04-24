// Code generated by counterfeiter. DO NOT EDIT.
package typesfakes

import (
	"sync"

	"github.com/livekit/livekit-server/pkg/rtc/types"
	"github.com/livekit/protocol/livekit"
	"github.com/livekit/protocol/utils"
)

type FakeParticipant struct {
	CloseStub        func(bool, types.ParticipantCloseReason) error
	closeMutex       sync.RWMutex
	closeArgsForCall []struct {
		arg1 bool
		arg2 types.ParticipantCloseReason
	}
	closeReturns struct {
		result1 error
	}
	closeReturnsOnCall map[int]struct {
		result1 error
	}
	DebugInfoStub        func() map[string]interface{}
	debugInfoMutex       sync.RWMutex
	debugInfoArgsForCall []struct {
	}
	debugInfoReturns struct {
		result1 map[string]interface{}
	}
	debugInfoReturnsOnCall map[int]struct {
		result1 map[string]interface{}
	}
	GetPublishedTrackStub        func(livekit.TrackID) types.MediaTrack
	getPublishedTrackMutex       sync.RWMutex
	getPublishedTrackArgsForCall []struct {
		arg1 livekit.TrackID
	}
	getPublishedTrackReturns struct {
		result1 types.MediaTrack
	}
	getPublishedTrackReturnsOnCall map[int]struct {
		result1 types.MediaTrack
	}
	GetPublishedTracksStub        func() []types.MediaTrack
	getPublishedTracksMutex       sync.RWMutex
	getPublishedTracksArgsForCall []struct {
	}
	getPublishedTracksReturns struct {
		result1 []types.MediaTrack
	}
	getPublishedTracksReturnsOnCall map[int]struct {
		result1 []types.MediaTrack
	}
	HasPermissionStub        func(livekit.TrackID, livekit.ParticipantIdentity) bool
	hasPermissionMutex       sync.RWMutex
	hasPermissionArgsForCall []struct {
		arg1 livekit.TrackID
		arg2 livekit.ParticipantIdentity
	}
	hasPermissionReturns struct {
		result1 bool
	}
	hasPermissionReturnsOnCall map[int]struct {
		result1 bool
	}
	HiddenStub        func() bool
	hiddenMutex       sync.RWMutex
	hiddenArgsForCall []struct {
	}
	hiddenReturns struct {
		result1 bool
	}
	hiddenReturnsOnCall map[int]struct {
		result1 bool
	}
	IDStub        func() livekit.ParticipantID
	iDMutex       sync.RWMutex
	iDArgsForCall []struct {
	}
	iDReturns struct {
		result1 livekit.ParticipantID
	}
	iDReturnsOnCall map[int]struct {
		result1 livekit.ParticipantID
	}
	IdentityStub        func() livekit.ParticipantIdentity
	identityMutex       sync.RWMutex
	identityArgsForCall []struct {
	}
	identityReturns struct {
		result1 livekit.ParticipantIdentity
	}
	identityReturnsOnCall map[int]struct {
		result1 livekit.ParticipantIdentity
	}
	IsPublisherStub        func() bool
	isPublisherMutex       sync.RWMutex
	isPublisherArgsForCall []struct {
	}
	isPublisherReturns struct {
		result1 bool
	}
	isPublisherReturnsOnCall map[int]struct {
		result1 bool
	}
	IsRecorderStub        func() bool
	isRecorderMutex       sync.RWMutex
	isRecorderArgsForCall []struct {
	}
	isRecorderReturns struct {
		result1 bool
	}
	isRecorderReturnsOnCall map[int]struct {
		result1 bool
	}
	RemovePublishedTrackStub        func(types.MediaTrack, bool, bool)
	removePublishedTrackMutex       sync.RWMutex
	removePublishedTrackArgsForCall []struct {
		arg1 types.MediaTrack
		arg2 bool
		arg3 bool
	}
	SetMetadataStub        func(string)
	setMetadataMutex       sync.RWMutex
	setMetadataArgsForCall []struct {
		arg1 string
	}
	SetNameStub        func(string)
	setNameMutex       sync.RWMutex
	setNameArgsForCall []struct {
		arg1 string
	}
	StartStub        func()
	startMutex       sync.RWMutex
	startArgsForCall []struct {
	}
	StateStub        func() livekit.ParticipantInfo_State
	stateMutex       sync.RWMutex
	stateArgsForCall []struct {
	}
	stateReturns struct {
		result1 livekit.ParticipantInfo_State
	}
	stateReturnsOnCall map[int]struct {
		result1 livekit.ParticipantInfo_State
	}
	SubscriptionPermissionStub        func() (*livekit.SubscriptionPermission, utils.TimedVersion)
	subscriptionPermissionMutex       sync.RWMutex
	subscriptionPermissionArgsForCall []struct {
	}
	subscriptionPermissionReturns struct {
		result1 *livekit.SubscriptionPermission
		result2 utils.TimedVersion
	}
	subscriptionPermissionReturnsOnCall map[int]struct {
		result1 *livekit.SubscriptionPermission
		result2 utils.TimedVersion
	}
	ToProtoStub        func() *livekit.ParticipantInfo
	toProtoMutex       sync.RWMutex
	toProtoArgsForCall []struct {
	}
	toProtoReturns struct {
		result1 *livekit.ParticipantInfo
	}
	toProtoReturnsOnCall map[int]struct {
		result1 *livekit.ParticipantInfo
	}
	UpdateSubscriptionPermissionStub        func(*livekit.SubscriptionPermission, utils.TimedVersion, func(participantIdentity livekit.ParticipantIdentity) types.LocalParticipant, func(participantID livekit.ParticipantID) types.LocalParticipant) error
	updateSubscriptionPermissionMutex       sync.RWMutex
	updateSubscriptionPermissionArgsForCall []struct {
		arg1 *livekit.SubscriptionPermission
		arg2 utils.TimedVersion
		arg3 func(participantIdentity livekit.ParticipantIdentity) types.LocalParticipant
		arg4 func(participantID livekit.ParticipantID) types.LocalParticipant
	}
	updateSubscriptionPermissionReturns struct {
		result1 error
	}
	updateSubscriptionPermissionReturnsOnCall map[int]struct {
		result1 error
	}
	UpdateVideoLayersStub        func(*livekit.UpdateVideoLayers) error
	updateVideoLayersMutex       sync.RWMutex
	updateVideoLayersArgsForCall []struct {
		arg1 *livekit.UpdateVideoLayers
	}
	updateVideoLayersReturns struct {
		result1 error
	}
	updateVideoLayersReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeParticipant) Close(arg1 bool, arg2 types.ParticipantCloseReason) error {
	fake.closeMutex.Lock()
	ret, specificReturn := fake.closeReturnsOnCall[len(fake.closeArgsForCall)]
	fake.closeArgsForCall = append(fake.closeArgsForCall, struct {
		arg1 bool
		arg2 types.ParticipantCloseReason
	}{arg1, arg2})
	stub := fake.CloseStub
	fakeReturns := fake.closeReturns
	fake.recordInvocation("Close", []interface{}{arg1, arg2})
	fake.closeMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeParticipant) CloseCallCount() int {
	fake.closeMutex.RLock()
	defer fake.closeMutex.RUnlock()
	return len(fake.closeArgsForCall)
}

func (fake *FakeParticipant) CloseCalls(stub func(bool, types.ParticipantCloseReason) error) {
	fake.closeMutex.Lock()
	defer fake.closeMutex.Unlock()
	fake.CloseStub = stub
}

func (fake *FakeParticipant) CloseArgsForCall(i int) (bool, types.ParticipantCloseReason) {
	fake.closeMutex.RLock()
	defer fake.closeMutex.RUnlock()
	argsForCall := fake.closeArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeParticipant) CloseReturns(result1 error) {
	fake.closeMutex.Lock()
	defer fake.closeMutex.Unlock()
	fake.CloseStub = nil
	fake.closeReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeParticipant) CloseReturnsOnCall(i int, result1 error) {
	fake.closeMutex.Lock()
	defer fake.closeMutex.Unlock()
	fake.CloseStub = nil
	if fake.closeReturnsOnCall == nil {
		fake.closeReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.closeReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeParticipant) DebugInfo() map[string]interface{} {
	fake.debugInfoMutex.Lock()
	ret, specificReturn := fake.debugInfoReturnsOnCall[len(fake.debugInfoArgsForCall)]
	fake.debugInfoArgsForCall = append(fake.debugInfoArgsForCall, struct {
	}{})
	stub := fake.DebugInfoStub
	fakeReturns := fake.debugInfoReturns
	fake.recordInvocation("DebugInfo", []interface{}{})
	fake.debugInfoMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeParticipant) DebugInfoCallCount() int {
	fake.debugInfoMutex.RLock()
	defer fake.debugInfoMutex.RUnlock()
	return len(fake.debugInfoArgsForCall)
}

func (fake *FakeParticipant) DebugInfoCalls(stub func() map[string]interface{}) {
	fake.debugInfoMutex.Lock()
	defer fake.debugInfoMutex.Unlock()
	fake.DebugInfoStub = stub
}

func (fake *FakeParticipant) DebugInfoReturns(result1 map[string]interface{}) {
	fake.debugInfoMutex.Lock()
	defer fake.debugInfoMutex.Unlock()
	fake.DebugInfoStub = nil
	fake.debugInfoReturns = struct {
		result1 map[string]interface{}
	}{result1}
}

func (fake *FakeParticipant) DebugInfoReturnsOnCall(i int, result1 map[string]interface{}) {
	fake.debugInfoMutex.Lock()
	defer fake.debugInfoMutex.Unlock()
	fake.DebugInfoStub = nil
	if fake.debugInfoReturnsOnCall == nil {
		fake.debugInfoReturnsOnCall = make(map[int]struct {
			result1 map[string]interface{}
		})
	}
	fake.debugInfoReturnsOnCall[i] = struct {
		result1 map[string]interface{}
	}{result1}
}

func (fake *FakeParticipant) GetPublishedTrack(arg1 livekit.TrackID) types.MediaTrack {
	fake.getPublishedTrackMutex.Lock()
	ret, specificReturn := fake.getPublishedTrackReturnsOnCall[len(fake.getPublishedTrackArgsForCall)]
	fake.getPublishedTrackArgsForCall = append(fake.getPublishedTrackArgsForCall, struct {
		arg1 livekit.TrackID
	}{arg1})
	stub := fake.GetPublishedTrackStub
	fakeReturns := fake.getPublishedTrackReturns
	fake.recordInvocation("GetPublishedTrack", []interface{}{arg1})
	fake.getPublishedTrackMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeParticipant) GetPublishedTrackCallCount() int {
	fake.getPublishedTrackMutex.RLock()
	defer fake.getPublishedTrackMutex.RUnlock()
	return len(fake.getPublishedTrackArgsForCall)
}

func (fake *FakeParticipant) GetPublishedTrackCalls(stub func(livekit.TrackID) types.MediaTrack) {
	fake.getPublishedTrackMutex.Lock()
	defer fake.getPublishedTrackMutex.Unlock()
	fake.GetPublishedTrackStub = stub
}

func (fake *FakeParticipant) GetPublishedTrackArgsForCall(i int) livekit.TrackID {
	fake.getPublishedTrackMutex.RLock()
	defer fake.getPublishedTrackMutex.RUnlock()
	argsForCall := fake.getPublishedTrackArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeParticipant) GetPublishedTrackReturns(result1 types.MediaTrack) {
	fake.getPublishedTrackMutex.Lock()
	defer fake.getPublishedTrackMutex.Unlock()
	fake.GetPublishedTrackStub = nil
	fake.getPublishedTrackReturns = struct {
		result1 types.MediaTrack
	}{result1}
}

func (fake *FakeParticipant) GetPublishedTrackReturnsOnCall(i int, result1 types.MediaTrack) {
	fake.getPublishedTrackMutex.Lock()
	defer fake.getPublishedTrackMutex.Unlock()
	fake.GetPublishedTrackStub = nil
	if fake.getPublishedTrackReturnsOnCall == nil {
		fake.getPublishedTrackReturnsOnCall = make(map[int]struct {
			result1 types.MediaTrack
		})
	}
	fake.getPublishedTrackReturnsOnCall[i] = struct {
		result1 types.MediaTrack
	}{result1}
}

func (fake *FakeParticipant) GetPublishedTracks() []types.MediaTrack {
	fake.getPublishedTracksMutex.Lock()
	ret, specificReturn := fake.getPublishedTracksReturnsOnCall[len(fake.getPublishedTracksArgsForCall)]
	fake.getPublishedTracksArgsForCall = append(fake.getPublishedTracksArgsForCall, struct {
	}{})
	stub := fake.GetPublishedTracksStub
	fakeReturns := fake.getPublishedTracksReturns
	fake.recordInvocation("GetPublishedTracks", []interface{}{})
	fake.getPublishedTracksMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeParticipant) GetPublishedTracksCallCount() int {
	fake.getPublishedTracksMutex.RLock()
	defer fake.getPublishedTracksMutex.RUnlock()
	return len(fake.getPublishedTracksArgsForCall)
}

func (fake *FakeParticipant) GetPublishedTracksCalls(stub func() []types.MediaTrack) {
	fake.getPublishedTracksMutex.Lock()
	defer fake.getPublishedTracksMutex.Unlock()
	fake.GetPublishedTracksStub = stub
}

func (fake *FakeParticipant) GetPublishedTracksReturns(result1 []types.MediaTrack) {
	fake.getPublishedTracksMutex.Lock()
	defer fake.getPublishedTracksMutex.Unlock()
	fake.GetPublishedTracksStub = nil
	fake.getPublishedTracksReturns = struct {
		result1 []types.MediaTrack
	}{result1}
}

func (fake *FakeParticipant) GetPublishedTracksReturnsOnCall(i int, result1 []types.MediaTrack) {
	fake.getPublishedTracksMutex.Lock()
	defer fake.getPublishedTracksMutex.Unlock()
	fake.GetPublishedTracksStub = nil
	if fake.getPublishedTracksReturnsOnCall == nil {
		fake.getPublishedTracksReturnsOnCall = make(map[int]struct {
			result1 []types.MediaTrack
		})
	}
	fake.getPublishedTracksReturnsOnCall[i] = struct {
		result1 []types.MediaTrack
	}{result1}
}

func (fake *FakeParticipant) HasPermission(arg1 livekit.TrackID, arg2 livekit.ParticipantIdentity) bool {
	fake.hasPermissionMutex.Lock()
	ret, specificReturn := fake.hasPermissionReturnsOnCall[len(fake.hasPermissionArgsForCall)]
	fake.hasPermissionArgsForCall = append(fake.hasPermissionArgsForCall, struct {
		arg1 livekit.TrackID
		arg2 livekit.ParticipantIdentity
	}{arg1, arg2})
	stub := fake.HasPermissionStub
	fakeReturns := fake.hasPermissionReturns
	fake.recordInvocation("HasPermission", []interface{}{arg1, arg2})
	fake.hasPermissionMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeParticipant) HasPermissionCallCount() int {
	fake.hasPermissionMutex.RLock()
	defer fake.hasPermissionMutex.RUnlock()
	return len(fake.hasPermissionArgsForCall)
}

func (fake *FakeParticipant) HasPermissionCalls(stub func(livekit.TrackID, livekit.ParticipantIdentity) bool) {
	fake.hasPermissionMutex.Lock()
	defer fake.hasPermissionMutex.Unlock()
	fake.HasPermissionStub = stub
}

func (fake *FakeParticipant) HasPermissionArgsForCall(i int) (livekit.TrackID, livekit.ParticipantIdentity) {
	fake.hasPermissionMutex.RLock()
	defer fake.hasPermissionMutex.RUnlock()
	argsForCall := fake.hasPermissionArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeParticipant) HasPermissionReturns(result1 bool) {
	fake.hasPermissionMutex.Lock()
	defer fake.hasPermissionMutex.Unlock()
	fake.HasPermissionStub = nil
	fake.hasPermissionReturns = struct {
		result1 bool
	}{result1}
}

func (fake *FakeParticipant) HasPermissionReturnsOnCall(i int, result1 bool) {
	fake.hasPermissionMutex.Lock()
	defer fake.hasPermissionMutex.Unlock()
	fake.HasPermissionStub = nil
	if fake.hasPermissionReturnsOnCall == nil {
		fake.hasPermissionReturnsOnCall = make(map[int]struct {
			result1 bool
		})
	}
	fake.hasPermissionReturnsOnCall[i] = struct {
		result1 bool
	}{result1}
}

func (fake *FakeParticipant) Hidden() bool {
	fake.hiddenMutex.Lock()
	ret, specificReturn := fake.hiddenReturnsOnCall[len(fake.hiddenArgsForCall)]
	fake.hiddenArgsForCall = append(fake.hiddenArgsForCall, struct {
	}{})
	stub := fake.HiddenStub
	fakeReturns := fake.hiddenReturns
	fake.recordInvocation("Hidden", []interface{}{})
	fake.hiddenMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeParticipant) HiddenCallCount() int {
	fake.hiddenMutex.RLock()
	defer fake.hiddenMutex.RUnlock()
	return len(fake.hiddenArgsForCall)
}

func (fake *FakeParticipant) HiddenCalls(stub func() bool) {
	fake.hiddenMutex.Lock()
	defer fake.hiddenMutex.Unlock()
	fake.HiddenStub = stub
}

func (fake *FakeParticipant) HiddenReturns(result1 bool) {
	fake.hiddenMutex.Lock()
	defer fake.hiddenMutex.Unlock()
	fake.HiddenStub = nil
	fake.hiddenReturns = struct {
		result1 bool
	}{result1}
}

func (fake *FakeParticipant) HiddenReturnsOnCall(i int, result1 bool) {
	fake.hiddenMutex.Lock()
	defer fake.hiddenMutex.Unlock()
	fake.HiddenStub = nil
	if fake.hiddenReturnsOnCall == nil {
		fake.hiddenReturnsOnCall = make(map[int]struct {
			result1 bool
		})
	}
	fake.hiddenReturnsOnCall[i] = struct {
		result1 bool
	}{result1}
}

func (fake *FakeParticipant) ID() livekit.ParticipantID {
	fake.iDMutex.Lock()
	ret, specificReturn := fake.iDReturnsOnCall[len(fake.iDArgsForCall)]
	fake.iDArgsForCall = append(fake.iDArgsForCall, struct {
	}{})
	stub := fake.IDStub
	fakeReturns := fake.iDReturns
	fake.recordInvocation("ID", []interface{}{})
	fake.iDMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeParticipant) IDCallCount() int {
	fake.iDMutex.RLock()
	defer fake.iDMutex.RUnlock()
	return len(fake.iDArgsForCall)
}

func (fake *FakeParticipant) IDCalls(stub func() livekit.ParticipantID) {
	fake.iDMutex.Lock()
	defer fake.iDMutex.Unlock()
	fake.IDStub = stub
}

func (fake *FakeParticipant) IDReturns(result1 livekit.ParticipantID) {
	fake.iDMutex.Lock()
	defer fake.iDMutex.Unlock()
	fake.IDStub = nil
	fake.iDReturns = struct {
		result1 livekit.ParticipantID
	}{result1}
}

func (fake *FakeParticipant) IDReturnsOnCall(i int, result1 livekit.ParticipantID) {
	fake.iDMutex.Lock()
	defer fake.iDMutex.Unlock()
	fake.IDStub = nil
	if fake.iDReturnsOnCall == nil {
		fake.iDReturnsOnCall = make(map[int]struct {
			result1 livekit.ParticipantID
		})
	}
	fake.iDReturnsOnCall[i] = struct {
		result1 livekit.ParticipantID
	}{result1}
}

func (fake *FakeParticipant) Identity() livekit.ParticipantIdentity {
	fake.identityMutex.Lock()
	ret, specificReturn := fake.identityReturnsOnCall[len(fake.identityArgsForCall)]
	fake.identityArgsForCall = append(fake.identityArgsForCall, struct {
	}{})
	stub := fake.IdentityStub
	fakeReturns := fake.identityReturns
	fake.recordInvocation("Identity", []interface{}{})
	fake.identityMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeParticipant) IdentityCallCount() int {
	fake.identityMutex.RLock()
	defer fake.identityMutex.RUnlock()
	return len(fake.identityArgsForCall)
}

func (fake *FakeParticipant) IdentityCalls(stub func() livekit.ParticipantIdentity) {
	fake.identityMutex.Lock()
	defer fake.identityMutex.Unlock()
	fake.IdentityStub = stub
}

func (fake *FakeParticipant) IdentityReturns(result1 livekit.ParticipantIdentity) {
	fake.identityMutex.Lock()
	defer fake.identityMutex.Unlock()
	fake.IdentityStub = nil
	fake.identityReturns = struct {
		result1 livekit.ParticipantIdentity
	}{result1}
}

func (fake *FakeParticipant) IdentityReturnsOnCall(i int, result1 livekit.ParticipantIdentity) {
	fake.identityMutex.Lock()
	defer fake.identityMutex.Unlock()
	fake.IdentityStub = nil
	if fake.identityReturnsOnCall == nil {
		fake.identityReturnsOnCall = make(map[int]struct {
			result1 livekit.ParticipantIdentity
		})
	}
	fake.identityReturnsOnCall[i] = struct {
		result1 livekit.ParticipantIdentity
	}{result1}
}

func (fake *FakeParticipant) IsPublisher() bool {
	fake.isPublisherMutex.Lock()
	ret, specificReturn := fake.isPublisherReturnsOnCall[len(fake.isPublisherArgsForCall)]
	fake.isPublisherArgsForCall = append(fake.isPublisherArgsForCall, struct {
	}{})
	stub := fake.IsPublisherStub
	fakeReturns := fake.isPublisherReturns
	fake.recordInvocation("IsPublisher", []interface{}{})
	fake.isPublisherMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeParticipant) IsPublisherCallCount() int {
	fake.isPublisherMutex.RLock()
	defer fake.isPublisherMutex.RUnlock()
	return len(fake.isPublisherArgsForCall)
}

func (fake *FakeParticipant) IsPublisherCalls(stub func() bool) {
	fake.isPublisherMutex.Lock()
	defer fake.isPublisherMutex.Unlock()
	fake.IsPublisherStub = stub
}

func (fake *FakeParticipant) IsPublisherReturns(result1 bool) {
	fake.isPublisherMutex.Lock()
	defer fake.isPublisherMutex.Unlock()
	fake.IsPublisherStub = nil
	fake.isPublisherReturns = struct {
		result1 bool
	}{result1}
}

func (fake *FakeParticipant) IsPublisherReturnsOnCall(i int, result1 bool) {
	fake.isPublisherMutex.Lock()
	defer fake.isPublisherMutex.Unlock()
	fake.IsPublisherStub = nil
	if fake.isPublisherReturnsOnCall == nil {
		fake.isPublisherReturnsOnCall = make(map[int]struct {
			result1 bool
		})
	}
	fake.isPublisherReturnsOnCall[i] = struct {
		result1 bool
	}{result1}
}

func (fake *FakeParticipant) IsRecorder() bool {
	fake.isRecorderMutex.Lock()
	ret, specificReturn := fake.isRecorderReturnsOnCall[len(fake.isRecorderArgsForCall)]
	fake.isRecorderArgsForCall = append(fake.isRecorderArgsForCall, struct {
	}{})
	stub := fake.IsRecorderStub
	fakeReturns := fake.isRecorderReturns
	fake.recordInvocation("IsRecorder", []interface{}{})
	fake.isRecorderMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeParticipant) IsRecorderCallCount() int {
	fake.isRecorderMutex.RLock()
	defer fake.isRecorderMutex.RUnlock()
	return len(fake.isRecorderArgsForCall)
}

func (fake *FakeParticipant) IsRecorderCalls(stub func() bool) {
	fake.isRecorderMutex.Lock()
	defer fake.isRecorderMutex.Unlock()
	fake.IsRecorderStub = stub
}

func (fake *FakeParticipant) IsRecorderReturns(result1 bool) {
	fake.isRecorderMutex.Lock()
	defer fake.isRecorderMutex.Unlock()
	fake.IsRecorderStub = nil
	fake.isRecorderReturns = struct {
		result1 bool
	}{result1}
}

func (fake *FakeParticipant) IsRecorderReturnsOnCall(i int, result1 bool) {
	fake.isRecorderMutex.Lock()
	defer fake.isRecorderMutex.Unlock()
	fake.IsRecorderStub = nil
	if fake.isRecorderReturnsOnCall == nil {
		fake.isRecorderReturnsOnCall = make(map[int]struct {
			result1 bool
		})
	}
	fake.isRecorderReturnsOnCall[i] = struct {
		result1 bool
	}{result1}
}

func (fake *FakeParticipant) RemovePublishedTrack(arg1 types.MediaTrack, arg2 bool, arg3 bool) {
	fake.removePublishedTrackMutex.Lock()
	fake.removePublishedTrackArgsForCall = append(fake.removePublishedTrackArgsForCall, struct {
		arg1 types.MediaTrack
		arg2 bool
		arg3 bool
	}{arg1, arg2, arg3})
	stub := fake.RemovePublishedTrackStub
	fake.recordInvocation("RemovePublishedTrack", []interface{}{arg1, arg2, arg3})
	fake.removePublishedTrackMutex.Unlock()
	if stub != nil {
		fake.RemovePublishedTrackStub(arg1, arg2, arg3)
	}
}

func (fake *FakeParticipant) RemovePublishedTrackCallCount() int {
	fake.removePublishedTrackMutex.RLock()
	defer fake.removePublishedTrackMutex.RUnlock()
	return len(fake.removePublishedTrackArgsForCall)
}

func (fake *FakeParticipant) RemovePublishedTrackCalls(stub func(types.MediaTrack, bool, bool)) {
	fake.removePublishedTrackMutex.Lock()
	defer fake.removePublishedTrackMutex.Unlock()
	fake.RemovePublishedTrackStub = stub
}

func (fake *FakeParticipant) RemovePublishedTrackArgsForCall(i int) (types.MediaTrack, bool, bool) {
	fake.removePublishedTrackMutex.RLock()
	defer fake.removePublishedTrackMutex.RUnlock()
	argsForCall := fake.removePublishedTrackArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeParticipant) SetMetadata(arg1 string) {
	fake.setMetadataMutex.Lock()
	fake.setMetadataArgsForCall = append(fake.setMetadataArgsForCall, struct {
		arg1 string
	}{arg1})
	stub := fake.SetMetadataStub
	fake.recordInvocation("SetMetadata", []interface{}{arg1})
	fake.setMetadataMutex.Unlock()
	if stub != nil {
		fake.SetMetadataStub(arg1)
	}
}

func (fake *FakeParticipant) SetMetadataCallCount() int {
	fake.setMetadataMutex.RLock()
	defer fake.setMetadataMutex.RUnlock()
	return len(fake.setMetadataArgsForCall)
}

func (fake *FakeParticipant) SetMetadataCalls(stub func(string)) {
	fake.setMetadataMutex.Lock()
	defer fake.setMetadataMutex.Unlock()
	fake.SetMetadataStub = stub
}

func (fake *FakeParticipant) SetMetadataArgsForCall(i int) string {
	fake.setMetadataMutex.RLock()
	defer fake.setMetadataMutex.RUnlock()
	argsForCall := fake.setMetadataArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeParticipant) SetName(arg1 string) {
	fake.setNameMutex.Lock()
	fake.setNameArgsForCall = append(fake.setNameArgsForCall, struct {
		arg1 string
	}{arg1})
	stub := fake.SetNameStub
	fake.recordInvocation("SetName", []interface{}{arg1})
	fake.setNameMutex.Unlock()
	if stub != nil {
		fake.SetNameStub(arg1)
	}
}

func (fake *FakeParticipant) SetNameCallCount() int {
	fake.setNameMutex.RLock()
	defer fake.setNameMutex.RUnlock()
	return len(fake.setNameArgsForCall)
}

func (fake *FakeParticipant) SetNameCalls(stub func(string)) {
	fake.setNameMutex.Lock()
	defer fake.setNameMutex.Unlock()
	fake.SetNameStub = stub
}

func (fake *FakeParticipant) SetNameArgsForCall(i int) string {
	fake.setNameMutex.RLock()
	defer fake.setNameMutex.RUnlock()
	argsForCall := fake.setNameArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeParticipant) Start() {
	fake.startMutex.Lock()
	fake.startArgsForCall = append(fake.startArgsForCall, struct {
	}{})
	stub := fake.StartStub
	fake.recordInvocation("Start", []interface{}{})
	fake.startMutex.Unlock()
	if stub != nil {
		fake.StartStub()
	}
}

func (fake *FakeParticipant) StartCallCount() int {
	fake.startMutex.RLock()
	defer fake.startMutex.RUnlock()
	return len(fake.startArgsForCall)
}

func (fake *FakeParticipant) StartCalls(stub func()) {
	fake.startMutex.Lock()
	defer fake.startMutex.Unlock()
	fake.StartStub = stub
}

func (fake *FakeParticipant) State() livekit.ParticipantInfo_State {
	fake.stateMutex.Lock()
	ret, specificReturn := fake.stateReturnsOnCall[len(fake.stateArgsForCall)]
	fake.stateArgsForCall = append(fake.stateArgsForCall, struct {
	}{})
	stub := fake.StateStub
	fakeReturns := fake.stateReturns
	fake.recordInvocation("State", []interface{}{})
	fake.stateMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeParticipant) StateCallCount() int {
	fake.stateMutex.RLock()
	defer fake.stateMutex.RUnlock()
	return len(fake.stateArgsForCall)
}

func (fake *FakeParticipant) StateCalls(stub func() livekit.ParticipantInfo_State) {
	fake.stateMutex.Lock()
	defer fake.stateMutex.Unlock()
	fake.StateStub = stub
}

func (fake *FakeParticipant) StateReturns(result1 livekit.ParticipantInfo_State) {
	fake.stateMutex.Lock()
	defer fake.stateMutex.Unlock()
	fake.StateStub = nil
	fake.stateReturns = struct {
		result1 livekit.ParticipantInfo_State
	}{result1}
}

func (fake *FakeParticipant) StateReturnsOnCall(i int, result1 livekit.ParticipantInfo_State) {
	fake.stateMutex.Lock()
	defer fake.stateMutex.Unlock()
	fake.StateStub = nil
	if fake.stateReturnsOnCall == nil {
		fake.stateReturnsOnCall = make(map[int]struct {
			result1 livekit.ParticipantInfo_State
		})
	}
	fake.stateReturnsOnCall[i] = struct {
		result1 livekit.ParticipantInfo_State
	}{result1}
}

func (fake *FakeParticipant) SubscriptionPermission() (*livekit.SubscriptionPermission, utils.TimedVersion) {
	fake.subscriptionPermissionMutex.Lock()
	ret, specificReturn := fake.subscriptionPermissionReturnsOnCall[len(fake.subscriptionPermissionArgsForCall)]
	fake.subscriptionPermissionArgsForCall = append(fake.subscriptionPermissionArgsForCall, struct {
	}{})
	stub := fake.SubscriptionPermissionStub
	fakeReturns := fake.subscriptionPermissionReturns
	fake.recordInvocation("SubscriptionPermission", []interface{}{})
	fake.subscriptionPermissionMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeParticipant) SubscriptionPermissionCallCount() int {
	fake.subscriptionPermissionMutex.RLock()
	defer fake.subscriptionPermissionMutex.RUnlock()
	return len(fake.subscriptionPermissionArgsForCall)
}

func (fake *FakeParticipant) SubscriptionPermissionCalls(stub func() (*livekit.SubscriptionPermission, utils.TimedVersion)) {
	fake.subscriptionPermissionMutex.Lock()
	defer fake.subscriptionPermissionMutex.Unlock()
	fake.SubscriptionPermissionStub = stub
}

func (fake *FakeParticipant) SubscriptionPermissionReturns(result1 *livekit.SubscriptionPermission, result2 utils.TimedVersion) {
	fake.subscriptionPermissionMutex.Lock()
	defer fake.subscriptionPermissionMutex.Unlock()
	fake.SubscriptionPermissionStub = nil
	fake.subscriptionPermissionReturns = struct {
		result1 *livekit.SubscriptionPermission
		result2 utils.TimedVersion
	}{result1, result2}
}

func (fake *FakeParticipant) SubscriptionPermissionReturnsOnCall(i int, result1 *livekit.SubscriptionPermission, result2 utils.TimedVersion) {
	fake.subscriptionPermissionMutex.Lock()
	defer fake.subscriptionPermissionMutex.Unlock()
	fake.SubscriptionPermissionStub = nil
	if fake.subscriptionPermissionReturnsOnCall == nil {
		fake.subscriptionPermissionReturnsOnCall = make(map[int]struct {
			result1 *livekit.SubscriptionPermission
			result2 utils.TimedVersion
		})
	}
	fake.subscriptionPermissionReturnsOnCall[i] = struct {
		result1 *livekit.SubscriptionPermission
		result2 utils.TimedVersion
	}{result1, result2}
}

func (fake *FakeParticipant) ToProto() *livekit.ParticipantInfo {
	fake.toProtoMutex.Lock()
	ret, specificReturn := fake.toProtoReturnsOnCall[len(fake.toProtoArgsForCall)]
	fake.toProtoArgsForCall = append(fake.toProtoArgsForCall, struct {
	}{})
	stub := fake.ToProtoStub
	fakeReturns := fake.toProtoReturns
	fake.recordInvocation("ToProto", []interface{}{})
	fake.toProtoMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeParticipant) ToProtoCallCount() int {
	fake.toProtoMutex.RLock()
	defer fake.toProtoMutex.RUnlock()
	return len(fake.toProtoArgsForCall)
}

func (fake *FakeParticipant) ToProtoCalls(stub func() *livekit.ParticipantInfo) {
	fake.toProtoMutex.Lock()
	defer fake.toProtoMutex.Unlock()
	fake.ToProtoStub = stub
}

func (fake *FakeParticipant) ToProtoReturns(result1 *livekit.ParticipantInfo) {
	fake.toProtoMutex.Lock()
	defer fake.toProtoMutex.Unlock()
	fake.ToProtoStub = nil
	fake.toProtoReturns = struct {
		result1 *livekit.ParticipantInfo
	}{result1}
}

func (fake *FakeParticipant) ToProtoReturnsOnCall(i int, result1 *livekit.ParticipantInfo) {
	fake.toProtoMutex.Lock()
	defer fake.toProtoMutex.Unlock()
	fake.ToProtoStub = nil
	if fake.toProtoReturnsOnCall == nil {
		fake.toProtoReturnsOnCall = make(map[int]struct {
			result1 *livekit.ParticipantInfo
		})
	}
	fake.toProtoReturnsOnCall[i] = struct {
		result1 *livekit.ParticipantInfo
	}{result1}
}

func (fake *FakeParticipant) UpdateSubscriptionPermission(arg1 *livekit.SubscriptionPermission, arg2 utils.TimedVersion, arg3 func(participantIdentity livekit.ParticipantIdentity) types.LocalParticipant, arg4 func(participantID livekit.ParticipantID) types.LocalParticipant) error {
	fake.updateSubscriptionPermissionMutex.Lock()
	ret, specificReturn := fake.updateSubscriptionPermissionReturnsOnCall[len(fake.updateSubscriptionPermissionArgsForCall)]
	fake.updateSubscriptionPermissionArgsForCall = append(fake.updateSubscriptionPermissionArgsForCall, struct {
		arg1 *livekit.SubscriptionPermission
		arg2 utils.TimedVersion
		arg3 func(participantIdentity livekit.ParticipantIdentity) types.LocalParticipant
		arg4 func(participantID livekit.ParticipantID) types.LocalParticipant
	}{arg1, arg2, arg3, arg4})
	stub := fake.UpdateSubscriptionPermissionStub
	fakeReturns := fake.updateSubscriptionPermissionReturns
	fake.recordInvocation("UpdateSubscriptionPermission", []interface{}{arg1, arg2, arg3, arg4})
	fake.updateSubscriptionPermissionMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3, arg4)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeParticipant) UpdateSubscriptionPermissionCallCount() int {
	fake.updateSubscriptionPermissionMutex.RLock()
	defer fake.updateSubscriptionPermissionMutex.RUnlock()
	return len(fake.updateSubscriptionPermissionArgsForCall)
}

func (fake *FakeParticipant) UpdateSubscriptionPermissionCalls(stub func(*livekit.SubscriptionPermission, utils.TimedVersion, func(participantIdentity livekit.ParticipantIdentity) types.LocalParticipant, func(participantID livekit.ParticipantID) types.LocalParticipant) error) {
	fake.updateSubscriptionPermissionMutex.Lock()
	defer fake.updateSubscriptionPermissionMutex.Unlock()
	fake.UpdateSubscriptionPermissionStub = stub
}

func (fake *FakeParticipant) UpdateSubscriptionPermissionArgsForCall(i int) (*livekit.SubscriptionPermission, utils.TimedVersion, func(participantIdentity livekit.ParticipantIdentity) types.LocalParticipant, func(participantID livekit.ParticipantID) types.LocalParticipant) {
	fake.updateSubscriptionPermissionMutex.RLock()
	defer fake.updateSubscriptionPermissionMutex.RUnlock()
	argsForCall := fake.updateSubscriptionPermissionArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3, argsForCall.arg4
}

func (fake *FakeParticipant) UpdateSubscriptionPermissionReturns(result1 error) {
	fake.updateSubscriptionPermissionMutex.Lock()
	defer fake.updateSubscriptionPermissionMutex.Unlock()
	fake.UpdateSubscriptionPermissionStub = nil
	fake.updateSubscriptionPermissionReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeParticipant) UpdateSubscriptionPermissionReturnsOnCall(i int, result1 error) {
	fake.updateSubscriptionPermissionMutex.Lock()
	defer fake.updateSubscriptionPermissionMutex.Unlock()
	fake.UpdateSubscriptionPermissionStub = nil
	if fake.updateSubscriptionPermissionReturnsOnCall == nil {
		fake.updateSubscriptionPermissionReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.updateSubscriptionPermissionReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeParticipant) UpdateVideoLayers(arg1 *livekit.UpdateVideoLayers) error {
	fake.updateVideoLayersMutex.Lock()
	ret, specificReturn := fake.updateVideoLayersReturnsOnCall[len(fake.updateVideoLayersArgsForCall)]
	fake.updateVideoLayersArgsForCall = append(fake.updateVideoLayersArgsForCall, struct {
		arg1 *livekit.UpdateVideoLayers
	}{arg1})
	stub := fake.UpdateVideoLayersStub
	fakeReturns := fake.updateVideoLayersReturns
	fake.recordInvocation("UpdateVideoLayers", []interface{}{arg1})
	fake.updateVideoLayersMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeParticipant) UpdateVideoLayersCallCount() int {
	fake.updateVideoLayersMutex.RLock()
	defer fake.updateVideoLayersMutex.RUnlock()
	return len(fake.updateVideoLayersArgsForCall)
}

func (fake *FakeParticipant) UpdateVideoLayersCalls(stub func(*livekit.UpdateVideoLayers) error) {
	fake.updateVideoLayersMutex.Lock()
	defer fake.updateVideoLayersMutex.Unlock()
	fake.UpdateVideoLayersStub = stub
}

func (fake *FakeParticipant) UpdateVideoLayersArgsForCall(i int) *livekit.UpdateVideoLayers {
	fake.updateVideoLayersMutex.RLock()
	defer fake.updateVideoLayersMutex.RUnlock()
	argsForCall := fake.updateVideoLayersArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeParticipant) UpdateVideoLayersReturns(result1 error) {
	fake.updateVideoLayersMutex.Lock()
	defer fake.updateVideoLayersMutex.Unlock()
	fake.UpdateVideoLayersStub = nil
	fake.updateVideoLayersReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeParticipant) UpdateVideoLayersReturnsOnCall(i int, result1 error) {
	fake.updateVideoLayersMutex.Lock()
	defer fake.updateVideoLayersMutex.Unlock()
	fake.UpdateVideoLayersStub = nil
	if fake.updateVideoLayersReturnsOnCall == nil {
		fake.updateVideoLayersReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.updateVideoLayersReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeParticipant) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.closeMutex.RLock()
	defer fake.closeMutex.RUnlock()
	fake.debugInfoMutex.RLock()
	defer fake.debugInfoMutex.RUnlock()
	fake.getPublishedTrackMutex.RLock()
	defer fake.getPublishedTrackMutex.RUnlock()
	fake.getPublishedTracksMutex.RLock()
	defer fake.getPublishedTracksMutex.RUnlock()
	fake.hasPermissionMutex.RLock()
	defer fake.hasPermissionMutex.RUnlock()
	fake.hiddenMutex.RLock()
	defer fake.hiddenMutex.RUnlock()
	fake.iDMutex.RLock()
	defer fake.iDMutex.RUnlock()
	fake.identityMutex.RLock()
	defer fake.identityMutex.RUnlock()
	fake.isPublisherMutex.RLock()
	defer fake.isPublisherMutex.RUnlock()
	fake.isRecorderMutex.RLock()
	defer fake.isRecorderMutex.RUnlock()
	fake.removePublishedTrackMutex.RLock()
	defer fake.removePublishedTrackMutex.RUnlock()
	fake.setMetadataMutex.RLock()
	defer fake.setMetadataMutex.RUnlock()
	fake.setNameMutex.RLock()
	defer fake.setNameMutex.RUnlock()
	fake.startMutex.RLock()
	defer fake.startMutex.RUnlock()
	fake.stateMutex.RLock()
	defer fake.stateMutex.RUnlock()
	fake.subscriptionPermissionMutex.RLock()
	defer fake.subscriptionPermissionMutex.RUnlock()
	fake.toProtoMutex.RLock()
	defer fake.toProtoMutex.RUnlock()
	fake.updateSubscriptionPermissionMutex.RLock()
	defer fake.updateSubscriptionPermissionMutex.RUnlock()
	fake.updateVideoLayersMutex.RLock()
	defer fake.updateVideoLayersMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeParticipant) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ types.Participant = new(FakeParticipant)
