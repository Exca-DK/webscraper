package prims

import (
	"fmt"
	"sort"
	"time"

	"github.com/Exca-DK/webscraper/clock"
)

// expirableItem represents an item with an associated expiration time.
type expirableItem[T any] struct {
	V        T
	deadline time.Time
}

func (i expirableItem[T]) String() string {
	return fmt.Sprintf("(item:%v,deadline:%s)", i.V, i.deadline.Format(time.StampNano))
}

// SimpleEvictableCache is a generic cache that supports automatic eviction of items based on their expiration times.
// It allows you to add items, mark them for eviction, and retrieve items based on their keys.
type SimpleEvictableCache[T comparable, Y any] struct {
	m          map[T]expirableItem[Y]
	q          Queue[expirableItem[T]]
	onEviction func(T, Y)
}

// NewSimpleEvictableCache creates a new instance of SimpleEvictableCache with a specified eviction callback.
func NewSimpleEvictableCache[T comparable, Y any](onEviction func(T, Y)) *SimpleEvictableCache[T, Y] {
	return &SimpleEvictableCache[T, Y]{
		m:          make(map[T]expirableItem[Y]),
		q:          make(Queue[expirableItem[T]], 0),
		onEviction: onEviction,
	}
}

// AddIfNotSeen adds an item to the cache if it hasn't been seen before, based on the provided key.
func (e *SimpleEvictableCache[T, Y]) AddIfNotSeen(key T, value Y, deadline time.Time) bool {
	if _, ok := e.m[key]; ok {
		return false
	}
	e.m[key] = expirableItem[Y]{
		V:        value,
		deadline: deadline,
	}
	// if it can't be evicted, dont bother
	if !deadline.IsZero() {
		e.mark(key, deadline)
	}
	// always try to evict
	e.tryEvict()
	return true
}

// Seen checks if an item with the specified key has been seen in the cache.
func (e *SimpleEvictableCache[T, Y]) Seen(key T) bool {
	_, ok := e.m[key]
	e.tryEvict()
	return ok
}

// Evict manually triggers the eviction process for the cache, removing expired items.
func (e *SimpleEvictableCache[T, Y]) Evict() {
	e.tryEvict()
}

// EvictSize returns the size of evictable items.
func (e *SimpleEvictableCache[T, Y]) EvictSize() int {
	return len(e.q)
}

func (e *SimpleEvictableCache[T, Y]) tryEvict() {
	for {
		keyElem, ok := e.q.Peek()
		if !ok {
			return
		}

		if !clock.CurrentClock().Now().After(keyElem.deadline) {
			return
		}

		e.q.Pop()
		valueElem := e.m[keyElem.V]
		delete(e.m, keyElem.V)
		e.evict(keyElem.V, valueElem.V)
	}
}

func (e *SimpleEvictableCache[T, Y]) evict(k T, v Y) {
	if e.onEviction != nil {
		e.onEviction(k, v)
	}
}

func (e *SimpleEvictableCache[T, Y]) mark(key T, deadline time.Time) {
	e.q.Push(expirableItem[T]{
		V:        key,
		deadline: deadline,
	})
	sort.SliceStable(e.q, func(i, j int) bool {
		a, b := e.q[i], e.q[j]
		return !a.deadline.After(b.deadline)
	})
}
