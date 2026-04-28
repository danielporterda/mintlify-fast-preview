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

func TestParseDropsNamedImportsWithoutSnippetReplacement(t *testing.T) {
	doc := Parse("import { networkData } from '/snippets/generated/data.mdx';\n# Body")
	if len(doc.Imports) != 0 {
		t.Fatalf("named import should not become snippet import: %+v", doc.Imports)
	}
	if strings.Contains(doc.Body, "networkData") {
		t.Fatalf("named import line should be removed: %q", doc.Body)
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

func TestRenderCollectsHeadingsForTOC(t *testing.T) {
	rendered := Render("# Page\n\n## Install `mintfast`\n\n### From source\n\n#### Hidden depth")
	if len(rendered.Headings) != 2 {
		t.Fatalf("headings = %+v", rendered.Headings)
	}
	if rendered.Headings[0].Level != 2 || rendered.Headings[0].ID != "install-mintfast" || rendered.Headings[0].Text != "Install mintfast" {
		t.Fatalf("first heading = %+v", rendered.Headings[0])
	}
	if rendered.Headings[1].Level != 3 || rendered.Headings[1].ID != "from-source" {
		t.Fatalf("second heading = %+v", rendered.Headings[1])
	}
	if !strings.Contains(rendered.HTML, `<h2 id="install-mintfast">Install <code>mintfast</code></h2>`) {
		t.Fatalf("missing rendered heading inline markup: %s", rendered.HTML)
	}
}

func TestRenderDeduplicatesHeadingIDs(t *testing.T) {
	rendered := Render("## Install\n\n## Install\n\n### Install")
	for _, want := range []string{
		`<h2 id="install">Install</h2>`,
		`<h2 id="install-1">Install</h2>`,
		`<h3 id="install-2">Install</h3>`,
	} {
		if !strings.Contains(rendered.HTML, want) {
			t.Fatalf("missing %q in %s", want, rendered.HTML)
		}
	}
}

func TestRenderBodyCodeFenceMetadata(t *testing.T) {
	got := RenderBody("```bash grpcurl\necho ok\n```")
	if !strings.Contains(got, `data-language="bash"`) {
		t.Fatalf("missing parsed language: %s", got)
	}
	if !strings.Contains(got, `<div class="mintfast-code-title">grpcurl</div>`) {
		t.Fatalf("missing code title: %s", got)
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

func TestRenderBodyTables(t *testing.T) {
	got := RenderBody("| Name | Type |\n| --- | --- |\n| `id` | [string](/types/string) |\n| enabled | bool |")
	for _, want := range []string{
		"<table>",
		"<thead>",
		"<th>Name</th>",
		"<tbody>",
		"<td><code>id</code></td>",
		`<td><a href="/types/string">string</a></td>`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %s", want, got)
		}
	}
}

func TestRenderBodyBlockquotesAndRules(t *testing.T) {
	got := RenderBody("> First line\n> second `line`\n\n---\n\nAfter")
	if !strings.Contains(got, "<blockquote>") || !strings.Contains(got, "<code>line</code>") {
		t.Fatalf("missing blockquote: %s", got)
	}
	if !strings.Contains(got, "<hr>") {
		t.Fatalf("missing horizontal rule: %s", got)
	}
}

func TestRenderBodyInlineFormattingAndImages(t *testing.T) {
	got := RenderBody("This is **strong**, *em*, ~~gone~~, and ![Logo](/logo.svg).")
	for _, want := range []string{
		"<strong>strong</strong>",
		"<em>em</em>",
		"<del>gone</del>",
		`<img src="/logo.svg" alt="Logo">`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %s", want, got)
		}
	}
}

func TestRenderBodyMintlifyComponentAliases(t *testing.T) {
	got := RenderBody(`<CardGroup cols={2}>
<Card title="A" href="/a">
Body
</Card>
</CardGroup>
<Frame>
![Screen](/screen.png)
</Frame>
<Steps>
<Step title="Install">
Run it.
</Step>
</Steps>
<Tabs>
<Tab title="Go">
go test ./...
</Tab>
</Tabs>`)
	for _, want := range []string{
		`<div class="mintfast-card-group">`,
		`<figure class="mintfast-frame">`,
		`<div class="mintfast-steps">`,
		`<section class="mintfast-step"><h3>Install</h3>`,
		`<div class="mintfast-tabs">`,
		`<section class="mintfast-tab"><h3>Go</h3>`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %s", want, got)
		}
	}
}

func TestRenderBodyFieldComponents(t *testing.T) {
	got := RenderBody(`<ParamField path="id" type="string" required>
Identifier.
</ParamField>
<ResponseField name="ok" type="boolean">
Success flag.
</ResponseField>`)
	for _, want := range []string{
		`<div class="mintfast-field mintfast-field-param">`,
		`<code>id</code>`,
		`<span class="mintfast-field-type">string</span>`,
		`<span class="mintfast-field-required">required</span>`,
		`<div class="mintfast-field mintfast-field-response">`,
		`<code>ok</code>`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %s", want, got)
		}
	}
}

func TestRenderBodyPairedAndSelfClosingComponents(t *testing.T) {
	got := RenderBody(`<Warning title="Careful">
Read this.
</Warning>
<Info>
Heads up.
</Info>
<Card title="Standalone" href="/standalone" />`)
	for _, want := range []string{
		`<aside class="mintfast-admonition mintfast-admonition-warning"><strong>Careful</strong>`,
		`<p>Read this.</p>`,
		`</aside>`,
		`<aside class="mintfast-admonition mintfast-admonition-info"><strong>info</strong>`,
		`<a class="mintfast-card" href="/standalone"><strong>Standalone</strong></a>`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %s", want, got)
		}
	}
}
