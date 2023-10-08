package core

import "sync"

var (
	Jobs  = &jobs{&sync.Map{}}
	jobID = 0
)

type jobs struct {
	*sync.Map
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
func (j *jobs) Get(jobID int) *Job {
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
	ID           int
	Name         string
	Description  string
	Protocol     string
	Host         string
	Port         uint16
	Domains      []string
	JobCtrl      chan bool
	PersistentID string
}

func NextJobID() int {
	newID := jobID + 1
	jobID++
	return newID
}
