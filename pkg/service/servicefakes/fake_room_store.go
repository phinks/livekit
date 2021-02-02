// Code generated by counterfeiter. DO NOT EDIT.
package servicefakes

import (
	"sync"

	"github.com/livekit/livekit-server/pkg/service"
	"github.com/livekit/livekit-server/proto/livekit"
)

type FakeRoomStore struct {
	CreateRoomStub        func(*livekit.Room) error
	createRoomMutex       sync.RWMutex
	createRoomArgsForCall []struct {
		arg1 *livekit.Room
	}
	createRoomReturns struct {
		result1 error
	}
	createRoomReturnsOnCall map[int]struct {
		result1 error
	}
	DeleteRoomStub        func(string) error
	deleteRoomMutex       sync.RWMutex
	deleteRoomArgsForCall []struct {
		arg1 string
	}
	deleteRoomReturns struct {
		result1 error
	}
	deleteRoomReturnsOnCall map[int]struct {
		result1 error
	}
	GetRoomStub        func(string) (*livekit.Room, error)
	getRoomMutex       sync.RWMutex
	getRoomArgsForCall []struct {
		arg1 string
	}
	getRoomReturns struct {
		result1 *livekit.Room
		result2 error
	}
	getRoomReturnsOnCall map[int]struct {
		result1 *livekit.Room
		result2 error
	}
	ListRoomsStub        func() ([]*livekit.Room, error)
	listRoomsMutex       sync.RWMutex
	listRoomsArgsForCall []struct {
	}
	listRoomsReturns struct {
		result1 []*livekit.Room
		result2 error
	}
	listRoomsReturnsOnCall map[int]struct {
		result1 []*livekit.Room
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeRoomStore) CreateRoom(arg1 *livekit.Room) error {
	fake.createRoomMutex.Lock()
	ret, specificReturn := fake.createRoomReturnsOnCall[len(fake.createRoomArgsForCall)]
	fake.createRoomArgsForCall = append(fake.createRoomArgsForCall, struct {
		arg1 *livekit.Room
	}{arg1})
	stub := fake.CreateRoomStub
	fakeReturns := fake.createRoomReturns
	fake.recordInvocation("CreateRoom", []interface{}{arg1})
	fake.createRoomMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeRoomStore) CreateRoomCallCount() int {
	fake.createRoomMutex.RLock()
	defer fake.createRoomMutex.RUnlock()
	return len(fake.createRoomArgsForCall)
}

func (fake *FakeRoomStore) CreateRoomCalls(stub func(*livekit.Room) error) {
	fake.createRoomMutex.Lock()
	defer fake.createRoomMutex.Unlock()
	fake.CreateRoomStub = stub
}

func (fake *FakeRoomStore) CreateRoomArgsForCall(i int) *livekit.Room {
	fake.createRoomMutex.RLock()
	defer fake.createRoomMutex.RUnlock()
	argsForCall := fake.createRoomArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeRoomStore) CreateRoomReturns(result1 error) {
	fake.createRoomMutex.Lock()
	defer fake.createRoomMutex.Unlock()
	fake.CreateRoomStub = nil
	fake.createRoomReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeRoomStore) CreateRoomReturnsOnCall(i int, result1 error) {
	fake.createRoomMutex.Lock()
	defer fake.createRoomMutex.Unlock()
	fake.CreateRoomStub = nil
	if fake.createRoomReturnsOnCall == nil {
		fake.createRoomReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.createRoomReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeRoomStore) DeleteRoom(arg1 string) error {
	fake.deleteRoomMutex.Lock()
	ret, specificReturn := fake.deleteRoomReturnsOnCall[len(fake.deleteRoomArgsForCall)]
	fake.deleteRoomArgsForCall = append(fake.deleteRoomArgsForCall, struct {
		arg1 string
	}{arg1})
	stub := fake.DeleteRoomStub
	fakeReturns := fake.deleteRoomReturns
	fake.recordInvocation("DeleteRoom", []interface{}{arg1})
	fake.deleteRoomMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeRoomStore) DeleteRoomCallCount() int {
	fake.deleteRoomMutex.RLock()
	defer fake.deleteRoomMutex.RUnlock()
	return len(fake.deleteRoomArgsForCall)
}

func (fake *FakeRoomStore) DeleteRoomCalls(stub func(string) error) {
	fake.deleteRoomMutex.Lock()
	defer fake.deleteRoomMutex.Unlock()
	fake.DeleteRoomStub = stub
}

func (fake *FakeRoomStore) DeleteRoomArgsForCall(i int) string {
	fake.deleteRoomMutex.RLock()
	defer fake.deleteRoomMutex.RUnlock()
	argsForCall := fake.deleteRoomArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeRoomStore) DeleteRoomReturns(result1 error) {
	fake.deleteRoomMutex.Lock()
	defer fake.deleteRoomMutex.Unlock()
	fake.DeleteRoomStub = nil
	fake.deleteRoomReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeRoomStore) DeleteRoomReturnsOnCall(i int, result1 error) {
	fake.deleteRoomMutex.Lock()
	defer fake.deleteRoomMutex.Unlock()
	fake.DeleteRoomStub = nil
	if fake.deleteRoomReturnsOnCall == nil {
		fake.deleteRoomReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.deleteRoomReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeRoomStore) GetRoom(arg1 string) (*livekit.Room, error) {
	fake.getRoomMutex.Lock()
	ret, specificReturn := fake.getRoomReturnsOnCall[len(fake.getRoomArgsForCall)]
	fake.getRoomArgsForCall = append(fake.getRoomArgsForCall, struct {
		arg1 string
	}{arg1})
	stub := fake.GetRoomStub
	fakeReturns := fake.getRoomReturns
	fake.recordInvocation("GetRoom", []interface{}{arg1})
	fake.getRoomMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeRoomStore) GetRoomCallCount() int {
	fake.getRoomMutex.RLock()
	defer fake.getRoomMutex.RUnlock()
	return len(fake.getRoomArgsForCall)
}

func (fake *FakeRoomStore) GetRoomCalls(stub func(string) (*livekit.Room, error)) {
	fake.getRoomMutex.Lock()
	defer fake.getRoomMutex.Unlock()
	fake.GetRoomStub = stub
}

func (fake *FakeRoomStore) GetRoomArgsForCall(i int) string {
	fake.getRoomMutex.RLock()
	defer fake.getRoomMutex.RUnlock()
	argsForCall := fake.getRoomArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeRoomStore) GetRoomReturns(result1 *livekit.Room, result2 error) {
	fake.getRoomMutex.Lock()
	defer fake.getRoomMutex.Unlock()
	fake.GetRoomStub = nil
	fake.getRoomReturns = struct {
		result1 *livekit.Room
		result2 error
	}{result1, result2}
}

func (fake *FakeRoomStore) GetRoomReturnsOnCall(i int, result1 *livekit.Room, result2 error) {
	fake.getRoomMutex.Lock()
	defer fake.getRoomMutex.Unlock()
	fake.GetRoomStub = nil
	if fake.getRoomReturnsOnCall == nil {
		fake.getRoomReturnsOnCall = make(map[int]struct {
			result1 *livekit.Room
			result2 error
		})
	}
	fake.getRoomReturnsOnCall[i] = struct {
		result1 *livekit.Room
		result2 error
	}{result1, result2}
}

func (fake *FakeRoomStore) ListRooms() ([]*livekit.Room, error) {
	fake.listRoomsMutex.Lock()
	ret, specificReturn := fake.listRoomsReturnsOnCall[len(fake.listRoomsArgsForCall)]
	fake.listRoomsArgsForCall = append(fake.listRoomsArgsForCall, struct {
	}{})
	stub := fake.ListRoomsStub
	fakeReturns := fake.listRoomsReturns
	fake.recordInvocation("ListRooms", []interface{}{})
	fake.listRoomsMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeRoomStore) ListRoomsCallCount() int {
	fake.listRoomsMutex.RLock()
	defer fake.listRoomsMutex.RUnlock()
	return len(fake.listRoomsArgsForCall)
}

func (fake *FakeRoomStore) ListRoomsCalls(stub func() ([]*livekit.Room, error)) {
	fake.listRoomsMutex.Lock()
	defer fake.listRoomsMutex.Unlock()
	fake.ListRoomsStub = stub
}

func (fake *FakeRoomStore) ListRoomsReturns(result1 []*livekit.Room, result2 error) {
	fake.listRoomsMutex.Lock()
	defer fake.listRoomsMutex.Unlock()
	fake.ListRoomsStub = nil
	fake.listRoomsReturns = struct {
		result1 []*livekit.Room
		result2 error
	}{result1, result2}
}

func (fake *FakeRoomStore) ListRoomsReturnsOnCall(i int, result1 []*livekit.Room, result2 error) {
	fake.listRoomsMutex.Lock()
	defer fake.listRoomsMutex.Unlock()
	fake.ListRoomsStub = nil
	if fake.listRoomsReturnsOnCall == nil {
		fake.listRoomsReturnsOnCall = make(map[int]struct {
			result1 []*livekit.Room
			result2 error
		})
	}
	fake.listRoomsReturnsOnCall[i] = struct {
		result1 []*livekit.Room
		result2 error
	}{result1, result2}
}

func (fake *FakeRoomStore) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.createRoomMutex.RLock()
	defer fake.createRoomMutex.RUnlock()
	fake.deleteRoomMutex.RLock()
	defer fake.deleteRoomMutex.RUnlock()
	fake.getRoomMutex.RLock()
	defer fake.getRoomMutex.RUnlock()
	fake.listRoomsMutex.RLock()
	defer fake.listRoomsMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeRoomStore) recordInvocation(key string, args []interface{}) {
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

var _ service.RoomStore = new(FakeRoomStore)
