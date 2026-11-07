package pack

import (
	"fmt"
	"path"
)

// UpdateEntry moves an entry to the latest version compatible with the pack's
// Minecraft version and loader, on each platform it is on. It looks projects
// up by ID, so slugs that differ between platforms don't matter.
//
// It returns what changed (empty: already up to date), what couldn't be
// checked (e.g. no compatible version), and the required dependencies of the
// latest version, to be given to AddDeps. Platforms whose client is nil are
// skipped.
func UpdateEntry(s Sources, p *Pack, kind Kind, m *Mod) (changes, problems []string, deps []Dep) {
	if m.Modrinth != nil && s.Modrinth != nil {
		fresh, err := fetchModrinth(s.Modrinth, p, m.Modrinth.ProjectID, kind)
		if err != nil {
			problems = append(problems, "Modrinth: "+err.Error())
		} else {
			deps = append(deps, fresh.deps...)
			if fresh.mod.Modrinth.VersionID != m.Modrinth.VersionID {
				changes = append(changes, fmt.Sprintf("modrinth: %s → %s",
					path.Base(m.Modrinth.Path), path.Base(fresh.mod.Modrinth.Path)))
				fresh.mod.Modrinth.Env = m.Modrinth.Env // keep env, it may have been edited by hand
				m.Modrinth = fresh.mod.Modrinth
			}
		}
	}

	if m.CurseForge != nil && s.CurseForge != nil {
		mod, err := s.CurseForge.ModByID(m.CurseForge.ProjectID)
		var fresh *found
		if err == nil {
			fresh, err = curseForgeMod(s.CurseForge, p, mod, kind)
		}
		if err != nil {
			problems = append(problems, "CurseForge: "+err.Error())
		} else {
			deps = append(deps, fresh.deps...)
			if fresh.mod.CurseForge.FileID != m.CurseForge.FileID {
				changes = append(changes, fmt.Sprintf("curseforge: file %d → %d",
					m.CurseForge.FileID, fresh.mod.CurseForge.FileID))
				m.CurseForge.FileID = fresh.mod.CurseForge.FileID // keep required / isLocked
			}
		}
	}
	return changes, problems, deps
}
