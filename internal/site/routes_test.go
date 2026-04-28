package site

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/danielporterda/mintlify-fast-preview/internal/config"
)

func TestResolvePageExtensionless(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "reference"), 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "reference", "protobuf.mdx")
	if err := os.WriteFile(file, []byte("# Protobuf"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ResolvePage(root, "reference/protobuf")
	if err != nil {
		t.Fatal(err)
	}
	if got != file {
		t.Fatalf("ResolvePage = %q, want %q", got, file)
	}
}

func TestBuildRoutesFailsForMissingPage(t *testing.T) {
	docs := &config.Docs{Navigation: config.Navigation{Pages: []config.PageEntry{{Path: "missing"}}}}
	if _, err := BuildRoutes(t.TempDir(), docs); err == nil {
		t.Fatal("expected missing page error")
	}
}
