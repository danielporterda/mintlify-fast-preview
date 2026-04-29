package preview

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type reloadHub struct {
	mu          sync.Mutex
	subscribers map[chan struct{}]struct{}
}

func newReloadHub() *reloadHub {
	return &reloadHub{subscribers: map[chan struct{}]struct{}{}}
}

func (h *reloadHub) subscribe() chan struct{} {
	ch := make(chan struct{}, 1)
	h.mu.Lock()
	h.subscribers[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *reloadHub) unsubscribe(ch chan struct{}) {
	h.mu.Lock()
	delete(h.subscribers, ch)
	close(ch)
	h.mu.Unlock()
}

func (h *reloadHub) broadcast() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subscribers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

type fileState struct {
	modTime time.Time
	size    int64
}

func watchForReloads(root string, onChange func(), interval time.Duration, stop <-chan struct{}) {
	previous := snapshot(root)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			next := snapshot(root)
			if changed(previous, next) {
				previous = next
				onChange()
			}
		case <-stop:
			return
		}
	}
}

func snapshot(root string) map[string]fileState {
	files := map[string]fileState{}
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !watchedPath(path) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		files[rel] = fileState{modTime: info.ModTime(), size: info.Size()}
		return nil
	})
	return files
}

func watchedPath(path string) bool {
	if filepath.Base(path) == "docs.json" {
		return true
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".mdx", ".css", ".js", ".json", ".yaml", ".yml":
		return true
	default:
		return false
	}
}

func changed(before, after map[string]fileState) bool {
	if len(before) != len(after) {
		return true
	}
	for path, oldState := range before {
		newState, ok := after[path]
		if !ok || oldState.size != newState.size || !oldState.modTime.Equal(newState.modTime) {
			return true
		}
	}
	return false
}
