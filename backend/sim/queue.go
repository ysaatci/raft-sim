package sim

import "container/heap"

// Time is virtual simulation time in milliseconds.
type Time int64

// queue is a priority queue ordered by time, then by insertion order, so
// items scheduled for the same instant come out in a deterministic order.
type queue[T any] struct {
	items queueItems[T]
	seq   uint64
}

type queueItem[T any] struct {
	at    Time
	seq   uint64
	value T
}

func (q *queue[T]) push(at Time, v T) {
	heap.Push(&q.items, queueItem[T]{at: at, seq: q.seq, value: v})
	q.seq++
}

// popDue removes and returns the earliest item if it is due at or before now.
func (q *queue[T]) popDue(now Time) (T, bool) {
	if len(q.items) == 0 || q.items[0].at > now {
		var zero T
		return zero, false
	}
	return heap.Pop(&q.items).(queueItem[T]).value, true
}

func (q *queue[T]) len() int { return len(q.items) }

// each visits every item in unspecified order.
func (q *queue[T]) each(f func(at Time, v T)) {
	for _, it := range q.items {
		f(it.at, it.value)
	}
}

type queueItems[T any] []queueItem[T]

func (h queueItems[T]) Len() int { return len(h) }
func (h queueItems[T]) Less(i, j int) bool {
	if h[i].at != h[j].at {
		return h[i].at < h[j].at
	}
	return h[i].seq < h[j].seq
}
func (h queueItems[T]) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *queueItems[T]) Push(x any)   { *h = append(*h, x.(queueItem[T])) }
func (h *queueItems[T]) Pop() any {
	old := *h
	it := old[len(old)-1]
	*h = old[:len(old)-1]
	return it
}
