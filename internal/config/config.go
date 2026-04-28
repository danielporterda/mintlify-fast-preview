package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type Docs struct {
	Schema     string     `json:"$schema"`
	Name       string     `json:"name"`
	Theme      string     `json:"theme"`
	Colors     Colors     `json:"colors"`
	Logo       Logo       `json:"logo"`
	Navigation Navigation `json:"navigation"`
}

type Colors struct {
	Primary string `json:"primary"`
	Light   string `json:"light"`
	Dark    string `json:"dark"`
}

type Logo struct {
	Light string `json:"light"`
	Dark  string `json:"dark"`
	Href  string `json:"href"`
}

type Navigation struct {
	Dropdowns []Dropdown  `json:"dropdowns"`
	Groups    []Group     `json:"groups"`
	Pages     []PageEntry `json:"pages"`
}

type Dropdown struct {
	Dropdown string      `json:"dropdown"`
	Icon     string      `json:"icon"`
	Versions []Version   `json:"versions"`
	Groups   []Group     `json:"groups"`
	Pages    []PageEntry `json:"pages"`
}

type Version struct {
	Version string      `json:"version"`
	Groups  []Group     `json:"groups"`
	Pages   []PageEntry `json:"pages"`
}

type Group struct {
	Group string      `json:"group"`
	Pages []PageEntry `json:"pages"`
}

type PageEntry struct {
	Path  string
	Group string
	Pages []PageEntry
}

type Page struct {
	Path     string
	Route    string
	Dropdown string
	Version  string
	Group    string
}

func Load(root string) (*Docs, error) {
	bytes, err := os.ReadFile(filepath.Join(root, "docs.json"))
	if err != nil {
		return nil, fmt.Errorf("read docs.json: %w", err)
	}
	var docs Docs
	if err := json.Unmarshal(bytes, &docs); err != nil {
		return nil, fmt.Errorf("parse docs.json: %w", err)
	}
	return &docs, nil
}

func (d Docs) FlattenPages() []Page {
	var pages []Page
	var addPages func(dropdown, version, group string, refs []PageEntry)
	addPages = func(dropdown, version, group string, refs []PageEntry) {
		for _, ref := range refs {
			if ref.Group != "" {
				addPages(dropdown, version, ref.Group, ref.Pages)
				continue
			}
			if ref.Path == "" {
				continue
			}
			pages = append(pages, Page{
				Path:     strings.TrimPrefix(ref.Path, "/"),
				Route:    CleanRoute(ref.Path),
				Dropdown: dropdown,
				Version:  version,
				Group:    group,
			})
		}
	}
	addPages("", "", "", d.Navigation.Pages)
	for _, group := range d.Navigation.Groups {
		addPages("", "", group.Group, group.Pages)
	}
	for _, dropdown := range d.Navigation.Dropdowns {
		addPages(dropdown.Dropdown, "", "", dropdown.Pages)
		for _, group := range dropdown.Groups {
			addPages(dropdown.Dropdown, "", group.Group, group.Pages)
		}
		for _, version := range dropdown.Versions {
			addPages(dropdown.Dropdown, version.Version, "", version.Pages)
			for _, group := range version.Groups {
				addPages(dropdown.Dropdown, version.Version, group.Group, group.Pages)
			}
		}
	}
	return pages
}

func CleanRoute(ref string) string {
	clean := path.Clean("/" + strings.TrimPrefix(ref, "/"))
	clean = strings.TrimSuffix(clean, ".mdx")
	clean = strings.TrimSuffix(clean, ".md")
	if clean == "/index" {
		return "/"
	}
	if strings.HasSuffix(clean, "/index") {
		return strings.TrimSuffix(clean, "/index")
	}
	return clean
}

func (p *PageEntry) UnmarshalJSON(data []byte) error {
	var pathRef string
	if err := json.Unmarshal(data, &pathRef); err == nil {
		p.Path = pathRef
		return nil
	}
	var object struct {
		Group string      `json:"group"`
		Pages []PageEntry `json:"pages"`
		Page  string      `json:"page"`
		Path  string      `json:"path"`
		Href  string      `json:"href"`
	}
	if err := json.Unmarshal(data, &object); err != nil {
		return err
	}
	p.Group = object.Group
	p.Pages = object.Pages
	switch {
	case object.Page != "":
		p.Path = object.Page
	case object.Path != "":
		p.Path = object.Path
	case object.Href != "" && !strings.HasPrefix(object.Href, "http://") && !strings.HasPrefix(object.Href, "https://") && !strings.HasPrefix(object.Href, "mailto:"):
		p.Path = object.Href
	}
	return nil
}
