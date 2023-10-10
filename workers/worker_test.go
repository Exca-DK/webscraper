package workers

import (
	"context"
	"errors"
	"testing"
)

func TestWorker(t *testing.T) {
	t.Run("failing worker", func(t *testing.T) {
		worker := NewWorker(Job{exec: func(_ context.Context, _ string) error { return errors.New("foo") }, description: "bar"})
		stats := worker.Run(context.Background())
		if stats.JobError == nil {
			t.Fatal("worker stats should include error")
		}
	})

	t.Run("passing worker", func(t *testing.T) {
		worker := NewWorker(Job{exec: func(_ context.Context, _ string) error { return nil }, description: "bar"})
		stats := worker.Run(context.Background())
		if stats.JobError != nil {
			t.Fatal("worker stats should not include error")
		}
	})
}
