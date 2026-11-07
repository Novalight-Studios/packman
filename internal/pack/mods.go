package pack

import (
	"fmt"
	"strconv"
	"strings"
)

// FindSame returns the entry of this kind that is the same project as m:
// same slug, or same Modrinth / CurseForge project ID. Matching on IDs
// avoids duplicates when the slug differs between platforms.
func (p *Pack) FindSame(kind Kind, m Mod) *Mod {
	list := *p.Items(kind)
	for i := range list {
		e := &list[i]
		switch {
		case e.Slug == m.Slug,
			e.Modrinth != nil && m.Modrinth != nil && e.Modrinth.ProjectID == m.Modrinth.ProjectID,
			e.CurseForge != nil && m.CurseForge != nil && e.CurseForge.ProjectID == m.CurseForge.ProjectID:
			return e
		}
	}
	return nil
}

// FillMissing copies the platform blocks that m lacks from other.
// It never overwrites existing data and reports whether anything changed.
func (m *Mod) FillMissing(other Mod) bool {
	changed := false
	if m.Modrinth == nil && other.Modrinth != nil {
		m.Modrinth = other.Modrinth
		changed = true
	}
	if m.CurseForge == nil && other.CurseForge != nil {
		m.CurseForge = other.CurseForge
		changed = true
	}
	return changed
}

// Platforms lists the platforms this entry is available on.
func (m Mod) Platforms() []string {
	var out []string
	if m.Modrinth != nil {
		out = append(out, "modrinth")
	}
	if m.CurseForge != nil {
		out = append(out, "curseforge")
	}
	return out
}

// Removed describes an entry taken out of the pack.
type Removed struct {
	Kind Kind
	Mod  Mod
}

type hit struct {
	kind  Kind
	index int
}

func (p *Pack) find(kinds []Kind, match func(Mod) bool) []hit {
	var hits []hit
	for _, k := range kinds {
		for i, m := range *p.Items(k) {
			if match(m) {
				hits = append(hits, hit{kind: k, index: i})
			}
		}
	}
	return hits
}

// lookup finds the single entry matching query (slug first, then name,
// case-insensitive) in the given kind, or in every kind if kind is empty.
func (p *Pack) lookup(kind Kind, query string) (hit, error) {
	kinds := AllKinds
	if kind != "" {
		kinds = []Kind{kind}
	}

	hits := p.find(kinds, func(m Mod) bool { return strings.EqualFold(m.Slug, query) })
	if len(hits) == 0 {
		hits = p.find(kinds, func(m Mod) bool { return strings.EqualFold(m.Name, query) })
	}
	switch len(hits) {
	case 0:
		return hit{}, fmt.Errorf("%q not found in the pack", query)
	case 1:
		return hits[0], nil
	}
	return hit{}, fmt.Errorf("%q matches several entries: use --type or the exact slug", query)
}

// Find returns the entry matching query. The pointer targets the slice
// element, so changes are kept in the pack.
func (p *Pack) Find(kind Kind, query string) (Kind, *Mod, error) {
	h, err := p.lookup(kind, query)
	if err != nil {
		return "", nil, err
	}
	return h.kind, &(*p.Items(h.kind))[h.index], nil
}

// Remove deletes the entry matching query from the pack.
func (p *Pack) Remove(kind Kind, query string) (*Removed, error) {
	h, err := p.lookup(kind, query)
	if err != nil {
		return nil, err
	}
	list := p.Items(h.kind)
	removed := (*list)[h.index]
	*list = append((*list)[:h.index], (*list)[h.index+1:]...)
	return &Removed{Kind: h.kind, Mod: removed}, nil
}

// FindProject returns the entry that is the given project (Modrinth project
// ID, or CurseForge mod ID as a string) on a platform.
func (p *Pack) FindProject(platform, id string) (Kind, *Mod) {
	for _, k := range AllKinds {
		list := *p.Items(k)
		for i := range list {
			m := &list[i]
			switch {
			case platform == "modrinth" && m.Modrinth != nil && m.Modrinth.ProjectID == id,
				platform == "curseforge" && m.CurseForge != nil && strconv.Itoa(m.CurseForge.ProjectID) == id:
				return k, m
			}
		}
	}
	return "", nil
}
