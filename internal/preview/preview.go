package preview

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/danielporterda/mintlify-fast-preview/internal/config"
	"github.com/danielporterda/mintlify-fast-preview/internal/mdx"
	"github.com/danielporterda/mintlify-fast-preview/internal/search"
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
	return copyStaticAssets(root, out)
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
	rendered := mdx.Render(body)
	return shell(root, docs, title, rendered.HTML, rendered.Headings), nil
}

func Serve(root, host string, port int, noOpen bool) error {
	handler, err := NewHandler(root)
	if err != nil {
		return err
	}
	addr := fmt.Sprintf("%s:%d", host, port)
	stop := make(chan struct{})
	defer close(stop)
	go watchForReloads(root, handler.reload, 250*time.Millisecond, stop)
	return http.ListenAndServe(addr, handler)
}

func shell(root string, docs *config.Docs, title, body string, headings []mdx.Heading) string {
	primary := docs.Colors.Primary
	if primary == "" {
		primary = "#5c4ee5"
	}
	return "<!doctype html><html><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\"><title>" + html.EscapeString(title) + "</title><style>" + baseCSS(primary) + "</style>" + customCSS(root) + liveReloadScript() + copyButtonScript() + "</head><body><div class=\"layout\"><nav class=\"nav\"><strong class=\"brand\">" + html.EscapeString(docs.Name) + "</strong>" + navHTML(docs) + "</nav><main class=\"content\">" + body + "</main>" + tocHTML(headings) + "</div></body></html>"
}

type Handler struct {
	root        string
	docs        *config.Docs
	routes      map[string]site.Route
	searchIndex *search.Index
	reload      *reloadHub
}

func NewHandler(root string) (*Handler, error) {
	docs, err := config.Load(root)
	if err != nil {
		return nil, err
	}
	routes, err := site.BuildRoutes(root, docs)
	if err != nil {
		return nil, err
	}
	byRoute := map[string]site.Route{}
	for _, route := range routes {
		byRoute[route.URL] = route
	}
	index := buildSearchIndex(root, docs, routes)
	return &Handler{root: root, docs: docs, routes: byRoute, searchIndex: index, reload: newReloadHub()}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.URL.Path == "/_mintfast/search" {
		h.serveSearch(w, r)
		return
	}
	if r.URL.Path == "/_mintfast/events" {
		h.serveEvents(w, r)
		return
	}
	routePath := config.CleanRoute(r.URL.Path)
	if route, ok := h.routes[routePath]; ok {
		page, err := RenderPage(h.root, h.docs, route)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("content-type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(page))
		return
	}
	http.FileServer(http.Dir(h.root)).ServeHTTP(w, r)
}

func (h *Handler) serveSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	results := h.searchIndex.Search(r.URL.Query().Get("q"))
	_ = json.NewEncoder(w).Encode(results)
}

func (h *Handler) serveEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "text/event-stream")
	w.Header().Set("cache-control", "no-cache")
	flusher, _ := w.(http.Flusher)
	_, _ = io.WriteString(w, "event: ready\ndata: {}\n\n")
	if flusher != nil {
		flusher.Flush()
	}
	events := h.reload.subscribe()
	defer h.reload.unsubscribe(events)
	for {
		select {
		case <-events:
			_, _ = io.WriteString(w, "event: reload\ndata: {}\n\n")
			if flusher != nil {
				flusher.Flush()
			}
		case <-r.Context().Done():
			return
		}
	}
}

func navHTML(docs *config.Docs) string {
	var b strings.Builder
	b.WriteString(`<div class="nav-tree">`)
	for _, dropdown := range docs.Navigation.Dropdowns {
		b.WriteString(`<section class="nav-section"><h2>`)
		b.WriteString(html.EscapeString(dropdown.Dropdown))
		b.WriteString(`</h2>`)
		for _, group := range dropdown.Groups {
			writeGroup(&b, group)
		}
		for _, version := range dropdown.Versions {
			b.WriteString(`<h3>`)
			b.WriteString(html.EscapeString(version.Version))
			b.WriteString(`</h3>`)
			for _, group := range version.Groups {
				writeGroup(&b, group)
			}
		}
		b.WriteString(`</section>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}

func writeGroup(b *strings.Builder, group config.Group) {
	b.WriteString(`<div class="nav-group"><h4>`)
	b.WriteString(html.EscapeString(group.Group))
	b.WriteString(`</h4><ul>`)
	for _, page := range group.Pages {
		if page.Group != "" || page.Path == "" {
			continue
		}
		route := config.CleanRoute(page.Path)
		b.WriteString(`<li><a href="`)
		b.WriteString(html.EscapeString(route))
		b.WriteString(`">`)
		b.WriteString(html.EscapeString(label(page.Path)))
		b.WriteString(`</a></li>`)
	}
	b.WriteString(`</ul></div>`)
}

func tocHTML(headings []mdx.Heading) string {
	var b strings.Builder
	b.WriteString(`<aside class="toc"><strong>On this page</strong>`)
	for _, heading := range headings {
		b.WriteString(`<a href="#`)
		b.WriteString(html.EscapeString(heading.ID))
		b.WriteString(`" class="toc-level-`)
		b.WriteString(fmt.Sprint(heading.Level))
		b.WriteString(`">`)
		b.WriteString(html.EscapeString(heading.Text))
		b.WriteString(`</a>`)
	}
	b.WriteString(`</aside>`)
	return b.String()
}

func label(page string) string {
	trimmed := strings.TrimSuffix(strings.Trim(page, "/"), "/index")
	base := filepath.Base(trimmed)
	base = strings.ReplaceAll(base, "-", " ")
	base = strings.ReplaceAll(base, "_", " ")
	if base == "." || base == "/" {
		return "Home"
	}
	return strings.Title(base)
}

func baseCSS(primary string) string {
	return `:root{--mintfast-primary:` + primary + `;--border:#e5e7eb;--muted:#6b7280;--code:#0f172a;--soft:#f9fafb}*{box-sizing:border-box}body{font-family:Inter,system-ui,sans-serif;margin:0;color:#111827;background:#fff}.layout{display:grid;grid-template-columns:280px minmax(0,1fr)220px;min-height:100vh}.nav{border-right:1px solid var(--border);padding:1rem;overflow:auto}.brand{display:block;margin-bottom:1rem}.nav h2{font-size:.8rem;text-transform:uppercase;color:var(--muted);margin:1rem 0 .5rem}.nav h3{font-size:.85rem;margin:.75rem 0 .4rem}.nav h4{font-size:.8rem;margin:.75rem 0 .25rem}.nav ul{list-style:none;margin:0;padding:0}.nav a{display:block;color:#374151;text-decoration:none;padding:.25rem 0}.content{padding:2rem;max-width:92rem;width:100%}.toc{border-left:1px solid var(--border);padding:1rem;color:var(--muted)}.toc strong{display:block;margin-bottom:.75rem}.toc a{display:block;text-decoration:none;color:#4b5563;font-size:.85rem;line-height:1.35;padding:.25rem 0}.toc-level-3{padding-left:.75rem}h1{font-size:2rem;line-height:1.2}h2{font-size:1.35rem;margin-top:2rem}a{color:var(--mintfast-primary)}table{border-collapse:collapse;width:100%;margin:1rem 0;font-size:.92rem}th,td{border:1px solid var(--border);padding:.55rem .7rem;text-align:left;vertical-align:top}th{background:var(--soft);font-weight:650}blockquote{border-left:4px solid var(--border);margin:1rem 0;padding:.25rem 1rem;color:#4b5563;background:var(--soft)}hr{border:0;border-top:1px solid var(--border);margin:2rem 0}.mintfast-code-block{position:relative;margin:1rem 0}.mintfast-copy{position:absolute;right:.5rem;top:.5rem;border:1px solid #475569;background:#1f2937;color:#e5e7eb;border-radius:6px;padding:.2rem .45rem;font-size:.75rem;cursor:pointer}.mintfast-code-title{background:#111827;color:#d1d5db;border-radius:8px 8px 0 0;padding:.45rem .75rem;font-size:.8rem;border-bottom:1px solid #374151}.mintfast-code-title+.mintfast-copy{top:.42rem}.mintfast-code-title~.mintfast-code{border-radius:0 0 8px 8px;margin-top:0}.mintfast-code{background:var(--code);color:#e2e8f0;border-radius:8px;padding:1rem;overflow:auto}.mintfast-card{display:block;border:1px solid var(--border);border-radius:8px;padding:1rem;text-decoration:none;color:inherit}.mintfast-card p{margin:.5rem 0 0}.mintfast-columns,.mintfast-card-group{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:1rem}.mintfast-admonition{display:block;border:1px solid var(--border);border-left:4px solid var(--mintfast-primary);border-radius:8px;padding:1rem;margin:1rem 0}.mintfast-admonition-warning{border-left-color:#f59e0b}.mintfast-accordion,.mintfast-frame,.mintfast-step,.mintfast-tab,.mintfast-field{border:1px solid var(--border);border-radius:8px;padding:1rem;margin:.75rem 0}.mintfast-frame img{max-width:100%;height:auto}.mintfast-steps,.mintfast-tabs,.mintfast-code-group{display:grid;gap:.75rem;margin:1rem 0}.mintfast-field-head{display:flex;gap:.5rem;align-items:center;flex-wrap:wrap}.mintfast-field-type,.mintfast-field-required{font-size:.75rem;border:1px solid var(--border);border-radius:999px;padding:.1rem .45rem;color:#4b5563}.mintfast-field-required{border-color:#f59e0b;color:#92400e}.mintfast-compat{border:1px dashed #a78bfa;background:#faf5ff;border-radius:8px;padding:1rem}.x2mdx-ref-operation-shell{display:grid;grid-template-columns:minmax(0,1fr)360px;gap:2rem}.x2mdx-ref-right-rail{position:sticky;top:1rem;align-self:start}@media(max-width:900px){.layout{grid-template-columns:1fr}.nav,.toc{display:none}.mintfast-columns,.mintfast-card-group,.x2mdx-ref-operation-shell{grid-template-columns:1fr}}`
}

func customCSS(root string) string {
	if info, err := os.Stat(filepath.Join(root, "styles.css")); err == nil && !info.IsDir() {
		return `<link rel="stylesheet" href="/styles.css">`
	}
	return ""
}

func liveReloadScript() string {
	return `<script>try{new EventSource('/_mintfast/events').addEventListener('reload',()=>location.reload())}catch(e){}</script>`
}

func copyButtonScript() string {
	return `<script>document.addEventListener('click',async(e)=>{const b=e.target.closest('.mintfast-copy');if(!b)return;const c=b.parentElement&&b.parentElement.querySelector('code');if(!c||!navigator.clipboard)return;await navigator.clipboard.writeText(c.innerText);b.textContent='Copied';setTimeout(()=>b.textContent='Copy',1200);});</script>`
}

func buildSearchIndex(root string, docs *config.Docs, routes []site.Route) *search.Index {
	documents := make([]search.Document, 0, len(routes))
	seen := map[string]bool{}
	for _, route := range routes {
		if seen[route.URL] {
			continue
		}
		seen[route.URL] = true
		bytes, err := os.ReadFile(route.FilePath)
		if err != nil {
			continue
		}
		doc := mdx.Parse(string(bytes))
		title := doc.Frontmatter["title"]
		if title == "" {
			title = label(route.Page.Path)
		}
		documents = append(documents, search.Document{
			Route: route.URL,
			Title: title,
			Body:  doc.Body,
		})
	}
	return search.NewIndex(documents)
}

func copyStaticAssets(root, out string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		ext := filepath.Ext(path)
		if rel == "docs.json" || ext == ".mdx" || ext == ".md" {
			return nil
		}
		target := filepath.Join(out, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		bytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, bytes, 0o644)
	})
}
