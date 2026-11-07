package export

import (
	"errors"

	"github.com/Alfakynz/packman/internal/download"
	"github.com/Alfakynz/packman/internal/pack"
)

var mrLoaderKey = map[string]string{
	"fabric":   "fabric-loader",
	"quilt":    "quilt-loader",
	"forge":    "forge",
	"neoforge": "neoforge",
}

// mrFile is an entry of "files" in modrinth.index.json.
type mrFile struct {
	Path      string      `json:"path"`
	Hashes    pack.Hashes `json:"hashes"`
	Env       pack.Env    `json:"env"`
	Downloads []string    `json:"downloads"`
	FileSize  int64       `json:"fileSize"`
}

// Modrinth writes a .mrpack. Entries that are on CurseForge only are
// downloaded and bundled under overrides/.
func Modrinth(p *pack.Pack, o Options) (*Report, error) {
	if err := validate(p); err != nil {
		return nil, err
	}
	rep := &Report{Path: outPath(o, p, ".mrpack")}

	err := build(rep.Path, func(a *archive) error {
		// The user's overrides go first: they win over bundled files.
		if err := a.addOverrides(o.Dir); err != nil {
			return err
		}

		files := []mrFile{}
		for _, e := range entries(p) {
			m := e.mod
			var err error

			switch {
			case m.Modrinth != nil:
				var rel string
				if rel, err = cleanRel(m.Modrinth.Path); err == nil {
					files = append(files, mrFile{
						Path:      rel,
						Hashes:    m.Modrinth.Hashes,
						Env:       m.Modrinth.Env,
						Downloads: m.Modrinth.Downloads,
						FileSize:  m.Modrinth.FileSize,
					})
				}
			case m.CurseForge != nil:
				o.progressf("Bundling %s (CurseForge only)", m.Name)
				if err = bundleFromCurseForge(a, o, e); err == nil {
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

		return a.addJSON("modrinth.index.json", map[string]any{
			"formatVersion": 1,
			"game":          "minecraft",
			"versionId":     p.Version,
			"name":          p.Name,
			"files":         files,
			"dependencies": map[string]string{
				"minecraft":          p.MCVersion,
				mrLoaderKey[p.Loader]: p.LoaderVersion,
			},
		})
	})
	if err != nil {
		return nil, err
	}
	return rep, nil
}

// bundleFromCurseForge downloads a CurseForge-only entry and stores it under
// overrides/<mods|resourcepacks|shaderpacks>/.
func bundleFromCurseForge(a *archive, o Options, e entry) error {
	if o.CurseForge == nil {
		return errors.New("CurseForge is not available")
	}
	c, err := o.CurseForge()
	if err != nil {
		return err
	}
	cf := e.mod.CurseForge
	f, err := c.File(cf.ProjectID, cf.FileID)
	if err != nil {
		return err
	}
	if f.DownloadURL == "" {
		return errors.New("its author doesn't allow third-party distribution, so it can't be bundled")
	}
	name, err := safeFileName(f.FileName)
	if err != nil {
		return err
	}

	zipPath := overridesDir + "/" + e.kind.Dir() + "/" + name
	if a.has(zipPath) { // already provided by the user's overrides
		return nil
	}
	local, err := download.Cached(f.DownloadURL, f.SHA1())
	if err != nil {
		return err
	}
	return a.addFile(zipPath, local)
}
