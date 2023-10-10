package prims

type Queue[T any] []T

func (q *Queue[T]) Push(elem T) {
	*q = append(*q, elem)
}

func (q *Queue[T]) Peek() (T, bool) {
	var t T
	if len(*q) == 0 {
		return t, false
	}
	t = (*q)[0]
	return t, true
}

func (q *Queue[T]) Pop() (T, bool) {
	elem, ok := q.Peek()
	q.move()
	return elem, ok
}

func (q *Queue[T]) move() {
	if len(*q) == 0 {
		return
	}
	*q = (*q)[1:]
}
