package mdx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseFrontmatterAndImports(t *testing.T) {
	doc := Parse("---\ntitle: \"Hello\"\ndescription: Test\n---\nimport Snip from \"/snippets/a.mdx\";\n# Body\n<Snip />")
	if doc.Frontmatter["title"] != "Hello" {
		t.Fatalf("title = %q", doc.Frontmatter["title"])
	}
	if doc.Imports["Snip"] != "/snippets/a.mdx" {
		t.Fatalf("imports = %+v", doc.Imports)
	}
	if strings.Contains(doc.Body, "import Snip") {
		t.Fatal("import line should be removed from body")
	}
}

func TestResolveImports(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "snippets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "snippets", "a.mdx"), []byte("snippet body"), 0o644); err != nil {
		t.Fatal(err)
	}
	body, err := ResolveImports(root, Document{
		Body:    "before\n<Snippet />\nafter",
		Imports: map[string]string{"Snippet": "/snippets/a.mdx"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "before\nsnippet body\nafter") {
		t.Fatalf("body = %q", body)
	}
}

func TestRenderBodyHeadingsAndCode(t *testing.T) {
	html := RenderBody("# Title\n\n```go\nfmt.Println(\"x\")\n```")
	if !strings.Contains(html, `<h1 id="title">Title</h1>`) {
		t.Fatalf("missing h1: %s", html)
	}
	if !strings.Contains(html, `data-language="go"`) {
		t.Fatalf("missing language: %s", html)
	}
	if !strings.Contains(html, "fmt.Println") {
		t.Fatalf("missing code: %s", html)
	}
}

func TestRenderBodyPassesGeneratedHTML(t *testing.T) {
	got := RenderBody(`<h1 className="x2mdx-ref-title">MediatorStatus</h1>
<dl class="x2mdx-ref-meta-grid">
  <dt>Protocol</dt>
</dl>`)
	if !strings.Contains(got, `<h1 class="x2mdx-ref-title">MediatorStatus</h1>`) {
		t.Fatalf("className was not normalized: %s", got)
	}
	if !strings.Contains(got, `<dt>Protocol</dt>`) {
		t.Fatalf("lowercase HTML was not preserved: %s", got)
	}
}

func TestRenderBodyListsAndInlineMarkup(t *testing.T) {
	got := RenderBody("See [docs](/docs) and `code`.\n\n- first\n- `second`\n\n1. one\n2. two")
	if !strings.Contains(got, `<a href="/docs">docs</a>`) {
		t.Fatalf("missing link: %s", got)
	}
	if !strings.Contains(got, `<code>code</code>`) {
		t.Fatalf("missing inline code: %s", got)
	}
	if !strings.Contains(got, "<ul>") || !strings.Contains(got, "<ol>") {
		t.Fatalf("missing lists: %s", got)
	}
}
