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

func TestParseDropsExportedReactBlocksAndJSXComments(t *testing.T) {
	doc := Parse(`---
title: Dashboard
---
import { networkData } from '/snippets/generated/data.mdx';

export const Tooltip = ({ children }) => {
  return (
    <span>{children}</span>
  )
}

{/* COPIED_START: some/source */}
Visible text
{/* COPIED_END */}

<Note>Keep me</Note>
<VersionDashboard />
`)
	if strings.Contains(doc.Body, "export const Tooltip") || strings.Contains(doc.Body, "COPIED_START") {
		t.Fatalf("export/jsx comment leaked into body: %s", doc.Body)
	}
	for _, want := range []string{"Visible text", "<Note>Keep me</Note>", "<VersionDashboard />"} {
		if !strings.Contains(doc.Body, want) {
			t.Fatalf("missing %q in %s", want, doc.Body)
		}
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

func TestRenderBodyNormalizesJSXStyleHTML(t *testing.T) {
	got := RenderBody(`<img src="/diagram.svg" className="align-center" style={{width: "80.0%"}} alt="Diagram" />`)
	for _, want := range []string{
		`class="align-center"`,
		`style="width: 80.0%"`,
		`alt="Diagram"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %s", want, got)
		}
	}
	if strings.Contains(got, "className") || strings.Contains(got, "{{") {
		t.Fatalf("JSX syntax leaked into HTML: %s", got)
	}
}

func TestRenderBodyNormalizesCamelCaseStyleKeys(t *testing.T) {
	got := RenderBody(`<p style={{fontSize: "smaller", fontStyle: "italic"}}>* only available.</p>`)
	if !strings.Contains(got, `style="font-size: smaller; font-style: italic"`) {
		t.Fatalf("style was not normalized: %s", got)
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

func TestRenderBodyMermaidFence(t *testing.T) {
	got := RenderBody("```mermaid\nflowchart LR\n  A[Start] --> B[Done]\n```")
	for _, want := range []string{
		`<div class="mermaid">`,
		`flowchart LR`,
		`A[Start] --&gt; B[Done]`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %s", want, got)
		}
	}
	if strings.Contains(got, `class="mintfast-code-block"`) {
		t.Fatalf("mermaid should not render as a generic code block: %s", got)
	}
}

func TestRenderBodyIndentedFencesInsideAccordion(t *testing.T) {
	got := RenderBody("<Accordion title=\"The splice container keeps crashing - what should I do?\">\n" +
		"**Diagnostic steps:**\n\n" +
		"1. **Check logs:**\n" +
		"   ```bash\n" +
		"   docker logs splice-validator-participant-1\n" +
		"   ```\n\n" +
		"2. **Verify resources:**\n" +
		"   - Docker memory >= 8GB\n" +
		"   - Docker CPU >= 4 cores\n\n" +
		"**Common solutions:**\n\n" +
		"```bash\n" +
		"colima stop\n" +
		"colima start --memory 8 --cpu 4\n" +
		"```\n" +
		"</Accordion>\n\n" +
		"<Accordion title=\"make build fails with env_file type errors - what's wrong?\">\n" +
		"**Error:**\n" +
		"```\n" +
		"'env_file[1]' expected type 'string', got unconvertible type 'map[string]interface {}'\n" +
		"```\n" +
		"</Accordion>")
	for _, want := range []string{
		`<details class="mintfast-accordion" open><summary>The splice container keeps crashing - what should I do?</summary>`,
		`data-language="bash"`,
		`docker logs splice-validator-participant-1`,
		`colima start --memory 8 --cpu 4`,
		`<details class="mintfast-accordion" open><summary>make build fails with env_file type errors - what&#39;s wrong?</summary>`,
		`env_file[1]`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %s", want, got)
		}
	}
	for _, unwanted := range []string{`&lt;Accordion`, `&lt;/Accordion` } {
		if strings.Contains(got, unwanted) {
			t.Fatalf("accordion markup leaked into output: %s", got)
		}
	}
}

func TestRenderBodyQuotedFenceDoesNotLeakFollowingComponents(t *testing.T) {
	got := RenderBody("> **TemplateName**\n" +
		"> Represents the contract data or the template fields.\n" +
		">\n" +
		"> ```daml\n" +
		"-- Code from: >\n" +
		"> ./code-snippets/Com/Acme/Templates.daml\n" +
		">\n" +
		">\n" +
		"-- [Include actual code example here]\n" +
		"```\n\n" +
		"The Java code generated for this variant is:\n\n" +
		"```java\n" +
		"package com.acme.enum;\n\n" +
		"public enum Color implements DamlEnum<Color> {\n" +
		"  RED,\n" +
		"  GREEN,\n" +
		"  BLUE;\n" +
		"}\n" +
		"```\n\n" +
		"##### Parameterized types\n\n" +
		"<Note>\n" +
		"This section is only included for completeness.\n" +
		"</Note>\n")
	for _, want := range []string{
		`data-language="daml"`,
		`./code-snippets/Com/Acme/Templates.daml`,
		`data-language="java"`,
		`DamlEnum&lt;Color&gt;`,
		`<h5 id="parameterized-types">Parameterized types</h5>`,
		`<aside class="mintfast-admonition mintfast-admonition-note"><strong>note</strong>`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %s", want, got)
		}
	}
	for _, unwanted := range []string{`&lt;Note&gt;`, `&lt;/Note&gt;`} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("quoted fence leaked following component markup: %s", got)
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

func TestRenderBodyCheckComponent(t *testing.T) {
	got := RenderBody(`<Check>
**Transaction content** is visible only to authorized parties.
</Check>`)
	for _, want := range []string{
		`<aside class="mintfast-admonition mintfast-admonition-check"><strong>check</strong>`,
		`<strong>Transaction content</strong>`,
		`authorized parties.`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %s", want, got)
		}
	}
}

func TestRenderBodyParsesExpressionStringProps(t *testing.T) {
	got := RenderBody(`<Card title={"Go SDK"} href={'/sdks/go'} />`)
	if !strings.Contains(got, `<a class="mintfast-card" href="/sdks/go"><strong>Go SDK</strong></a>`) {
		t.Fatalf("expression string props were not parsed: %s", got)
	}
}
