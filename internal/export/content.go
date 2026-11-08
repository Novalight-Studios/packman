package export

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/Novalight-Studios/packman/internal/curseforge"
	"github.com/Novalight-Studios/packman/internal/pack"
)

// ListOptions tunes ContentList.
type ListOptions struct {
	Platform string      // "modrinth" or "curseforge": which pages to link to
	Kinds    []pack.Kind // which types to list (empty: all)

	// CurseForge, if set, gives the real page URLs of CurseForge projects.
	// Without it they link to https://www.curseforge.com/projects/<id>.
	CurseForge *curseforge.Client
}

// ListReport is the result of ContentList.
type ListReport struct {
	Markdown  string   // empty if there is nothing to list
	Count     int      // number of entries listed
	Fallbacks []string // entries that aren't on Platform: they link to the other one
	Warnings  []string
}

var sectionTitles = map[pack.Kind]string{
	pack.KindMod:          "Mods used",
	pack.KindResourcePack: "Resource packs used",
	pack.KindShader:       "Shader packs used",
}

// ContentList writes the content of the pack as a Markdown list of links,
// one section per type, sorted by name.
func ContentList(p *pack.Pack, o ListOptions) (*ListReport, error) {
	if o.Platform != "modrinth" && o.Platform != "curseforge" {
		return nil, fmt.Errorf("unknown platform %q", o.Platform)
	}
	kinds := o.Kinds
	if len(kinds) == 0 {
		kinds = pack.AllKinds
	}
	rep := &ListReport{}

	// Real CurseForge page URLs, fetched in bulk.
	var cfURLs map[int]string
	if o.Platform == "curseforge" && o.CurseForge != nil {
		var ids []int
		for _, k := range kinds {
			for _, m := range *p.Items(k) {
				if m.CurseForge != nil {
					ids = append(ids, m.CurseForge.ProjectID)
				}
			}
		}
		if len(ids) > 0 {
			urls, err := o.CurseForge.WebsiteURLs(ids)
			if err != nil {
				rep.Warnings = append(rep.Warnings,
					"couldn't get the CurseForge page URLs ("+err.Error()+"), linking to project IDs instead")
			} else {
				cfURLs = urls
			}
		}
	}

	type item struct{ name, url string }
	type section struct {
		title string
		items []item
	}
	var sections []section

	for _, k := range kinds {
		var items []item
		for _, m := range *p.Items(k) {
			url, onPlatform := pageURL(m, o.Platform, cfURLs)
			if url == "" {
				rep.Warnings = append(rep.Warnings, m.Name+": no Modrinth or CurseForge data, left out")
				continue
			}
			if !onPlatform {
				rep.Fallbacks = append(rep.Fallbacks, m.Name)
			}
			items = append(items, item{name: strings.TrimSpace(m.Name), url: url})
		}
		if len(items) == 0 {
			continue
		}
		sort.SliceStable(items, func(i, j int) bool {
			return sortKey(items[i].name) < sortKey(items[j].name)
		})
		sections = append(sections, section{title: sectionTitles[k], items: items})
		rep.Count += len(items)
	}
	if len(sections) == 0 {
		return rep, nil
	}

	var b strings.Builder
	b.WriteString("# List of projects used\n\n")
	for _, s := range sections {
		fmt.Fprintf(&b, "- [%s](#%s)\n", s.title, anchor(s.title))
	}
	for _, s := range sections {
		fmt.Fprintf(&b, "\n## %s\n\n", s.title)
		for _, it := range s.items {
			fmt.Fprintf(&b, "- [%s](%s)\n", it.name, it.url)
		}
	}
	rep.Markdown = b.String()
	return rep, nil
}

// pageURL returns the page of an entry on the wanted platform. If the entry
// isn't there, it returns its page on the other platform, and false.
func pageURL(m pack.Mod, platform string, cfURLs map[int]string) (url string, onPlatform bool) {
	mr, cf := "", ""
	if m.Modrinth != nil {
		mr = "https://modrinth.com/project/" + m.Modrinth.ProjectID
	}
	if m.CurseForge != nil {
		cf = cfURLs[m.CurseForge.ProjectID]
		if cf == "" {
			cf = fmt.Sprintf("https://www.curseforge.com/projects/%d", m.CurseForge.ProjectID)
		}
	}

	if platform == "curseforge" {
		if cf != "" {
			return cf, true
		}
		return mr, false
	}
	if mr != "" {
		return mr, true
	}
	return cf, false
}

// sortKey sorts case-insensitively, ignoring leading symbols such as "[".
func sortKey(name string) string {
	trimmed := strings.TrimLeftFunc(name, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	return strings.ToLower(trimmed)
}

// anchor builds the GitHub anchor of a heading: "Mods used" -> "mods-used".
func anchor(title string) string {
	return strings.ReplaceAll(strings.ToLower(title), " ", "-")
}
