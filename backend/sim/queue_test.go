package sim

import (
	"slices"
	"testing"
)

func drain(q *queue[string], now Time) []string {
	var out []string
	for {
		v, ok := q.popDue(now)
		if !ok {
			return out
		}
		out = append(out, v)
	}
}

func TestQueueOrdersByTimeThenInsertion(t *testing.T) {
	var q queue[string]
	q.push(30, "c")
	q.push(10, "a1")
	q.push(20, "b")
	q.push(10, "a2")
	q.push(10, "a3")
	if got := drain(&q, 100); !slices.Equal(got, []string{"a1", "a2", "a3", "b", "c"}) {
		t.Fatalf("order = %v", got)
	}
}

func TestQueuePopsOnlyDueItems(t *testing.T) {
	var q queue[string]
	q.push(5, "early")
	q.push(15, "late")
	if got := drain(&q, 10); !slices.Equal(got, []string{"early"}) {
		t.Fatalf("due at 10 = %v", got)
	}
	if q.len() != 1 {
		t.Fatalf("len = %d, want 1", q.len())
	}
	if got := drain(&q, 15); !slices.Equal(got, []string{"late"}) {
		t.Fatalf("due at 15 = %v", got)
	}
}
