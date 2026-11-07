package export

import (
	"errors"

	"github.com/Novalight-Studios/packman/internal/download"
	"github.com/Novalight-Studios/packman/internal/pack"
)

// CurseForge writes a .zip importable by the CurseForge app. Entries that
// are on Modrinth only are downloaded and bundled under overrides/.
func CurseForge(p *pack.Pack, o Options) (*Report, error) {
	if err := validate(p); err != nil {
		return nil, err
	}
	rep := &Report{Path: outPath(o, p, ".zip")}

	err := build(rep.Path, func(a *archive) error {
		// The user's overrides go first: they win over bundled files.
		if err := a.addOverrides(o.Dir); err != nil {
			return err
		}

		files := []pack.CurseForge{} // same JSON fields as manifest.json "files"
		for _, e := range entries(p) {
			m := e.mod
			var err error

			switch {
			case m.CurseForge != nil:
				files = append(files, *m.CurseForge)
			case m.Modrinth != nil:
				o.progressf("Bundling %s (Modrinth only)", m.Name)
				if err = bundleFromModrinth(a, e); err == nil {
					rep.Bundled = append(rep.Bundled, m.Name)
				}
			default:
				err = errors.New("no Modrinth or CurseForge data")
			}

			if err != nil {
				if ferr := o.fail(rep, e, err); ferr != nil {
					return ferr
				}
			}
		}

		return a.addJSON("manifest.json", map[string]any{
			"minecraft": map[string]any{
				"version": p.MCVersion,
				"modLoaders": []map[string]any{
					{"id": p.Loader + "-" + p.LoaderVersion, "primary": true},
				},
			},
			"manifestType":    "minecraftModpack",
			"manifestVersion": 1,
			"name":            p.Name,
			"version":         p.Version,
			"files":           files,
			"overrides":       overridesDir,
		})
	})
	if err != nil {
		return nil, err
	}
	return rep, nil
}

// bundleFromModrinth downloads a Modrinth-only entry and stores it under
// overrides/, at the path recorded in packman.json (mods/, resourcepacks/...).
func bundleFromModrinth(a *archive, e entry) error {
	mr := e.mod.Modrinth
	if len(mr.Downloads) == 0 {
		return errors.New("no download URL")
	}
	rel, err := cleanRel(mr.Path)
	if err != nil {
		return err
	}

	zipPath := overridesDir + "/" + rel
	if a.has(zipPath) { // already provided by the user's overrides
		return nil
	}
	local, err := download.Cached(mr.Downloads[0], mr.Hashes.Sha1)
	if err != nil {
		return err
	}
	return a.addFile(zipPath, local)
}
