package test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/livekit/livekit-server/pkg/testutils"
	testclient "github.com/livekit/livekit-server/test/client"
)

func TestClientCouldConnect(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
		return
	}

	_, finish := setupSingleNodeTest("TestClientCouldConnect", testRoom)
	defer finish()

	c1 := createRTCClient("c1", defaultServerPort, nil)
	c2 := createRTCClient("c2", defaultServerPort, nil)
	waitUntilConnected(t, c1, c2)

	// ensure they both see each other
	testutils.WithTimeout(t, "c1 and c2 could connect", func() bool {
		if len(c1.RemoteParticipants()) == 0 || len(c2.RemoteParticipants()) == 0 {
			return false
		}
		//require.Equal()
		return true
	})
}

func TestSinglePublisher(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
		return
	}

	s, finish := setupSingleNodeTest("TestSinglePublisher", testRoom)
	defer finish()

	c1 := createRTCClient("c1", defaultServerPort, nil)
	c2 := createRTCClient("c2", defaultServerPort, nil)
	waitUntilConnected(t, c1, c2)

	// publish a track and ensure clients receive it ok
	t1, err := c1.AddStaticTrack("audio/opus", "audio", "webcam")
	require.NoError(t, err)
	defer t1.Stop()
	t2, err := c1.AddStaticTrack("video/vp8", "video", "webcam")
	require.NoError(t, err)
	defer t2.Stop()

	// a new client joins and should get the initial stream
	c3 := createRTCClient("c3", defaultServerPort, nil)

	success := testutils.WithTimeout(t, "c2 should receive two tracks", func() bool {
		if len(c2.SubscribedTracks()) == 0 {
			return false
		}
		// should have received two tracks
		if len(c2.SubscribedTracks()[c1.ID()]) != 2 {
			return false
		}

		tr1 := c2.SubscribedTracks()[c1.ID()][0]
		require.Equal(t, c1.ID(), tr1.StreamID())
		return true
	})
	if !success {
		t.FailNow()
	}

	// ensure that new client that has joined also received tracks
	waitUntilConnected(t, c3)
	success = testutils.WithTimeout(t, "c2 should receive two tracks", func() bool {
		if len(c3.SubscribedTracks()) == 0 {
			return false
		}
		// should have received two tracks
		if len(c3.SubscribedTracks()[c1.ID()]) != 2 {
			return false
		}
		return true
	})
	if !success {
		t.FailNow()
	}

	// ensure that the track ids are generated by server
	tracks := c3.SubscribedTracks()[c1.ID()]
	for _, tr := range tracks {
		require.True(t, strings.HasPrefix(tr.ID(), "TR_"), "track should begin with TR")
	}

	// when c3 disconnects.. ensure subscriber is cleaned up correctly
	c3.Stop()

	success = testutils.WithTimeout(t, "c3 is cleaned up as a subscriber", func() bool {
		room := s.RoomManager().GetRoom(context.Background(), testRoom)
		require.NotNil(t, room)

		p := room.GetParticipant("c1")
		require.NotNil(t, p)

		for _, t := range p.GetPublishedTracks() {
			if t.IsSubscriber(c3.ID()) {
				return false
			}
		}
		return true
	})
}

func TestAutoSubDisabled(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
		return
	}

	_, finish := setupSingleNodeTest("TestAutoSubDisabled", testRoom)
	defer finish()

	opts := testclient.Options{AutoSubscribe: false}
	c1 := createRTCClient("c1", defaultServerPort, &opts)
	c2 := createRTCClient("c2", defaultServerPort, &opts)
	defer c1.Stop()
	defer c2.Stop()
	waitUntilConnected(t, c1, c2)

	// c2 should not receive any tracks c1 publishes
	t1, err := c1.AddStaticTrack("audio/opus", "audio", "webcam")
	require.NoError(t, err)
	defer t1.Stop()

	time.Sleep(syncDelay)

	require.Empty(t, c2.SubscribedTracks()[c1.ID()])
}
