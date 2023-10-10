package prims

import "testing"

func TestQueue(t *testing.T) {
	queue := new(queue[string])
	queue.push("foo")
	queue.push("bar")
	queue.push("baz")

	a, _ := queue.pop()
	b, _ := queue.pop()
	c, _ := queue.pop()
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
