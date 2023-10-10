package prims

// Queue[T] is a generic data structure that represents a queue, allowing elements of type T to be
// enqueued, dequeued, and peeked. It maintains a first-in, first-out (FIFO) order for elements.
type Queue[T any] []T

// Push adds an element to the end of the queue.
func (q *Queue[T]) Push(elem T) {
	*q = append(*q, elem)
}

// Peek retrieves the element at the front of the queue without removing it.
func (q *Queue[T]) Peek() (T, bool) {
	var t T
	if len(*q) == 0 {
		return t, false
	}
	t = (*q)[0]
	return t, true
}

// Pop retrieves and removes the element at the front of the queue.
func (q *Queue[T]) Pop() (T, bool) {
	elem, ok := q.Peek()
	q.move()
	return elem, ok
}

// move advances the position of the front element in the queue by one, effectively removing it.
// If the queue is empty, no action is taken.
func (q *Queue[T]) move() {
	if len(*q) == 0 {
		return
	}
	*q = (*q)[1:]
}
