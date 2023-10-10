package prims

import (
	"testing"
	"time"

	"github.com/Exca-DK/webscraper/clock"
)

// TestEvictableCache tests the functionality of the SimpleEvictableCache by adding items with different
// expiration times, evicting them in the expected order, and verifying the cache's behavior.
func TestEvictableCache(t *testing.T) {
	var (
		items     = 4
		durations = make([]time.Duration, 0, items)
		evictedCh = make(chan int)
	)

	cache := NewSimpleEvictableCache[int, struct{}](func(i int, _ struct{}) {
		go func() { evictedCh <- i }()
	})

	for i := items; i > 0; i-- {
		durations = append(durations, time.Duration(i)*time.Second)
	}

	// use clock for time manipulation
	current := clock.CurrentClock()
	defer clock.SetClock(current)
	testingClock := clock.NewRewindableClock()
	clock.SetClock(testingClock)

	// check that items are succesfully added
	for i := 0; i < items; i++ {
		if !cache.AddIfNotSeen(i, struct{}{}, clock.CurrentClock().Add(durations[i])) {
			t.Fatal("failed adding first seen elem")
		}
	}

	// check that eviction is in proper order
	for i := 0; i < items; i++ {
		ts := clock.CurrentClock().Now()
		testingClock.Rewind(ts.Add(durations[len(durations)-i-1] + 1))
		cache.Evict()
		evicted := <-evictedCh
		if evicted != items-i-1 {
			t.Fatal("invalid item evicted", "evicted", evicted, "expected", items-i-1)
		}
		testingClock.Rewind(ts)
	}

	// check that its empty
	for i := 0; i < items; i++ {
		if cache.Seen(i) {
			t.Log("evicted item still seen")
		}
	}

	// check that you can readd them
	for i := 0; i < items; i++ {
		if !cache.AddIfNotSeen(i, struct{}{}, clock.CurrentClock().Add(1*time.Second*-1)) {
			t.Fatal("failed adding first seen elem")
		}
	}

	// check that multi items can be evicted at once
	cache.Evict()
	for i := 0; i < items; i++ {
		<-evictedCh
	}
}
