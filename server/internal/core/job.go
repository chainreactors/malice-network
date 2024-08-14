package core

import (
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"google.golang.org/protobuf/proto"
	"sync"
)

var (
	Jobs = &jobs{
		Map:  &sync.Map{},
		Ctrl: make(chan *clientpb.JobCtrl),
	}
	jobID  uint32 = 0
	ctrlID uint32 = 0
)

type jobs struct {
	*sync.Map
	Ctrl chan *clientpb.JobCtrl
}

func (j *jobs) Add(job *Job) {
	j.Store(job.ID, job)
	//EventBroker.Publish(Event{
	//	Job:       job,
	//	EventType: consts.JobStartedEvent,
	//})
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
func (j *jobs) Get(jobID uint32) *Job {
	if jobID <= 0 {
		return nil
	}
	val, ok := j.Load(jobID)
	if ok {
		return val.(*Job)
	}
	return nil
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
	ID           uint32
	Message      proto.Message
	JobCtrl      chan bool
	PersistentID string
}

func (j *Job) ToProtobuf() *clientpb.Job {
	job := &clientpb.Job{
		Id: j.ID,
	}
	switch j.Message.(type) {
	case *lispb.Pipeline:
		job.Pipeline = j.Message.(*lispb.Pipeline)
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
