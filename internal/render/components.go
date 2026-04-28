package render

import (
	"html"
	"strings"
)

func Admonition(kind, title string) string {
	escapedKind := html.EscapeString(kind)
	escapedTitle := html.EscapeString(title)
	if escapedTitle == "" {
		escapedTitle = escapedKind
	}
	return `<aside class="mintfast-admonition mintfast-admonition-` + escapedKind + `"><strong>` + escapedTitle + `</strong></aside>`
}

func CodeBlock(language, code string) string {
	return `<pre class="mintfast-code"><code data-language="` + html.EscapeString(language) + `">` + html.EscapeString(code) + `</code></pre>` + "\n"
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
