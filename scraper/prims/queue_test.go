package prims

import "testing"

func TestQueue(t *testing.T) {
	queue := new(Queue[string])
	queue.Push("foo")
	queue.Push("bar")
	queue.Push("baz")

	a, _ := queue.Pop()
	b, _ := queue.Pop()
	c, _ := queue.Pop()
	if a != "foo" {
		t.Fatalf("unexpected item. got %v, want %v", a, "foo")
	}
	if b != "bar" {
		t.Fatalf("unexpected item. got %v, want %v", b, "bar")
	}
	if c != "baz" {
		t.Fatalf("unexpected item. got %v, want %v", c, "baz")
	}
}
