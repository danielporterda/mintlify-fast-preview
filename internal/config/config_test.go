package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCleanRoute(t *testing.T) {
	tests := map[string]string{
		"index":                    "/",
		"/index.mdx":               "/",
		"reference/protobuf/index": "/reference/protobuf",
		"api-reference.mdx":        "/api-reference",
		"/a/b.md":                  "/a/b",
	}
	for input, want := range tests {
		if got := CleanRoute(input); got != want {
			t.Fatalf("CleanRoute(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestLoadAndFlattenPages(t *testing.T) {
	root := t.TempDir()
	err := os.WriteFile(filepath.Join(root, "docs.json"), []byte(`{
	  "name": "Docs",
	  "colors": {"primary":"#111"},
	  "navigation": {
	    "dropdowns": [{
	      "dropdown": "API Reference",
	      "versions": [{
	        "version": "MainNet",
	        "groups": [{"group":"Ledger API","pages":["reference/protobuf/index","reference/protobuf/operations/status"]}]
	      }]
	    }]
	  }
	}`), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	docs, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	pages := docs.FlattenPages()
	if len(pages) != 2 {
		t.Fatalf("len(pages) = %d, want 2", len(pages))
	}
	if pages[0].Dropdown != "API Reference" || pages[0].Version != "MainNet" || pages[0].Group != "Ledger API" {
		t.Fatalf("unexpected nav context: %+v", pages[0])
	}
	if pages[0].Route != "/reference/protobuf" {
		t.Fatalf("route = %q", pages[0].Route)
	}
}

func TestFlattenNestedPageGroups(t *testing.T) {
	root := t.TempDir()
	err := os.WriteFile(filepath.Join(root, "docs.json"), []byte(`{
	  "navigation": {
	    "groups": [{
	      "group": "Guides",
	      "pages": [
	        "guides/start",
	        {"group":"Nested","pages":["guides/deep"]}
	      ]
	    }]
	  }
	}`), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	docs, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	pages := docs.FlattenPages()
	if len(pages) != 2 {
		t.Fatalf("len(pages) = %d, want 2", len(pages))
	}
	if pages[0].Group != "Guides" || pages[1].Group != "Nested" {
		t.Fatalf("unexpected groups: %+v", pages)
	}
}
