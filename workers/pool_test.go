package workers

import (
	"context"
	"testing"
	"time"
)

func testFunc(ctx context.Context, id string) error {
	time.Sleep(time.Second)
	return nil
}

func TestPool(t *testing.T) {
	t.Run("stopped", func(t *testing.T) {
		var (
			ch         = make(chan JobStats)
			testWorker = NewWorker(Job{exec: testFunc, description: "test"})
		)

		pool := NewWorkPool(ch)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		cancel()

		pool.Start(ctx)
		for i := 0; i < 3; i++ {
			pool.AddWorker(testWorker)
		}

		var done int
		for range ch {
			done++
		}
		if done != 0 {
			t.Fatalf("unexpected worker execution amount. expected: %v, got %v", 0, done)
		}
	})

	t.Run("default threads", func(t *testing.T) {
		var (
			ch         = make(chan JobStats)
			times      = 3
			testWorker = NewWorker(Job{exec: testFunc, description: "test"})
		)

		pool := NewWorkPool(ch)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		pool.Start(ctx)
		for i := 0; i < times; i++ {
			pool.AddWorker(testWorker)
		}

		var done int
		for range ch {
			done++
		}
		if done != times {
			t.Fatalf("unexpected worker execution amount. expected: %v, got %v", times, done)
		}
	})

	t.Run("concurrency", func(t *testing.T) {
		var (
			times      = 3
			testWorker = NewWorker(Job{exec: testFunc, description: "test"})
		)
		previousDuration := 1 * time.Hour // some big value for initialization
		for i := 1; i < 3; i++ {
			ch := make(chan JobStats)
			pool := NewWorkPool(ch)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			pool.WithThreads(uint32(i)).Start(ctx)
			for i := 0; i < times; i++ {
				pool.AddWorker(testWorker)
			}
			var (
				done  int
				start = time.Now()
			)
			for range ch {
				done++
				if done == times {
					cancel()
				}
			}
			duration := time.Since(start)
			if duration > previousDuration {
				t.Fatalf("worker with higher threads amount executed slower. exec time: %v, prev: %v. threads: %v", duration, previousDuration, i)
			}
		}
	})
}
