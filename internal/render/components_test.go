package render

import (
	"strings"
	"testing"
)

func TestAdmonitionEscapesTitle(t *testing.T) {
	got := Admonition("warning", `<script>`)
	if strings.Contains(got, "<script>") {
		t.Fatalf("title was not escaped: %s", got)
	}
	if !strings.Contains(got, "mintfast-admonition-warning") {
		t.Fatalf("missing kind class: %s", got)
	}
}

func TestCodeBlockEscapesCode(t *testing.T) {
	got := CodeBlock("go", `<tag>`)
	if strings.Contains(got, "<tag>") {
		t.Fatalf("code was not escaped: %s", got)
	}
	if !strings.Contains(got, `data-language="go"`) {
		t.Fatalf("missing language: %s", got)
	}
	if !strings.Contains(got, `<button class="mintfast-copy" type="button" aria-label="Copy code">Copy</button>`) {
		t.Fatalf("missing copy button: %s", got)
	}
}

func TestCodeBlockWithTitle(t *testing.T) {
	got := CodeBlockWithTitle("bash", "grpcurl", "echo ok")
	if !strings.Contains(got, `data-language="bash"`) {
		t.Fatalf("missing language: %s", got)
	}
	if !strings.Contains(got, `<div class="mintfast-code-title">grpcurl</div>`) {
		t.Fatalf("missing title: %s", got)
	}
}

func TestCardEscapesHrefAndTitle(t *testing.T) {
	got := Card(`A&B`, `/x?y=1&z=2`)
	if !strings.Contains(got, "A&amp;B") || !strings.Contains(got, "&amp;z=2") {
		t.Fatalf("card not escaped: %s", got)
	}
}
