package agent

import (
	"sync"

	"github.com/livekit/protocol/livekit"
	"github.com/livekit/protocol/logger"
)

// Represents a job that is being executed by a worker
type Job struct {
	id        string
	jobType   livekit.JobType
	status    livekit.JobStatus
	namespace string

	mu   sync.Mutex
	load float32

	Logger logger.Logger
}

func NewJob(id, namespace string, jobType livekit.JobType) *Job {
	return &Job{
		id:        id,
		status:    livekit.JobStatus_JS_UNKNOWN,
		jobType:   jobType,
		namespace: namespace,
	}
}

func (j *Job) ID() string {
	return j.id
}

func (j *Job) Namespace() string {
	return j.namespace
}

func (j *Job) Type() livekit.JobType {
	return j.jobType
}

func (j *Job) WorkerLoad() float32 {
	// Current load that this job is taking on its worker
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.load
}

func (j *Job) UpdateStatus(req *livekit.UpdateJobStatus) {
	j.mu.Lock()

	if req.Status != nil {
		j.status = *req.Status // End of the job, SUCCESS or FAILURE

		if j.status == livekit.JobStatus_JS_FAILED {
			j.Logger.Errorw("job failed", nil, "id", j.id, "type", j.jobType, "error", req.Error)
		}
	}

	j.load = req.Load
	j.mu.Unlock()

	if req.Metadata != nil {
		j.UpdateMetadata(req.GetMetadata())
	}
}

func (j *Job) UpdateMetadata(metadata string) {
	j.Logger.Debugw("job metadata", nil, "id", j.id, "metadata", metadata)
}
