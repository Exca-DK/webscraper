package clock

import (
	"testing"
	"time"
)

func TestRewindableClock(t *testing.T) {
	clock := NewRewindableClock()
	now := time.Now()
	before := now.Add(5 * time.Second * -1)
	clock.Rewind(before)
	if !clock.Now().Equal(before) {
		t.Fatal("rewind failure")
	}
	if !clock.Add(5 * time.Second).Equal(now) {
		t.Fatal("addition failure")
	}
	clock.Rewind(now)
	if clock.Since(before) != 5*time.Second {
		t.Fatal("unexpected duration")
	}
}

func TestClockSwapping(t *testing.T) {
	current := CurrentClock()
	defer SetClock(current)

	rewindable := NewRewindableClock()
	ts := time.Now()
	rewindable.Rewind(ts)

	SetClock(rewindable)
	if !CurrentClock().Now().Equal(ts) {
		t.Fatal("clock not set")
	}

	SetClock(current)
	if CurrentClock().Now().Equal(ts) {
		t.Fatal("clock not reset")
	}
}
