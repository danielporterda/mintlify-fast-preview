package render

import (
	"html"
	"strings"
)

func Admonition(kind, title string) string {
	return AdmonitionOpen(kind, title) + "</aside>"
}

func AdmonitionOpen(kind, title string) string {
	escapedKind := html.EscapeString(kind)
	escapedTitle := html.EscapeString(title)
	if escapedTitle == "" {
		escapedTitle = escapedKind
	}
	return `<aside class="mintfast-admonition mintfast-admonition-` + escapedKind + `"><strong>` + escapedTitle + `</strong>`
}

func CodeBlock(language, code string) string {
	return CodeBlockWithTitle(language, "", code)
}

func CodeBlockWithTitle(language, title, code string) string {
	var b strings.Builder
	b.WriteString(`<div class="mintfast-code-block">`)
	if title != "" {
		b.WriteString(`<div class="mintfast-code-title">`)
		b.WriteString(html.EscapeString(title))
		b.WriteString(`</div>`)
	}
	b.WriteString(`<button class="mintfast-copy" type="button" aria-label="Copy code">Copy</button>`)
	b.WriteString(`<pre class="mintfast-code"><code data-language="`)
	b.WriteString(html.EscapeString(language))
	b.WriteString(`">`)
	b.WriteString(html.EscapeString(code))
	b.WriteString(`</code></pre></div>`)
	b.WriteString("\n")
	return b.String()
}

func Mermaid(diagram string) string {
	var b strings.Builder
	b.WriteString(`<div class="mermaid">`)
	b.WriteString(html.EscapeString(strings.TrimSpace(diagram)))
	b.WriteString(`</div>`)
	b.WriteString("\n")
	return b.String()
}

func Card(title, href string) string {
	return `<a class="mintfast-card" href="` + html.EscapeString(href) + `">` + html.EscapeString(title) + `</a>`
}

func CardOpen(title, href string) string {
	var b strings.Builder
	b.WriteString(`<a class="mintfast-card" href="`)
	b.WriteString(html.EscapeString(href))
	b.WriteString(`"><strong>`)
	b.WriteString(html.EscapeString(title))
	b.WriteString(`</strong>`)
	return b.String()
}

func Compatibility(name string) string {
	return `<div class="mintfast-compat" data-component="` + html.EscapeString(name) + `">` +
		`<strong>Unsupported interactive component</strong><p>` + html.EscapeString(name) +
		` needs the React compatibility runtime.</p></div>`
}
