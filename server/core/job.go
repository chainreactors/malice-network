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
	jobID  int32 = 0
	ctrlID int32 = 0
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
func (j *jobs) Get(jobID int32) *Job {
	if jobID <= 0 {
		return nil
	}
	val, ok := j.Load(jobID)
	if ok {
		return val.(*Job)
	}
	return nil
}

type Job struct {
	ID           int32
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

func NextJobID() int32 {
	newID := jobID + 1
	jobID++
	return newID
}

func NextCtrlID() int32 {
	ctrlID++
	return ctrlID
}
