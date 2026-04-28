package assets

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func Resolve(root, requestPath string) (string, error) {
	clean := filepath.Clean("/" + requestPath)
	clean = strings.TrimPrefix(clean, "/")
	resolved := filepath.Join(root, clean)
	rel, err := filepath.Rel(root, resolved)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", errors.New("asset escapes root")
	}
	if info, err := os.Stat(resolved); err == nil && !info.IsDir() {
		return resolved, nil
	}
	return "", errors.New("asset not found")
}
