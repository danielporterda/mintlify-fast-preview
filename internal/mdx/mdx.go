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

var importRE = regexp.MustCompile(`^import\s+(.+?)\s+from\s+["']([^"']+)["'];?\s*$`)
var defaultImportRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

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
	var out bytes.Buffer
	scanner := bufio.NewScanner(strings.NewReader(markup))
	inCode := false
	var codeLang string
	var code bytes.Buffer
	var paragraph []string
	var listType string
	var listItems []string

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

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "```") {
			if !inCode {
				flushParagraph()
				flushList()
				inCode = true
				codeLang = strings.TrimSpace(strings.TrimPrefix(line, "```"))
				code.Reset()
				continue
			}
			out.WriteString(render.CodeBlock(codeLang, code.String()))
			inCode = false
			continue
		}
		if inCode {
			code.WriteString(line)
			code.WriteByte('\n')
			continue
		}
		if item, ok := unorderedItem(line); ok {
			flushParagraph()
			if listType != "" && listType != "ul" {
				flushList()
			}
			listType = "ul"
			listItems = append(listItems, item)
			continue
		}
		if item, ok := orderedItem(line); ok {
			flushParagraph()
			if listType != "" && listType != "ol" {
				flushList()
			}
			listType = "ol"
			listItems = append(listItems, item)
			continue
		}
		rendered, block := renderLine(line)
		if block {
			flushParagraph()
			flushList()
			out.WriteString(rendered)
			continue
		}
		if strings.TrimSpace(line) == "" {
			flushParagraph()
			flushList()
			continue
		}
		paragraph = append(paragraph, strings.TrimSpace(line))
	}
	flushParagraph()
	flushList()
	return out.String()
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
	case strings.HasPrefix(trimmed, "<Warning"):
		return render.Admonition("warning", componentTitle(trimmed)) + "\n", true
	case strings.HasPrefix(trimmed, "<Info"):
		return render.Admonition("info", componentTitle(trimmed)) + "\n", true
	case strings.HasPrefix(trimmed, "<Note"):
		return render.Admonition("note", componentTitle(trimmed)) + "\n", true
	case strings.HasPrefix(trimmed, "<Tip"):
		return render.Admonition("tip", componentTitle(trimmed)) + "\n", true
	case strings.HasPrefix(trimmed, "<Columns"):
		return `<div class="mintfast-columns">` + "\n", true
	case strings.HasPrefix(trimmed, "</Columns"):
		return "</div>\n", true
	case strings.HasPrefix(trimmed, "<Column"):
		return `<div class="mintfast-column">` + "\n", true
	case strings.HasPrefix(trimmed, "</Column"):
		return "</div>\n", true
	case strings.HasPrefix(trimmed, "<Card"):
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

func attr(line, name string) string {
	key := name + `="`
	start := strings.Index(line, key)
	if start < 0 {
		key = name + `='`
		start = strings.Index(line, key)
		if start < 0 {
			return ""
		}
	}
	quote := key[len(key)-1]
	rest := line[start+len(key):]
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
	return replacer.Replace(line)
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
		out.WriteString(html.EscapeString(input[i : i+1]))
		i++
	}
	return out.String()
}
