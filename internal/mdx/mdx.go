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

var importRE = regexp.MustCompile(`^import\s+([A-Za-z_][A-Za-z0-9_]*)\s+from\s+"([^"]+)";?\s*$`)

func Parse(source string) Document {
	frontmatter, body := parseFrontmatter(source)
	imports := map[string]string{}
	var kept []string
	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		line := scanner.Text()
		if match := importRE.FindStringSubmatch(line); match != nil {
			imports[match[1]] = match[2]
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
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "```") {
			if !inCode {
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
		out.WriteString(renderLine(line))
	}
	return out.String()
}

func renderLine(line string) string {
	trimmed := strings.TrimSpace(line)
	switch {
	case trimmed == "":
		return "\n"
	case strings.HasPrefix(trimmed, "# "):
		return "<h1>" + html.EscapeString(strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))) + "</h1>\n"
	case strings.HasPrefix(trimmed, "## "):
		return "<h2>" + html.EscapeString(strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))) + "</h2>\n"
	case strings.HasPrefix(trimmed, "### "):
		return "<h3>" + html.EscapeString(strings.TrimSpace(strings.TrimPrefix(trimmed, "### "))) + "</h3>\n"
	case strings.HasPrefix(trimmed, "<Warning"):
		return render.Admonition("warning", componentTitle(trimmed)) + "\n"
	case strings.HasPrefix(trimmed, "<Info"):
		return render.Admonition("info", componentTitle(trimmed)) + "\n"
	case strings.HasPrefix(trimmed, "<Note"):
		return render.Admonition("note", componentTitle(trimmed)) + "\n"
	default:
		return "<p>" + html.EscapeString(trimmed) + "</p>\n"
	}
}

func componentTitle(line string) string {
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
