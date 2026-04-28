package render

import "html"

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
