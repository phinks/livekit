package streamallocator

import (
	"sort"

	"github.com/livekit/protocol/livekit"
	"github.com/livekit/protocol/logger"

	"github.com/livekit/livekit-server/pkg/sfu"
	"github.com/livekit/livekit-server/pkg/sfu/buffer"
)

type Track struct {
	downTrack   *sfu.DownTrack
	source      livekit.TrackSource
	isSimulcast bool
	priority    uint8
	publisherID livekit.ParticipantID
	logger      logger.Logger

	maxLayer buffer.VideoLayer

	totalPackets       uint32
	totalRepeatedNacks uint32
	nackInfos map[uint16]sfu.NackInfo

	isDirty bool

	isPaused bool
}

func NewTrack(
	downTrack *sfu.DownTrack,
	source livekit.TrackSource,
	isSimulcast bool,
	publisherID livekit.ParticipantID,
	logger logger.Logger,
) *Track {
	t := &Track{
		downTrack:   downTrack,
		source:      source,
		isSimulcast: isSimulcast,
		publisherID: publisherID,
		logger:      logger,
		nackInfos: make(map[uint16]sfu.NackInfo),
		isPaused:    true,
	}
	t.SetPriority(0)
	t.SetMaxLayer(downTrack.MaxLayer())

	return t
}

func (t *Track) SetDirty(isDirty bool) bool {
	if t.isDirty == isDirty {
		return false
	}

	t.isDirty = isDirty
	return true
}

func (t *Track) SetPaused(isPaused bool) bool {
	if t.isPaused == isPaused {
		return false
	}

	t.isPaused = isPaused
	return true
}

func (t *Track) SetPriority(priority uint8) bool {
	if priority == 0 {
		switch t.source {
		case livekit.TrackSource_SCREEN_SHARE:
			priority = PriorityDefaultScreenshare
		default:
			priority = PriorityDefaultVideo
		}
	}

	if t.priority == priority {
		return false
	}

	t.priority = priority
	return true
}

func (t *Track) Priority() uint8 {
	return t.priority
}

func (t *Track) DownTrack() *sfu.DownTrack {
	return t.downTrack
}

func (t *Track) IsManaged() bool {
	return t.source != livekit.TrackSource_SCREEN_SHARE || t.isSimulcast
}

func (t *Track) ID() livekit.TrackID {
	return livekit.TrackID(t.downTrack.ID())
}

func (t *Track) PublisherID() livekit.ParticipantID {
	return t.publisherID
}

func (t *Track) SetMaxLayer(layer buffer.VideoLayer) bool {
	if t.maxLayer == layer {
		return false
	}

	t.maxLayer = layer
	return true
}

func (t *Track) WritePaddingRTP(bytesToSend int) int {
	return t.downTrack.WritePaddingRTP(bytesToSend, false)
}

func (t *Track) AllocateOptimal(allowOvershoot bool) sfu.VideoAllocation {
	return t.downTrack.AllocateOptimal(allowOvershoot)
}

func (t *Track) ProvisionalAllocatePrepare() {
	t.downTrack.ProvisionalAllocatePrepare()
}

func (t *Track) ProvisionalAllocate(availableChannelCapacity int64, layer buffer.VideoLayer, allowPause bool, allowOvershoot bool) int64 {
	return t.downTrack.ProvisionalAllocate(availableChannelCapacity, layer, allowPause, allowOvershoot)
}

func (t *Track) ProvisionalAllocateGetCooperativeTransition(allowOvershoot bool) sfu.VideoTransition {
	return t.downTrack.ProvisionalAllocateGetCooperativeTransition(allowOvershoot)
}

func (t *Track) ProvisionalAllocateGetBestWeightedTransition() sfu.VideoTransition {
	return t.downTrack.ProvisionalAllocateGetBestWeightedTransition()
}

func (t *Track) ProvisionalAllocateCommit() sfu.VideoAllocation {
	return t.downTrack.ProvisionalAllocateCommit()
}

func (t *Track) AllocateNextHigher(availableChannelCapacity int64, allowOvershoot bool) (sfu.VideoAllocation, bool) {
	return t.downTrack.AllocateNextHigher(availableChannelCapacity, allowOvershoot)
}

func (t *Track) GetNextHigherTransition(allowOvershoot bool) (sfu.VideoTransition, bool) {
	return t.downTrack.GetNextHigherTransition(allowOvershoot)
}

func (t *Track) Pause() sfu.VideoAllocation {
	return t.downTrack.Pause()
}

func (t *Track) IsDeficient() bool {
	return t.downTrack.IsDeficient()
}

func (t *Track) BandwidthRequested() int64 {
	return t.downTrack.BandwidthRequested()
}

func (t *Track) DistanceToDesired() float64 {
	return t.downTrack.DistanceToDesired()
}

func (t *Track) GetNackDelta() (uint32, uint32) {
	totalPackets, totalRepeatedNacks := t.downTrack.GetNackStats()

	packetDelta := totalPackets - t.totalPackets
	t.totalPackets = totalPackets

	nackDelta := totalRepeatedNacks - t.totalRepeatedNacks
	t.totalRepeatedNacks = totalRepeatedNacks

	return packetDelta, nackDelta
}

func (t *Track) UpdateNack(nackInfos []sfu.NackInfo) {
	for _, ni := range nackInfos {
		t.nackInfos[ni.SequenceNumber] = ni
	}
}

func (t *Track) GetAndResetNackStats() (lowest uint16, highest uint16, numNacked int, numNacks int, numRuns int) {
	if len(t.nackInfos) == 0 {
		return
	}

	sns := make([]uint16, 0, len(t.nackInfos))
	for _, ni := range t.nackInfos {
		if lowest == 0 || ni.SequenceNumber - lowest > (1 << 15) {
			lowest = ni.SequenceNumber
		}
		if highest == 0 || highest - ni.SequenceNumber > (1 << 15) {
			highest = ni.SequenceNumber
		}
		numNacks += int(ni.Attempts)
		sns = append(sns, ni.SequenceNumber)
	}
	numNacked = len(t.nackInfos)

	// find number of runs, i. e. bursts of contiguous sequence numbers NACKed, does not include isolated NACKs
	sort.Slice(sns, func(i, j int) bool {
		return (sns[i] - sns[j]) > (1 << 15)
	})

	rsn := sns[0]
	rsi := 0
	for i := 1; i < len(sns); i++ {
		if sns[i] == rsn + 1 {
			continue
		}

		if (i - rsi - 1) > 0 {
			numRuns++
		}

		rsn = sns[i]
		rsi = i
	}

	t.nackInfos = make(map[uint16]sfu.NackInfo)
	return
}

// ------------------------------------------------

type TrackSorter []*Track

func (t TrackSorter) Len() int {
	return len(t)
}

func (t TrackSorter) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t TrackSorter) Less(i, j int) bool {
	//
	// TrackSorter is used to allocate layer-by-layer.
	// So, higher priority track should come earlier so that it gets an earlier shot at each layer
	//
	if t[i].priority != t[j].priority {
		return t[i].priority > t[j].priority
	}

	if t[i].maxLayer.Spatial != t[j].maxLayer.Spatial {
		return t[i].maxLayer.Spatial > t[j].maxLayer.Spatial
	}

	return t[i].maxLayer.Temporal > t[j].maxLayer.Temporal
}

// ------------------------------------------------

type MaxDistanceSorter []*Track

func (m MaxDistanceSorter) Len() int {
	return len(m)
}

func (m MaxDistanceSorter) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

func (m MaxDistanceSorter) Less(i, j int) bool {
	//
	// MaxDistanceSorter is used to find a deficient track to use for probing during recovery from congestion.
	// So, higher priority track should come earlier so that they have a chance to recover sooner.
	//
	if m[i].priority != m[j].priority {
		return m[i].priority > m[j].priority
	}

	return m[i].DistanceToDesired() > m[j].DistanceToDesired()
}

// ------------------------------------------------

type MinDistanceSorter []*Track

func (m MinDistanceSorter) Len() int {
	return len(m)
}

func (m MinDistanceSorter) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

func (m MinDistanceSorter) Less(i, j int) bool {
	//
	// MinDistanceSorter is used to find excess bandwidth in cooperative allocation.
	// So, lower priority track should come earlier so that they contribute bandwidth to higher priority tracks.
	//
	if m[i].priority != m[j].priority {
		return m[i].priority < m[j].priority
	}

	return m[i].DistanceToDesired() < m[j].DistanceToDesired()
}

// ------------------------------------------------
