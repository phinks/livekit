package videolayerselector

import (
	dd "github.com/livekit/livekit-server/pkg/sfu/dependencydescriptor"
	"github.com/livekit/protocol/logger"
)

type FrameChain struct {
	logger         logger.Logger
	decisions      *SelectorDecisionCache
	broken         bool
	chainIdx       int
	active         bool
	updatingActive bool

	expectFrames []uint64
}

func NewFrameChain(decisions *SelectorDecisionCache, chainIdx int, logger logger.Logger) *FrameChain {
	return &FrameChain{
		logger:    logger,
		decisions: decisions,
		broken:    true,
		chainIdx:  chainIdx,
		active:    false,
	}
}

func (fc *FrameChain) OnFrame(extFrameNum uint64, fd *dd.FrameDependencyTemplate) bool {
	if !fc.active {
		return false
	}

	// A decodable frame with frame_chain_fdiff equal to 0 indicates that the Chain is intact.
	if fd.ChainDiffs[fc.chainIdx] == 0 {
		if fc.broken {
			fc.broken = false
			fc.logger.Debugw("frame chain intact", "chanIdx", fc.chainIdx)
		}
		fc.expectFrames = fc.expectFrames[:0]
		return true
	}

	if fc.broken {
		return false
	}

	prevFrameInChain := extFrameNum - uint64(fd.ChainDiffs[fc.chainIdx])
	sd, err := fc.decisions.GetDecision(prevFrameInChain)
	if err != nil {
		fc.logger.Debugw("could not get decision", "err", err, "frame", extFrameNum, "prevFrame", prevFrameInChain)
	}

	var intact bool
	switch {
	case sd == selectorDecisionForwarded:
		intact = true

	case sd == selectorDecisionUnknown:
		intact = true
		fc.expectFrames = append(fc.expectFrames, prevFrameInChain)
		fc.decisions.ExpectDecision(prevFrameInChain, fc.OnExpectFrameChanged)
	}

	if !intact {
		fc.broken = true
		fc.logger.Debugw("frame chain broken", "chanIdx", fc.chainIdx, "sd", sd, "frame", extFrameNum, "prevFrame", prevFrameInChain)
	}
	return intact
}

func (fc *FrameChain) OnExpectFrameChanged(frameNum uint64, decision selectorDecision) {
	for i, f := range fc.expectFrames {
		if f == frameNum {
			if decision != selectorDecisionForwarded {
				fc.broken = true
			}
			fc.expectFrames[i] = fc.expectFrames[len(fc.expectFrames)-1]
			fc.expectFrames = fc.expectFrames[:len(fc.expectFrames)-1]
			break
		}
	}
}

func (fc *FrameChain) Broken() bool {
	return fc.broken
}

func (fc *FrameChain) BeginUpdateActive() {
	fc.updatingActive = false
}

func (fc *FrameChain) UpdateActive(active bool) {
	fc.updatingActive = fc.updatingActive || active
}

func (fc *FrameChain) EndUpdateActive() {
	active := fc.updatingActive
	fc.updatingActive = false

	if active == fc.active {
		return
	}

	// if the chain transit from inactive to active, reset broken to wait a decodable SWITCH frame
	if !fc.active {
		fc.broken = true
	}

	fc.active = active
}
