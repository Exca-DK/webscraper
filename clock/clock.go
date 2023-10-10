package clock

import (
	"sync/atomic"
	"time"
)

type Clock interface {
	Add(d time.Duration) time.Time
	Now() time.Time
	Since(time.Time) time.Duration
}

var (
	_ Clock = (realClock)(realClock{})
	_ Clock = (*RewindableClock)(nil)
)

type realClock struct{}

func (r realClock) Add(d time.Duration) time.Time { return r.Now().Add(d) }

func (realClock) Now() time.Time { return time.Now() }

func (realClock) Since(t time.Time) time.Duration { return time.Since(t) }

type RewindableClock struct {
	t atomic.Pointer[time.Time]
}

func (c *RewindableClock) Rewind(t time.Time) { c.t.Store(&t) }

func (c *RewindableClock) Add(d time.Duration) time.Time { return c.Now().Add(d) }

func (c *RewindableClock) Now() time.Time { return *c.t.Load() }

func (c *RewindableClock) Since(t time.Time) time.Duration { return c.Now().Sub(t) }

var clock Clock = realClock{}

func CurrentClock() Clock {
	return clock
}

func SetClock(c Clock) {
	clock = c
}

func Now() time.Time {
	return clock.Now()
}

func Since(t time.Time) time.Duration {
	return clock.Since(t)
}

func NewRewindableClock() *RewindableClock {
	c := &RewindableClock{}
	c.Rewind(time.Now())
	return c
}
