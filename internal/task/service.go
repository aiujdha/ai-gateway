// Package task describes asynchronous job state, without a worker or queue.
package task

import "context"

type Status string

const (
	Pending   Status = "pending"
	Running   Status = "running"
	Succeeded Status = "succeeded"
	Failed    Status = "failed"
	TimedOut  Status = "timed_out"
)

type Task struct {
	ID        string
	Model     string
	Status    Status
	ResultURL string
	Error     string
}

type Service interface {
	Submit(ctx context.Context, model, prompt string) (Task, error)
	Get(ctx context.Context, taskID string) (Task, error)
	// Transition reserves a conditional-update boundary; persistence is not implemented.
	Transition(ctx context.Context, taskID string, from, to Status) error
}
