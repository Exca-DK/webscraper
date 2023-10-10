package prims

type queue[T any] []T

func (q *queue[T]) push(elem T) {
	*q = append(*q, elem)
}

func (q *queue[T]) peek() (T, bool) {
	var t T
	if len(*q) == 0 {
		return t, false
	}
	t = (*q)[0]
	return t, true
}

func (q *queue[T]) pop() (T, bool) {
	elem, ok := q.peek()
	q.move()
	return elem, ok
}

func (q *queue[T]) move() {
	if len(*q) == 0 {
		return
	}
	*q = (*q)[1:]
}
