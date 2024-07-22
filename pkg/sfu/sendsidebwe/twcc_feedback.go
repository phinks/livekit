package sendsidebwe

import (
	"errors"
	"sync"
	"time"

	"github.com/livekit/protocol/logger"
	"github.com/pion/rtcp"
)

// ------------------------------------------------------

const (
	outlierReportFactor            = 4
	estimatedFeedbackIntervalAlpha = float64(0.9)
)

// ------------------------------------------------------

var (
	errFeedbackReportOutOfOrder = errors.New("feedback report out-of-order")
)

// ------------------------------------------------------

type TWCCFeedbackInfo struct {
	BaseSN   uint16
	Arrivals []int64
}

// ------------------------------------------------------

type TWCCFeedback struct {
	logger logger.Logger

	lock                      sync.RWMutex
	lastFeedbackTime          time.Time
	estimatedFeedbackInterval time.Duration
	highestFeedbackCount      uint8
	// SSBWE-TODO- maybe store some history of reports as is?
}

func NewTWCCFeedback(logger logger.Logger) *TWCCFeedback {
	return &TWCCFeedback{
		logger: logger,
	}
}

func (t *TWCCFeedback) HandleRTCP(report *rtcp.TransportLayerCC) (*TWCCFeedbackInfo, error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	sinceLast := time.Duration(0)
	isInOrder := true
	now := time.Now()
	if !t.lastFeedbackTime.IsZero() {
		isInOrder = (report.FbPktCount - t.highestFeedbackCount) < (1 << 7)
		if isInOrder {
			sinceLast = now.Sub(t.lastFeedbackTime)
			if t.estimatedFeedbackInterval == 0 {
				t.estimatedFeedbackInterval = sinceLast
			} else {
				// filter out outliers from estimate
				if sinceLast > t.estimatedFeedbackInterval/outlierReportFactor && sinceLast < outlierReportFactor*t.estimatedFeedbackInterval {
					// smoothed version of inter feedback interval
					t.estimatedFeedbackInterval = time.Duration(estimatedFeedbackIntervalAlpha*float64(t.estimatedFeedbackInterval) + (1.0-estimatedFeedbackIntervalAlpha)*float64(sinceLast))
				}
			}
		}
	}
	if !isInOrder {
		return nil, errFeedbackReportOutOfOrder
	}

	t.lastFeedbackTime = now
	t.highestFeedbackCount = report.FbPktCount

	// reconstruct arrival times (at the remote end) of packets
	arrivals := make([]int64, report.PacketStatusCount)
	snIdx := 0
	deltaIdx := 0
	refTime := int64(report.ReferenceTime) * 64 * 1000 // in us
	for _, chunk := range report.PacketChunks {
		switch chunk := chunk.(type) {
		case *rtcp.RunLengthChunk:
			for i := uint16(0); i < chunk.RunLength; i++ {
				if chunk.PacketStatusSymbol != rtcp.TypeTCCPacketNotReceived {
					refTime += report.RecvDeltas[deltaIdx].Delta
					deltaIdx++

					arrivals[snIdx] = refTime
				}
				snIdx++
			}

		case *rtcp.StatusVectorChunk:
			for _, symbol := range chunk.SymbolList {
				if symbol != rtcp.TypeTCCPacketNotReceived {
					refTime += report.RecvDeltas[deltaIdx].Delta
					deltaIdx++

					arrivals[snIdx] = refTime
				}
				snIdx++
			}
		}
	}

	t.logger.Infow("TWCC feedback", "report", report.String()) // REMOVE
	return &TWCCFeedbackInfo{
		BaseSN:   report.BaseSequenceNumber,
		Arrivals: arrivals,
	}, nil
}

// ------------------------------------------------
