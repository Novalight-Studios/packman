package pack

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Novalight-Studios/packman/internal/curseforge"
	"github.com/Novalight-Studios/packman/internal/modrinth"
)

// Largest file downloaded to match it on the other platform by fingerprint.
const maxMatchSize = 200 << 20 // 200 MiB

// Sources are the platforms to query. A nil client means "disabled".
type Sources struct {
	Modrinth   *modrinth.Client
	CurseForge *curseforge.Client
}

// Options tunes a lookup.
type Options struct {
	Kind       Kind   // empty: detect (mod, resource pack or shader)
	Modrinth   string // slug on Modrinth, if different from the one given
	CurseForge string // slug on CurseForge, if different from the one given
}

// Dep is a required dependency of an entry: it has to be in the pack too.
type Dep struct {
	Platform string // "modrinth" (project ID) or "curseforge" (mod ID)
	ID       string
	By       string // name of the entry that requires it
}

// Result is a resolved entry, ready to be added to the pack.
type Result struct {
	Mod      *Mod
	Kind     Kind
	Deps     []Dep // required dependencies, see AddDeps
	Warnings []string
}

// found is what a lookup on one platform brings back.
type found struct {
	mod  *Mod
	kind Kind
	deps []Dep
	sha1 string // sha1 of the chosen CurseForge file, to match it on Modrinth
}

// Resolve looks an entry up on every enabled platform and merges the results
// into a single Mod. The kind (mod, resource pack, shader) comes from
// Options, or from Modrinth's project type.
//
// When the entry is found on one platform but not on the other under the same
// slug, it is matched through the file itself: by sha1 on Modrinth, by
// fingerprint on CurseForge. A platform where the entry isn't found is
// reported in warnings, as long as at least one platform has it.
func Resolve(s Sources, p *Pack, slug string, o Options) (*Result, error) {
	var mr, cf *found
	var mrErr, cfErr error
	kind := o.Kind

	if s.Modrinth != nil {
		mr, mrErr = fetchModrinth(s.Modrinth, p, firstNonEmpty(o.Modrinth, slug), kind)
		if mr != nil {
			kind = mr.kind // restricts the CurseForge search to the same type
		}
	}
	if s.CurseForge != nil {
		cfSlug := firstNonEmpty(o.CurseForge, slug)
		if o.CurseForge == "" && mr != nil {
			cfSlug = mr.mod.Slug // canonical slug: the user may have given an ID
		}
		cf, cfErr = fetchCurseForge(s.CurseForge, p, cfSlug, kind)
	}

	// Found on Modrinth only -> look for the same file on CurseForge.
	if mr != nil && cf == nil && s.CurseForge != nil && o.CurseForge == "" &&
		errors.Is(cfErr, curseforge.ErrNotFound) {
		m, err := matchOnCurseForge(s.CurseForge, p, mr)
		switch {
		case err == nil:
			cf, cfErr = m, nil
		case !errors.Is(err, curseforge.ErrNotFound):
			cfErr = err
		}
	}
	// Found on CurseForge only -> look for the same file on Modrinth.
	if cf != nil && mr == nil && s.Modrinth != nil && o.Modrinth == "" &&
		errors.Is(mrErr, modrinth.ErrNotFound) {
		m, err := matchOnModrinth(s.Modrinth, p, cf.sha1, cf.kind)
		switch {
		case err == nil:
			mr, mrErr = m, nil
		case !errors.Is(err, modrinth.ErrNotFound):
			mrErr = err
		}
	}

	var problems []string
	if mrErr != nil {
		problems = append(problems, describe("Modrinth", mrErr, modrinth.ErrNotFound, "--modrinth-slug"))
	}
	if cfErr != nil {
		problems = append(problems, describe("CurseForge", cfErr, curseforge.ErrNotFound, "--curseforge-slug"))
	}

	switch {
	case mr != nil && cf != nil:
		mr.mod.CurseForge = cf.mod.CurseForge
		return &Result{Mod: mr.mod, Kind: mr.kind, Deps: append(mr.deps, cf.deps...)}, nil
	case mr != nil:
		return &Result{Mod: mr.mod, Kind: mr.kind, Deps: mr.deps, Warnings: problems}, nil
	case cf != nil:
		return &Result{Mod: cf.mod, Kind: cf.kind, Deps: cf.deps, Warnings: problems}, nil
	case len(problems) == 0:
		return nil, errors.New("no platform enabled")
	}
	return nil, errors.New(strings.Join(problems, "; "))
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// "not found on X" already names the platform; other errors don't.
func describe(platform string, err, notFound error, flag string) string {
	if errors.Is(err, notFound) {
		return err.Error() + " (different slug there? use " + flag + ")"
	}
	return platform + ": " + err.Error()
}

// fetchModrinth resolves a project on Modrinth (slug or project ID) for the
// pack's Minecraft version and loader. want restricts the type (empty: any
// supported type). The CurseForge block is left empty.
func fetchModrinth(c *modrinth.Client, p *Pack, slug string, want Kind) (*found, error) {
	proj, err := c.Project(slug)
	if err != nil {
		return nil, err
	}
	kind := Kind(proj.ProjectType)
	if !kind.Valid() {
		return nil, fmt.Errorf("%q is a %s, which packman can't handle", proj.Slug, proj.ProjectType)
	}
	if want != "" && kind != want {
		return nil, fmt.Errorf("%q is a %s, not a %s", proj.Slug, kind.Label(), want.Label())
	}

	ver, err := c.LatestVersion(proj.ID, p.MCVersion, kind.loader(p))
	if err != nil {
		return nil, err
	}
	file, ok := ver.PrimaryFile()
	if !ok {
		return nil, errors.New("this version has no file")
	}

	var deps []Dep
	for _, id := range ver.RequiredProjects() {
		deps = append(deps, Dep{Platform: "modrinth", ID: id, By: proj.Title})
	}

	return &found{
		kind: kind,
		deps: deps,
		mod: &Mod{
			Slug: proj.Slug,
			Name: proj.Title,
			Modrinth: &Modrinth{
				ProjectID: proj.ID,
				VersionID: ver.ID,
				Path:      kind.Dir() + "/" + file.Filename,
				Hashes:    Hashes{Sha1: file.Hashes.Sha1, Sha512: file.Hashes.Sha512},
				Env:       Env{Client: normalizeEnv(proj.ClientSide), Server: normalizeEnv(proj.ServerSide)},
				Downloads: []string{file.URL},
				FileSize:  file.Size,
			},
		},
	}, nil
}

// fetchCurseForge resolves a project on CurseForge by slug. want restricts the
// type (empty: any supported type). The Modrinth block is left empty.
func fetchCurseForge(c *curseforge.Client, p *Pack, slug string, want Kind) (*found, error) {
	mod, err := c.FindModBySlug(slug, want.curseForgeClass())
	if err != nil {
		return nil, err
	}
	return curseForgeMod(c, p, mod, want)
}

func curseForgeMod(c *curseforge.Client, p *Pack, mod *curseforge.Mod, want Kind) (*found, error) {
	kind := want
	if kind == "" {
		kind = kindFromClass(mod.ClassID)
	}
	if !kind.Valid() {
		return nil, fmt.Errorf("%q isn't a mod, resource pack or shader", mod.Slug)
	}
	file, err := c.LatestFile(mod.ID, p.MCVersion, kind.loader(p))
	if err != nil {
		return nil, err
	}

	var deps []Dep
	for _, id := range file.RequiredMods() {
		deps = append(deps, Dep{Platform: "curseforge", ID: strconv.Itoa(id), By: mod.Name})
	}

	return &found{
		kind: kind,
		deps: deps,
		sha1: file.SHA1(),
		mod: &Mod{
			Slug: mod.Slug,
			Name: mod.Name,
			CurseForge: &CurseForge{
				ProjectID: mod.ID,
				FileID:    file.ID,
				Required:  true,
				IsLocked:  false,
			},
		},
	}, nil
}

// matchOnCurseForge downloads the Modrinth file and asks CurseForge which
// project it belongs to, through its fingerprint.
func matchOnCurseForge(c *curseforge.Client, p *Pack, mr *found) (*found, error) {
	if mr.mod.Modrinth == nil || len(mr.mod.Modrinth.Downloads) == 0 {
		return nil, curseforge.ErrNotFound
	}
	if mr.mod.Modrinth.FileSize > maxMatchSize {
		return nil, errors.New("file too large to match automatically (use --curseforge-slug)")
	}
	data, err := download(mr.mod.Modrinth.Downloads[0], mr.mod.Modrinth.Hashes.Sha1)
	if err != nil {
		return nil, err
	}
	modID, err := c.ModIDByFingerprint(curseforge.Fingerprint(data))
	if err != nil {
		return nil, err
	}
	mod, err := c.ModByID(modID)
	if err != nil {
		return nil, err
	}
	return curseForgeMod(c, p, mod, mr.kind)
}

// matchOnModrinth asks Modrinth which version a CurseForge file is, by sha1.
func matchOnModrinth(c *modrinth.Client, p *Pack, sha1Hex string, want Kind) (*found, error) {
	if sha1Hex == "" {
		return nil, modrinth.ErrNotFound
	}
	ver, err := c.VersionByHash(sha1Hex)
	if err != nil {
		return nil, err
	}
	return fetchModrinth(c, p, ver.ProjectID, want)
}

// download fetches a file in memory and checks it against its sha1.
func download(url, wantSHA1 string) ([]byte, error) {
	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: %s", resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxMatchSize))
	if err != nil {
		return nil, err
	}
	sum := sha1.Sum(data)
	if wantSHA1 != "" && !strings.EqualFold(hex.EncodeToString(sum[:]), wantSHA1) {
		return nil, errors.New("downloaded file doesn't match its sha1")
	}
	return data, nil
}

// modrinth.index.json only accepts required | optional | unsupported.
func normalizeEnv(s string) string {
	switch s {
	case "required", "optional", "unsupported":
		return s
	default: // "unknown"
		return "required"
	}
}
