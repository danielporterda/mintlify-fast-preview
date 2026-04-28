package watch

import (
	"sort"
	"time"
)

type Event struct {
	Path string
	At   time.Time
}

func Coalesce(events []Event, window time.Duration) []Event {
	if len(events) == 0 {
		return nil
	}
	sort.Slice(events, func(i, j int) bool {
		if events[i].Path == events[j].Path {
			return events[i].At.Before(events[j].At)
		}
		return events[i].Path < events[j].Path
	})
	out := []Event{events[0]}
	for _, event := range events[1:] {
		last := &out[len(out)-1]
		if event.Path == last.Path && event.At.Sub(last.At) <= window {
			last.At = event.At
			continue
		}
		out = append(out, event)
	}
	return out
}
