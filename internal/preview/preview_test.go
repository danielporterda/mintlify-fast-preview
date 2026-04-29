package preview

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/danielporterda/mintlify-fast-preview/internal/config"
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

func TestNavHTMLRendersNestedGroups(t *testing.T) {
	got := navHTML(&config.Docs{
		Navigation: config.Navigation{
			Groups: []config.Group{{
				Group: "Guides",
				Pages: []config.PageEntry{
					{Path: "guides/start"},
					{Group: "Nested", Pages: []config.PageEntry{{Path: "guides/deep"}}},
				},
			}},
		},
	})
	for _, want := range []string{
		`<a href="/guides/start">Start</a>`,
		`<h5>Nested</h5>`,
		`<a href="/guides/deep">Deep</a>`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %s", want, got)
		}
	}
}

func TestHandlerIncludesCopyButtonScript(t *testing.T) {
	handler, err := NewHandler(filepath.Join("..", "..", "testdata", "docs-main"))
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/reference/protobuf/operations/com-digitalasset-canton-admin-mediator-v30/mediatorstatusservice/mediatorstatus", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `class="mintfast-copy"`) || !strings.Contains(body, `navigator.clipboard.writeText`) {
		t.Fatalf("missing copy button behavior: %s", body)
	}
}

func TestHandlerIncludesDarkModeControls(t *testing.T) {
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
	for _, want := range []string{
		`class="mintfast-theme-toggle"`,
		`localStorage.getItem('mintfast-theme')`,
		`html[data-theme=dark] body`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %s", want, body)
		}
	}
}

func TestHandlerRendersRightRailTOC(t *testing.T) {
	handler, err := NewHandler(filepath.Join("..", "..", "testdata", "docs-main"))
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/reference/protobuf/operations/com-digitalasset-canton-admin-mediator-v30/mediatorstatusservice/mediatorstatus", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `<aside class="toc"><strong>On this page</strong><a href="#request" class="toc-level-2">Request</a>`) {
		t.Fatalf("missing right rail toc: %s", body)
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
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/_mintfast/events", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(rec, req)
		close(done)
	}()
	time.Sleep(10 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("events endpoint did not close after context cancellation")
	}
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

func TestSearchIndexIncludesResolvedSnippetContent(t *testing.T) {
	handler, err := NewHandler(filepath.Join("..", "..", "testdata", "docs-main"))
	if err != nil {
		t.Fatal(err)
	}
	results := handler.searchIndex.Search("readyz")
	found := false
	for _, result := range results {
		if result.Route == "/global-synchronizer/troubleshooting-guide/common-questions" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected snippet-backed page in search results: %+v", results)
	}
}

func TestHandlerRefreshesSearchIndexAfterFileChange(t *testing.T) {
	root := writeDocsFixture(t)
	handler, err := NewHandler(root)
	if err != nil {
		t.Fatal(err)
	}
	if results := handler.searchIndex.Search("needle"); len(results) != 0 {
		t.Fatalf("unexpected initial search result: %+v", results)
	}
	if err := os.WriteFile(filepath.Join(root, "index.mdx"), []byte("---\ntitle: Home\n---\n# Home\nneedle"), 0o644); err != nil {
		t.Fatal(err)
	}
	handler.handleFileChange()
	if results := handler.searchIndex.Search("needle"); len(results) == 0 || results[0].Route != "/" {
		t.Fatalf("missing refreshed search result: %+v", results)
	}
}

func TestHandlerRefreshesRoutesAfterDocsJSONChange(t *testing.T) {
	root := writeDocsFixture(t)
	handler, err := NewHandler(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "new-page.mdx"), []byte("---\ntitle: New Page\n---\n# New Page"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs.json"), []byte(`{"name":"Docs","navigation":{"pages":["index","new-page"]}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	handler.handleFileChange()
	req := httptest.NewRequest(http.MethodGet, "/new-page", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `<h1 id="new-page">New Page</h1>`) {
		t.Fatalf("missing new page render: %s", rec.Body.String())
	}
}

func TestReloadHubBroadcastsToSubscribers(t *testing.T) {
	hub := newReloadHub()
	sub := hub.subscribe()
	defer hub.unsubscribe(sub)
	hub.broadcast()
	select {
	case <-sub:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for reload broadcast")
	}
}

func writeDocsFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "docs.json"), []byte(`{"name":"Docs","navigation":{"pages":["index"]}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "index.mdx"), []byte("---\ntitle: Home\n---\n# Home\ninitial"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestSnapshotDetectsWatchedFileChanges(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "index.mdx")
	if err := os.WriteFile(file, []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := snapshot(root)
	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(file, []byte("two"), 0o644); err != nil {
		t.Fatal(err)
	}
	after := snapshot(root)
	if !changed(before, after) {
		t.Fatal("expected changed snapshot")
	}
}
