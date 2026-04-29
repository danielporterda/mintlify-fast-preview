package mdx

import (
	"bufio"
	"bytes"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/danielporterda/mintlify-fast-preview/internal/render"
)

type Document struct {
	Frontmatter map[string]string
	Body        string
	Imports     map[string]string
}

type Rendered struct {
	HTML     string
	Headings []Heading
}

type Heading struct {
	Level int
	ID    string
	Text  string
}

var importRE = regexp.MustCompile(`^import\s+(.+?)\s+from\s+["']([^"']+)["'];?\s*$`)
var defaultImportRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
var imageTextRE = regexp.MustCompile(`!\[([^\]]*)\]\([^)]+\)`)
var linkTextRE = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
var htmlTagRE = regexp.MustCompile(`<[^>]+>`)
var jsxStyleRE = regexp.MustCompile(`style=\{\{([^}]*)\}\}`)

func Parse(source string) Document {
	frontmatter, body := parseFrontmatter(source)
	imports := map[string]string{}
	var kept []string
	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		line := scanner.Text()
		if match := importRE.FindStringSubmatch(strings.TrimSpace(line)); match != nil {
			symbol := strings.TrimSpace(match[1])
			if defaultImportRE.MatchString(symbol) {
				imports[symbol] = match[2]
			}
			continue
		}
		kept = append(kept, line)
	}
	return Document{Frontmatter: frontmatter, Body: strings.Join(kept, "\n"), Imports: imports}
}

func parseFrontmatter(source string) (map[string]string, string) {
	frontmatter := map[string]string{}
	if !strings.HasPrefix(source, "---\n") {
		return frontmatter, source
	}
	rest := strings.TrimPrefix(source, "---\n")
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return frontmatter, source
	}
	for _, line := range strings.Split(rest[:end], "\n") {
		key, value, ok := strings.Cut(line, ":")
		if ok {
			frontmatter[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"`)
		}
	}
	body := strings.TrimPrefix(rest[end:], "\n---")
	body = strings.TrimPrefix(body, "\n")
	return frontmatter, body
}

func ResolveImports(root string, doc Document) (string, error) {
	body := doc.Body
	for symbol, importPath := range doc.Imports {
		resolved, err := resolveImport(root, importPath)
		if err != nil {
			return "", err
		}
		bytes, err := os.ReadFile(resolved)
		if err != nil {
			return "", err
		}
		body = strings.ReplaceAll(body, "<"+symbol+" />", string(bytes))
	}
	return body, nil
}

func resolveImport(root, importPath string) (string, error) {
	clean := strings.TrimPrefix(importPath, "/")
	candidates := []string{filepath.Join(root, clean)}
	if filepath.Ext(clean) == "" {
		candidates = append(candidates, filepath.Join(root, clean+".mdx"), filepath.Join(root, clean+".md"))
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("import not found: %s", importPath)
}

func RenderBody(markup string) string {
	return Render(markup).HTML
}

func Render(markup string) Rendered {
	var out bytes.Buffer
	scanner := bufio.NewScanner(strings.NewReader(markup))
	inCode := false
	var codeLang string
	var codeTitle string
	var code bytes.Buffer
	var paragraph []string
	var listType string
	var listItems []string
	var tableLines []string
	var quoteLines []string
	var headings []Heading
	ids := newSlugger()

	flushParagraph := func() {
		if len(paragraph) == 0 {
			return
		}
		out.WriteString("<p>")
		out.WriteString(renderInline(strings.Join(paragraph, " ")))
		out.WriteString("</p>\n")
		paragraph = nil
	}
	flushList := func() {
		if len(listItems) == 0 {
			return
		}
		out.WriteString("<")
		out.WriteString(listType)
		out.WriteString(">\n")
		for _, item := range listItems {
			out.WriteString("<li>")
			out.WriteString(renderInline(item))
			out.WriteString("</li>\n")
		}
		out.WriteString("</")
		out.WriteString(listType)
		out.WriteString(">\n")
		listItems = nil
		listType = ""
	}
	flushTable := func() {
		if len(tableLines) == 0 {
			return
		}
		out.WriteString(renderTable(tableLines))
		tableLines = nil
	}
	flushQuote := func() {
		if len(quoteLines) == 0 {
			return
		}
		out.WriteString("<blockquote>\n<p>")
		out.WriteString(renderInline(strings.Join(quoteLines, " ")))
		out.WriteString("</p>\n</blockquote>\n")
		quoteLines = nil
	}
	flushBlocks := func() {
		flushParagraph()
		flushList()
		flushTable()
		flushQuote()
	}

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "```") {
			if !inCode {
				flushBlocks()
				inCode = true
				codeLang, codeTitle = parseFenceInfo(strings.TrimSpace(strings.TrimPrefix(line, "```")))
				code.Reset()
				continue
			}
			out.WriteString(render.CodeBlockWithTitle(codeLang, codeTitle, code.String()))
			inCode = false
			continue
		}
		if inCode {
			code.WriteString(line)
			code.WriteByte('\n')
			continue
		}
		if level, text, ok := headingLine(line); ok {
			flushBlocks()
			plain := inlineText(text)
			id := ids.id(plain)
			if level == 2 || level == 3 {
				headings = append(headings, Heading{Level: level, ID: id, Text: plain})
			}
			out.WriteString("<h")
			out.WriteString(fmt.Sprint(level))
			out.WriteString(` id="`)
			out.WriteString(id)
			out.WriteString(`">`)
			out.WriteString(renderInline(text))
			out.WriteString("</h")
			out.WriteString(fmt.Sprint(level))
			out.WriteString(">\n")
			continue
		}
		if tableLine(line) {
			flushParagraph()
			flushList()
			flushQuote()
			tableLines = append(tableLines, strings.TrimSpace(line))
			continue
		}
		if quote, ok := blockquoteLine(line); ok {
			flushParagraph()
			flushList()
			flushTable()
			quoteLines = append(quoteLines, quote)
			continue
		}
		if item, ok := unorderedItem(line); ok {
			flushParagraph()
			flushTable()
			flushQuote()
			if listType != "" && listType != "ul" {
				flushList()
			}
			listType = "ul"
			listItems = append(listItems, item)
			continue
		}
		if item, ok := orderedItem(line); ok {
			flushParagraph()
			flushTable()
			flushQuote()
			if listType != "" && listType != "ol" {
				flushList()
			}
			listType = "ol"
			listItems = append(listItems, item)
			continue
		}
		rendered, block := renderLine(line)
		if block {
			flushBlocks()
			out.WriteString(rendered)
			continue
		}
		if strings.TrimSpace(line) == "" {
			flushBlocks()
			continue
		}
		flushTable()
		flushQuote()
		paragraph = append(paragraph, strings.TrimSpace(line))
	}
	flushBlocks()
	return Rendered{HTML: out.String(), Headings: headings}
}

func renderLine(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	switch {
	case trimmed == "":
		return "\n", true
	case strings.HasPrefix(trimmed, "# "):
		text := strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
		return `<h1 id="` + slug(text) + `">` + html.EscapeString(text) + "</h1>\n", true
	case strings.HasPrefix(trimmed, "## "):
		text := strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))
		return `<h2 id="` + slug(text) + `">` + html.EscapeString(text) + "</h2>\n", true
	case strings.HasPrefix(trimmed, "### "):
		text := strings.TrimSpace(strings.TrimPrefix(trimmed, "### "))
		return `<h3 id="` + slug(text) + `">` + html.EscapeString(text) + "</h3>\n", true
	case strings.HasPrefix(trimmed, "#### "):
		text := strings.TrimSpace(strings.TrimPrefix(trimmed, "#### "))
		return `<h4 id="` + slug(text) + `">` + html.EscapeString(text) + "</h4>\n", true
	case strings.HasPrefix(trimmed, "##### "):
		text := strings.TrimSpace(strings.TrimPrefix(trimmed, "##### "))
		return `<h5 id="` + slug(text) + `">` + html.EscapeString(text) + "</h5>\n", true
	case strings.HasPrefix(trimmed, "###### "):
		text := strings.TrimSpace(strings.TrimPrefix(trimmed, "###### "))
		return `<h6 id="` + slug(text) + `">` + html.EscapeString(text) + "</h6>\n", true
	case trimmed == "---" || trimmed == "***" || trimmed == "___":
		return "<hr>\n", true
	case strings.HasPrefix(trimmed, "<Warning"):
		return admonitionTag("warning", trimmed), true
	case strings.HasPrefix(trimmed, "</Warning"):
		return "</aside>\n", true
	case strings.HasPrefix(trimmed, "<Info"):
		return admonitionTag("info", trimmed), true
	case strings.HasPrefix(trimmed, "</Info"):
		return "</aside>\n", true
	case strings.HasPrefix(trimmed, "<Note"):
		return admonitionTag("note", trimmed), true
	case strings.HasPrefix(trimmed, "</Note"):
		return "</aside>\n", true
	case strings.HasPrefix(trimmed, "<Tip"):
		return admonitionTag("tip", trimmed), true
	case strings.HasPrefix(trimmed, "</Tip"):
		return "</aside>\n", true
	case strings.HasPrefix(trimmed, "<Columns"):
		return `<div class="mintfast-columns">` + "\n", true
	case strings.HasPrefix(trimmed, "</Columns"):
		return "</div>\n", true
	case strings.HasPrefix(trimmed, "<CardGroup"):
		return `<div class="mintfast-card-group">` + "\n", true
	case strings.HasPrefix(trimmed, "</CardGroup"):
		return "</div>\n", true
	case strings.HasPrefix(trimmed, "<Column"):
		return `<div class="mintfast-column">` + "\n", true
	case strings.HasPrefix(trimmed, "</Column"):
		return "</div>\n", true
	case strings.HasPrefix(trimmed, "<Card"):
		if selfClosing(trimmed) {
			return render.CardOpen(componentTitle(trimmed), attr(trimmed, "href")) + "</a>\n", true
		}
		return render.CardOpen(componentTitle(trimmed), attr(trimmed, "href")) + "\n", true
	case strings.HasPrefix(trimmed, "</Card"):
		return "</a>\n", true
	case strings.HasPrefix(trimmed, "<AccordionGroup"):
		return `<div class="mintfast-accordion-group">` + "\n", true
	case strings.HasPrefix(trimmed, "</AccordionGroup"):
		return "</div>\n", true
	case strings.HasPrefix(trimmed, "<Accordion"):
		return `<details class="mintfast-accordion" open><summary>` + html.EscapeString(componentTitle(trimmed)) + `</summary>` + "\n", true
	case strings.HasPrefix(trimmed, "</Accordion"):
		return "</details>\n", true
	case strings.HasPrefix(trimmed, "<Frame"):
		return `<figure class="mintfast-frame">` + "\n", true
	case strings.HasPrefix(trimmed, "</Frame"):
		return "</figure>\n", true
	case strings.HasPrefix(trimmed, "<Steps"):
		return `<div class="mintfast-steps">` + "\n", true
	case strings.HasPrefix(trimmed, "</Steps"):
		return "</div>\n", true
	case strings.HasPrefix(trimmed, "<Step"):
		return `<section class="mintfast-step"><h3>` + html.EscapeString(componentTitle(trimmed)) + "</h3>\n", true
	case strings.HasPrefix(trimmed, "</Step"):
		return "</section>\n", true
	case strings.HasPrefix(trimmed, "<Tabs"):
		return `<div class="mintfast-tabs">` + "\n", true
	case strings.HasPrefix(trimmed, "</Tabs"):
		return "</div>\n", true
	case strings.HasPrefix(trimmed, "<Tab"):
		return `<section class="mintfast-tab"><h3>` + html.EscapeString(componentTitle(trimmed)) + "</h3>\n", true
	case strings.HasPrefix(trimmed, "</Tab"):
		return "</section>\n", true
	case strings.HasPrefix(trimmed, "<CodeGroup"):
		return `<div class="mintfast-code-group">` + "\n", true
	case strings.HasPrefix(trimmed, "</CodeGroup"):
		return "</div>\n", true
	case strings.HasPrefix(trimmed, "<ParamField"):
		return fieldOpen("param", trimmed) + "\n", true
	case strings.HasPrefix(trimmed, "</ParamField"):
		return "</div>\n", true
	case strings.HasPrefix(trimmed, "<ResponseField"):
		return fieldOpen("response", trimmed) + "\n", true
	case strings.HasPrefix(trimmed, "</ResponseField"):
		return "</div>\n", true
	case strings.HasPrefix(trimmed, "<RequestExample"):
		return `<div class="mintfast-example mintfast-example-request">` + "\n", true
	case strings.HasPrefix(trimmed, "</RequestExample"):
		return "</div>\n", true
	case strings.HasPrefix(trimmed, "<ResponseExample"):
		return `<div class="mintfast-example mintfast-example-response">` + "\n", true
	case strings.HasPrefix(trimmed, "</ResponseExample"):
		return "</div>\n", true
	case strings.HasPrefix(trimmed, "<VersionDashboard"):
		return render.Compatibility("VersionDashboard") + "\n", true
	case isHTMLLine(trimmed):
		return normalizeHTML(trimmed) + "\n", true
	case isUnknownComponent(trimmed):
		return render.Compatibility(componentName(trimmed)) + "\n", true
	default:
		return "", false
	}
}

func componentTitle(line string) string {
	return attr(line, "title")
}

func admonitionTag(kind, line string) string {
	if selfClosing(line) {
		return render.Admonition(kind, componentTitle(line)) + "\n"
	}
	return render.AdmonitionOpen(kind, componentTitle(line)) + "\n"
}

func selfClosing(line string) bool {
	return strings.HasSuffix(strings.TrimSpace(line), "/>")
}

func attr(line, name string) string {
	key := name + `=`
	start := strings.Index(line, key)
	if start < 0 {
		return ""
	}
	rest := strings.TrimSpace(line[start+len(key):])
	if strings.HasPrefix(rest, "{") {
		rest = strings.TrimSpace(strings.TrimPrefix(rest, "{"))
	}
	if rest == "" || (rest[0] != '"' && rest[0] != '\'') {
		return ""
	}
	quote := rest[0]
	rest = rest[1:]
	end := strings.IndexRune(rest, rune(quote))
	if end < 0 {
		return ""
	}
	return rest[:end]
}

func isHTMLLine(line string) bool {
	if !strings.HasPrefix(line, "<") {
		return false
	}
	rest := strings.TrimPrefix(line, "<")
	rest = strings.TrimPrefix(rest, "/")
	if rest == "" {
		return false
	}
	first := rest[0]
	return first >= 'a' && first <= 'z'
}

func normalizeHTML(line string) string {
	replacer := strings.NewReplacer("className=", "class=")
	return jsxStyleRE.ReplaceAllStringFunc(replacer.Replace(line), func(match string) string {
		parts := jsxStyleRE.FindStringSubmatch(match)
		if len(parts) != 2 {
			return match
		}
		style := normalizeStyleObject(parts[1])
		if style == "" {
			return ""
		}
		return `style="` + html.EscapeString(style) + `"`
	})
}

func normalizeStyleObject(input string) string {
	var declarations []string
	for _, raw := range strings.Split(input, ",") {
		key, value, ok := strings.Cut(raw, ":")
		if !ok {
			continue
		}
		key = cssName(strings.TrimSpace(key))
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key == "" || value == "" {
			continue
		}
		declarations = append(declarations, key+": "+value)
	}
	return strings.Join(declarations, "; ")
}

func cssName(input string) string {
	var b strings.Builder
	for i, r := range input {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r + ('a' - 'A'))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func fieldOpen(kind, line string) string {
	name := attr(line, "path")
	if name == "" {
		name = attr(line, "name")
	}
	fieldType := attr(line, "type")
	var b strings.Builder
	b.WriteString(`<div class="mintfast-field mintfast-field-`)
	b.WriteString(kind)
	b.WriteString(`"><div class="mintfast-field-head"><code>`)
	b.WriteString(html.EscapeString(name))
	b.WriteString(`</code>`)
	if fieldType != "" {
		b.WriteString(`<span class="mintfast-field-type">`)
		b.WriteString(html.EscapeString(fieldType))
		b.WriteString(`</span>`)
	}
	if strings.Contains(line, " required") || strings.Contains(line, " required>") {
		b.WriteString(`<span class="mintfast-field-required">required</span>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}

func isUnknownComponent(line string) bool {
	if !strings.HasPrefix(line, "<") || strings.HasPrefix(line, "</") {
		return false
	}
	name := componentName(line)
	return name != "" && name[0] >= 'A' && name[0] <= 'Z'
}

func componentName(line string) string {
	line = strings.TrimPrefix(line, "<")
	line = strings.TrimPrefix(line, "/")
	for i, r := range line {
		if r == ' ' || r == '/' || r == '>' {
			return line[:i]
		}
	}
	return ""
}

func slug(text string) string {
	lower := strings.ToLower(text)
	var b strings.Builder
	prevDash := false
	for _, r := range lower {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

type slugger struct {
	seen map[string]int
}

func newSlugger() *slugger {
	return &slugger{seen: map[string]int{}}
}

func (s *slugger) id(text string) string {
	base := slug(text)
	if base == "" {
		base = "section"
	}
	count := s.seen[base]
	s.seen[base] = count + 1
	if count == 0 {
		return base
	}
	return fmt.Sprintf("%s-%d", base, count)
}

func headingLine(line string) (int, string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "#") {
		return 0, "", false
	}
	level := 0
	for level < len(trimmed) && trimmed[level] == '#' {
		level++
	}
	if level == 0 || level > 6 || level >= len(trimmed) || trimmed[level] != ' ' {
		return 0, "", false
	}
	return level, strings.TrimSpace(trimmed[level+1:]), true
}

func inlineText(input string) string {
	text := imageTextRE.ReplaceAllString(input, "$1")
	text = linkTextRE.ReplaceAllString(text, "$1")
	text = htmlTagRE.ReplaceAllString(text, "")
	replacer := strings.NewReplacer("`", "", "**", "", "__", "", "~~", "", "*", "", "_", "")
	text = replacer.Replace(text)
	return strings.Join(strings.Fields(text), " ")
}

func oldComponentTitle(line string) string {
	const key = `title="`
	start := strings.Index(line, key)
	if start < 0 {
		return ""
	}
	rest := line[start+len(key):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return ""
	}
	return rest[:end]
}

func tableLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "|") && strings.HasSuffix(trimmed, "|") && strings.Count(trimmed, "|") >= 2
}

func renderTable(lines []string) string {
	if len(lines) < 2 || !tableSeparator(lines[1]) {
		var b strings.Builder
		for _, line := range lines {
			b.WriteString("<p>")
			b.WriteString(renderInline(line))
			b.WriteString("</p>\n")
		}
		return b.String()
	}
	var b strings.Builder
	headers := splitTableRow(lines[0])
	b.WriteString("<table>\n<thead>\n<tr>")
	for _, header := range headers {
		b.WriteString("<th>")
		b.WriteString(renderInline(header))
		b.WriteString("</th>")
	}
	b.WriteString("</tr>\n</thead>\n<tbody>\n")
	for _, line := range lines[2:] {
		b.WriteString("<tr>")
		for _, cell := range splitTableRow(line) {
			b.WriteString("<td>")
			b.WriteString(renderInline(cell))
			b.WriteString("</td>")
		}
		b.WriteString("</tr>\n")
	}
	b.WriteString("</tbody>\n</table>\n")
	return b.String()
}

func tableSeparator(line string) bool {
	cells := splitTableRow(line)
	if len(cells) == 0 {
		return false
	}
	for _, cell := range cells {
		cleaned := strings.Trim(cell, " :-")
		if cleaned != "" {
			return false
		}
		if !strings.Contains(cell, "-") {
			return false
		}
	}
	return true
}

func splitTableRow(line string) []string {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimPrefix(trimmed, "|")
	trimmed = strings.TrimSuffix(trimmed, "|")
	var cells []string
	var current strings.Builder
	escaped := false
	for _, r := range trimmed {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if r == '|' {
			cells = append(cells, strings.TrimSpace(current.String()))
			current.Reset()
			continue
		}
		current.WriteRune(r)
	}
	cells = append(cells, strings.TrimSpace(current.String()))
	return cells
}

func blockquoteLine(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, ">") {
		return "", false
	}
	return strings.TrimSpace(strings.TrimPrefix(trimmed, ">")), true
}

func unorderedItem(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
		return strings.TrimSpace(trimmed[2:]), true
	}
	return "", false
}

func orderedItem(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	dot := strings.Index(trimmed, ". ")
	if dot <= 0 {
		return "", false
	}
	for _, r := range trimmed[:dot] {
		if r < '0' || r > '9' {
			return "", false
		}
	}
	return strings.TrimSpace(trimmed[dot+2:]), true
}

func renderInline(input string) string {
	var out strings.Builder
	for i := 0; i < len(input); {
		if input[i] == '`' {
			end := strings.IndexByte(input[i+1:], '`')
			if end >= 0 {
				out.WriteString("<code>")
				out.WriteString(html.EscapeString(input[i+1 : i+1+end]))
				out.WriteString("</code>")
				i += end + 2
				continue
			}
		}
		if strings.HasPrefix(input[i:], "![") {
			closeText := strings.IndexByte(input[i+2:], ']')
			if closeText >= 0 {
				openURL := i + 2 + closeText + 1
				if openURL < len(input) && input[openURL] == '(' {
					closeURL := strings.IndexByte(input[openURL+1:], ')')
					if closeURL >= 0 {
						alt := input[i+2 : i+2+closeText]
						url := input[openURL+1 : openURL+1+closeURL]
						out.WriteString(`<img src="`)
						out.WriteString(html.EscapeString(url))
						out.WriteString(`" alt="`)
						out.WriteString(html.EscapeString(alt))
						out.WriteString(`">`)
						i = openURL + closeURL + 2
						continue
					}
				}
			}
		}
		if input[i] == '[' {
			closeText := strings.IndexByte(input[i+1:], ']')
			if closeText >= 0 {
				openURL := i + 1 + closeText + 1
				if openURL < len(input) && input[openURL] == '(' {
					closeURL := strings.IndexByte(input[openURL+1:], ')')
					if closeURL >= 0 {
						text := input[i+1 : i+1+closeText]
						url := input[openURL+1 : openURL+1+closeURL]
						out.WriteString(`<a href="`)
						out.WriteString(html.EscapeString(url))
						out.WriteString(`">`)
						out.WriteString(html.EscapeString(text))
						out.WriteString("</a>")
						i = openURL + closeURL + 2
						continue
					}
				}
			}
		}
		if strings.HasPrefix(input[i:], "**") {
			end := strings.Index(input[i+2:], "**")
			if end >= 0 {
				out.WriteString("<strong>")
				out.WriteString(renderInline(input[i+2 : i+2+end]))
				out.WriteString("</strong>")
				i += end + 4
				continue
			}
		}
		if strings.HasPrefix(input[i:], "~~") {
			end := strings.Index(input[i+2:], "~~")
			if end >= 0 {
				out.WriteString("<del>")
				out.WriteString(renderInline(input[i+2 : i+2+end]))
				out.WriteString("</del>")
				i += end + 4
				continue
			}
		}
		if input[i] == '*' {
			end := strings.IndexByte(input[i+1:], '*')
			if end >= 0 {
				out.WriteString("<em>")
				out.WriteString(renderInline(input[i+1 : i+1+end]))
				out.WriteString("</em>")
				i += end + 2
				continue
			}
		}
		out.WriteString(html.EscapeString(input[i : i+1]))
		i++
	}
	return out.String()
}

func parseFenceInfo(info string) (language, title string) {
	info = strings.TrimSpace(info)
	if info == "" {
		return "", ""
	}
	fields := strings.Fields(info)
	language = fields[0]
	if len(fields) == 1 {
		return language, ""
	}
	title = strings.TrimSpace(strings.TrimPrefix(info, language))
	title = strings.Trim(title, `"`)
	return language, title
}
