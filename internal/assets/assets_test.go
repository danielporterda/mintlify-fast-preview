package assets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveAssetInsideRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "images"), 0o755); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "images", "logo.svg")
	if err := os.WriteFile(want, []byte("<svg/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Resolve(root, "/images/logo.svg")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("Resolve = %q, want %q", got, want)
	}
}

func TestResolveRejectsTraversal(t *testing.T) {
	if _, err := Resolve(t.TempDir(), "../secret"); err == nil {
		t.Fatal("expected traversal rejection")
	}
}
