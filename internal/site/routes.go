package site

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/danielporterda/mintlify-fast-preview/internal/config"
)

type Route struct {
	URL      string
	FilePath string
	Page     config.Page
}

func BuildRoutes(root string, docs *config.Docs) ([]Route, error) {
	var routes []Route
	for _, page := range docs.FlattenPages() {
		file, err := ResolvePage(root, page.Path)
		if err != nil {
			return nil, err
		}
		routes = append(routes, Route{URL: page.Route, FilePath: file, Page: page})
	}
	return routes, nil
}

func ResolvePage(root, ref string) (string, error) {
	clean := strings.TrimPrefix(ref, "/")
	candidates := []string{
		filepath.Join(root, clean),
		filepath.Join(root, clean+".mdx"),
		filepath.Join(root, clean+".md"),
		filepath.Join(root, clean, "index.mdx"),
		filepath.Join(root, clean, "index.md"),
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", errors.New("page not found: " + ref)
}
