// Copyright 2023 Steven Bui

package jobworker

import (
	"fmt"
	"sync"
)

type JobWorker struct {
	mutex    sync.Mutex
	userJobs map[string][]*JobInfo
}

func NewJobWorker() *JobWorker {
	return &JobWorker{userJobs: make(map[string][]*JobInfo)}
}

func (jw *JobWorker) AddJob(username string, job *JobInfo) error {
	fmt.Printf("adding user %s job to map\n", username)
	jw.mutex.Lock()
	defer jw.mutex.Unlock()
	jobs, ok := jw.userJobs[username]
	if !ok {
		jw.userJobs[username] = []*JobInfo{job}
		return nil
	}

	jobs = append(jobs, job)
	jw.userJobs[username] = jobs
	fmt.Printf("total map length is %d\n", len(jw.userJobs))

	return nil
}

func (jw *JobWorker) FindJob(username string, jobID string) (*JobInfo, error) {
	fmt.Printf("total map length is %d\n", len(jw.userJobs))
	jobs := jw.userJobs[username]
	fmt.Printf("user job length is %d\n", len(jobs))
	for _, j := range jobs {
		if j.JobID == jobID {
			return j, nil
		}
	}
	return nil, fmt.Errorf("cannot find a job with id %s", jobID)
}
