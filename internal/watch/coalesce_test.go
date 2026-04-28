package watch

import (
	"testing"
	"time"
)

func TestCoalesceMergesSamePathInsideWindow(t *testing.T) {
	now := time.Unix(10, 0)
	got := Coalesce([]Event{
		{Path: "a.mdx", At: now},
		{Path: "a.mdx", At: now.Add(20 * time.Millisecond)},
		{Path: "b.mdx", At: now.Add(30 * time.Millisecond)},
	}, 50*time.Millisecond)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2: %+v", len(got), got)
	}
}

func TestCoalesceKeepsEventsOutsideWindow(t *testing.T) {
	now := time.Unix(10, 0)
	got := Coalesce([]Event{
		{Path: "a.mdx", At: now},
		{Path: "a.mdx", At: now.Add(time.Second)},
	}, 50*time.Millisecond)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
}
