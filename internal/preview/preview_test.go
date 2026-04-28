package preview

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandlerServesRenderedRoute(t *testing.T) {
	handler, err := NewHandler(filepath.Join("..", "..", "testdata", "docs-main"))
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api-reference", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `<h1 id="api-reference">API Reference</h1>`) {
		t.Fatalf("missing rendered body: %s", body)
	}
	if !strings.Contains(body, `class="nav-tree"`) {
		t.Fatalf("missing nav shell: %s", body)
	}
}

func TestHandlerRejectsUnsupportedMethods(t *testing.T) {
	handler, err := NewHandler(filepath.Join("..", "..", "testdata", "docs-main"))
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api-reference", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestHandlerServesSearchEndpoint(t *testing.T) {
	handler, err := NewHandler(filepath.Join("..", "..", "testdata", "docs-main"))
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/_mintfast/search?q=wallet", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "/api-reference") {
		t.Fatalf("missing search result: %s", rec.Body.String())
	}
}

func TestHandlerServesEventsEndpoint(t *testing.T) {
	handler, err := NewHandler(filepath.Join("..", "..", "testdata", "docs-main"))
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/_mintfast/events", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec.Header().Get("content-type") != "text/event-stream" {
		t.Fatalf("content-type = %q", rec.Header().Get("content-type"))
	}
}

func TestRenderStaticCopiesAssets(t *testing.T) {
	out := t.TempDir()
	root := filepath.Join("..", "..", "testdata", "docs-main")
	if err := RenderStatic(root, out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "styles.css")); err != nil {
		t.Fatalf("expected styles.css copy: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "index.html")); err != nil {
		t.Fatalf("expected index render: %v", err)
	}
}

func TestSearchIndexDeduplicatesRepeatedRoutes(t *testing.T) {
	handler, err := NewHandler(filepath.Join("..", "..", "testdata", "docs-main"))
	if err != nil {
		t.Fatal(err)
	}
	results := handler.searchIndex.Search("Ledger")
	seen := map[string]bool{}
	for _, result := range results {
		if seen[result.Route] {
			t.Fatalf("duplicate route in search results: %s", result.Route)
		}
		seen[result.Route] = true
	}
}
