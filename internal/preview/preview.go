package preview

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/danielporterda/mintlify-fast-preview/internal/config"
	"github.com/danielporterda/mintlify-fast-preview/internal/mdx"
	"github.com/danielporterda/mintlify-fast-preview/internal/site"
)

func Validate(root string) error {
	docs, err := config.Load(root)
	if err != nil {
		return err
	}
	_, err = site.BuildRoutes(root, docs)
	return err
}

func RenderStatic(root, out string) error {
	docs, err := config.Load(root)
	if err != nil {
		return err
	}
	routes, err := site.BuildRoutes(root, docs)
	if err != nil {
		return err
	}
	for _, route := range routes {
		html, err := RenderPage(root, docs, route)
		if err != nil {
			return err
		}
		target := filepath.Join(out, strings.TrimPrefix(route.URL, "/"), "index.html")
		if route.URL == "/" {
			target = filepath.Join(out, "index.html")
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, []byte(html), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func RenderPage(root string, docs *config.Docs, route site.Route) (string, error) {
	bytes, err := os.ReadFile(route.FilePath)
	if err != nil {
		return "", err
	}
	doc := mdx.Parse(string(bytes))
	body, err := mdx.ResolveImports(root, doc)
	if err != nil {
		return "", err
	}
	title := doc.Frontmatter["title"]
	if title == "" {
		title = docs.Name
	}
	return shell(docs, title, mdx.RenderBody(body)), nil
}

func Serve(root, host string, port int, noOpen bool) error {
	if err := Validate(root); err != nil {
		return err
	}
	addr := fmt.Sprintf("%s:%d", host, port)
	return http.ListenAndServe(addr, http.FileServer(http.Dir(root)))
}

func shell(docs *config.Docs, title, body string) string {
	primary := docs.Colors.Primary
	if primary == "" {
		primary = "#5c4ee5"
	}
	return "<!doctype html><html><head><meta charset=\"utf-8\"><title>" + title + "</title><style>:root{--mintfast-primary:" + primary + "}body{font-family:Inter,system-ui,sans-serif;margin:0}.layout{display:grid;grid-template-columns:280px 1fr;min-height:100vh}.nav{border-right:1px solid #e5e7eb;padding:1rem}.content{padding:2rem;max-width:92rem}.mintfast-code{background:#0f172a;color:#e2e8f0;padding:1rem;overflow:auto}.mintfast-card{display:block;border:1px solid #d1d5db;border-radius:8px;padding:1rem}</style></head><body><div class=\"layout\"><nav class=\"nav\"><strong>" + docs.Name + "</strong></nav><main class=\"content\">" + body + "</main></div></body></html>"
}
