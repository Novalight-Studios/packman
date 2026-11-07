package pack

import (
	"errors"
	"fmt"
	"strconv"
)

// Added is an entry added to the pack because another one requires it.
type Added struct {
	Kind     Kind
	Mod      Mod
	Required string // name of the entry that requires it
}

// AddDeps adds the dependencies that are not in the pack yet, then theirs,
// and so on. A dependency is looked up on every enabled platform, like any
// other entry. Those that can't be resolved (no version for the pack's
// Minecraft version, for instance) are reported in problems and don't stop
// the others.
//
// It appends to the pack's lists: call it once the loop that holds pointers
// to entries (Find, Items) is over.
func AddDeps(s Sources, p *Pack, deps []Dep) (added []Added, problems []string) {
	visited := map[string]bool{}
	queue := append([]Dep(nil), deps...)

	for len(queue) > 0 {
		d := queue[0]
		queue = queue[1:]

		key := d.Platform + ":" + d.ID
		if visited[key] {
			continue
		}
		visited[key] = true
		if _, m := p.FindProject(d.Platform, d.ID); m != nil {
			continue // already in the pack
		}

		slug, kind, err := s.depSlug(d)
		if err != nil {
			problems = append(problems, fmt.Sprintf("dependency %s %s of %s: %v", d.Platform, d.ID, d.By, err))
			continue
		}
		res, err := Resolve(s, p, slug, Options{Kind: kind})
		if err != nil {
			problems = append(problems, fmt.Sprintf("dependency %s of %s: %v", slug, d.By, err))
			continue
		}
		for _, w := range res.Warnings {
			problems = append(problems, fmt.Sprintf("dependency %s: %s", res.Mod.Slug, w))
		}

		// The same project can be reached with its ID on the other platform.
		if res.Mod.Modrinth != nil {
			visited["modrinth:"+res.Mod.Modrinth.ProjectID] = true
		}
		if res.Mod.CurseForge != nil {
			visited["curseforge:"+strconv.Itoa(res.Mod.CurseForge.ProjectID)] = true
		}
		if p.FindSame(res.Kind, *res.Mod) != nil {
			continue
		}

		list := p.Items(res.Kind)
		*list = append(*list, *res.Mod)
		added = append(added, Added{Kind: res.Kind, Mod: *res.Mod, Required: d.By})
		queue = append(queue, res.Deps...)
	}
	return added, problems
}

// depSlug turns a dependency into a slug (and kind, if known) for Resolve.
func (s Sources) depSlug(d Dep) (string, Kind, error) {
	if d.Platform == "modrinth" {
		if s.Modrinth == nil {
			return "", "", errors.New("Modrinth is not available")
		}
		return d.ID, "", nil // Modrinth accepts project IDs where it expects slugs
	}

	if s.CurseForge == nil {
		return "", "", errors.New("CurseForge is not available")
	}
	id, err := strconv.Atoi(d.ID)
	if err != nil {
		return "", "", err
	}
	mod, err := s.CurseForge.ModByID(id)
	if err != nil {
		return "", "", err
	}
	return mod.Slug, kindFromClass(mod.ClassID), nil
}
