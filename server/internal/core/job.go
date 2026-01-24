package core

import (
	"fmt"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/types"
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

func jobKey(listenerID, name string) string {
	return listenerID + ":" + name
}

func (j *jobs) AddPipeline(pipe *clientpb.Pipeline) *Job {
	if pipe == nil {
		return &Job{ID: NextJobID()}
	}

	if pipe.ListenerId == "" || pipe.Name == "" {
		return &Job{ID: NextJobID(), Name: pipe.Name, Pipeline: pipe}
	}

	key := jobKey(pipe.ListenerId, pipe.Name)
	if val, ok := j.Load(key); ok && val != nil {
		existing := val.(*Job)
		existing.Pipeline = pipe
		existing.Name = pipe.Name
		Listeners.AddPipeline(pipe)
		return existing
	}

	job := &Job{
		ID:       NextJobID(),
		Name:     pipe.Name,
		Pipeline: pipe,
	}
	Listeners.AddPipeline(job.Pipeline)
	j.Store(key, job)
	return job
}

// Remove - Remove a job
func (j *jobs) Remove(listenerID, jobName string) {
	if listenerID == "" || jobName == "" {
		return
	}
	_, _ = j.LoadAndDelete(jobKey(listenerID, jobName))
}

// Get - Get a Job
func (j *jobs) Get(jobName string) (*Job, error) {
	if jobName == "" {
		return nil, types.ErrNotFoundPipeline
	}

	var matches []*Job
	for _, job := range j.All() {
		if job.Name == jobName {
			matches = append(matches, job)
		}
	}

	switch len(matches) {
	case 0:
		return nil, types.ErrNotFoundPipeline
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("multiple jobs found for %s, require listener_id", jobName)
	}
}

func (j *jobs) GetByListener(jobName, listenerID string) (*Job, error) {
	if jobName == "" {
		return nil, types.ErrNotFoundPipeline
	}
	if listenerID == "" {
		return nil, fmt.Errorf("listener_id required")
	}
	val, ok := j.Load(jobKey(listenerID, jobName))
	if ok && val != nil {
		return val.(*Job), nil
	}
	return nil, types.ErrNotFoundPipeline
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
