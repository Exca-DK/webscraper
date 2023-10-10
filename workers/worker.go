package workers

import (
	"context"
	"time"
)

var empty Worker = Worker{createdAt: time.Now(), job: Job{
	exec:        func(ctx context.Context, id string) error { return nil },
	description: "",
}}

// Job represents a task or job that can be executed.
// It includes an execution method and a description of the job.
type Job struct {
	exec        func(ctx context.Context, id string) error
	description string
}

// NewJob creates a new Job with the provided job description and execution method.
func NewJob(description string, method func(ctx context.Context, id string) error) Job {
	return Job{exec: method, description: description}
}

// JobStats represents statistics and information about a job execution.
type JobStats struct {
	JobDescription string
	JobError       error
	CreatedAt      time.Time
	StartedAt      time.Time
	FinishedAt     time.Time
}

// Worker represents a worker that executes a job.
// It includes information about when the worker was created and the job it is assigned to.
type Worker struct {
	createdAt time.Time
	job       Job
}

func NewWorker(job Job) Worker {
	return Worker{createdAt: time.Now(), job: job}
}

func emptyWorker() Worker {
	return empty
}

// Run executes the assigned job with the provided context.
// It returns JobStats with information about the job execution.
func (w Worker) Run(ctx context.Context) JobStats {
	startedAt := time.Now()
	err := w.job.exec(ctx, w.job.description)
	return JobStats{
		JobDescription: w.job.description,
		JobError:       err,
		CreatedAt:      w.createdAt,
		StartedAt:      startedAt,
		FinishedAt:     time.Now(),
	}
}
