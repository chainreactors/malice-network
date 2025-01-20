package core

import (
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"sync"
)

var (
	Jobs = &jobs{
		Map: &sync.Map{},
	}
	jobID  uint32 = 0
	ctrlID uint32 = 0
)

type jobs struct {
	*sync.Map
}

func (j *jobs) AddPipeline(pipe *clientpb.Pipeline) *Job {
	job := &Job{
		ID:       NextJobID(),
		Name:     pipe.Name,
		Pipeline: pipe,
	}
	j.Add(job)
	return job
}

func (j *jobs) Add(job *Job) {
	Listeners.AddPipeline(job.Pipeline)
	j.Store(job.ID, job)
}

// Remove - Remove a job
func (j *jobs) Remove(job *Job) {
	_, _ = j.LoadAndDelete(job.ID)
	//if ok {
	//	EventBroker.Publish(Event{
	//		Job:       job,
	//		EventType: consts.JobStoppedEvent,
	//	})
	//}
}

// Get - Get a Job
func (j *jobs) Get(jobName string) (*Job, error) {
	if jobName == "" {
		return nil, errs.ErrNotFoundPipeline
	}
	for _, job := range j.All() {
		if job.Name == jobName {
			return job, nil
		}
	}
	return nil, errs.ErrNotFoundPipeline
}

func (j *jobs) All() []*Job {
	jobs := []*Job{}
	j.Range(func(key, value interface{}) bool {
		jobs = append(jobs, value.(*Job))
		return true
	})
	return jobs
}

type Job struct {
	ID       uint32
	Name     string
	Pipeline *clientpb.Pipeline
}

func (j *Job) ToProtobuf() *clientpb.Job {
	job := &clientpb.Job{
		Id:       j.ID,
		Name:     j.Name,
		Pipeline: j.Pipeline,
	}
	return job
}

func CurrentJobID() uint32 {
	return jobID
}

func NextJobID() uint32 {
	jobID++
	return jobID
}

func NextCtrlID() uint32 {
	ctrlID++
	return ctrlID
}
