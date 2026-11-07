// Package export generates the import files of the modpack platforms
// (Modrinth .mrpack, CurseForge .zip) from packman.json.
//
// An entry that exists on the target platform is referenced there. An entry
// that exists only on the other platform is downloaded and bundled in the
// archive as an override, so the exported pack is always complete.
package export

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Novalight-Studios/packman/internal/curseforge"
	"github.com/Novalight-Studios/packman/internal/pack"
)

const overridesDir = "overrides"

type Options struct {
	Dir         string // project folder (holds packman.json and overrides/)
	OutDir      string // where the exported files go
	SkipMissing bool   // leave out entries that can't be included, instead of failing

	// CurseForge returns a client, created on demand: it is only needed to
	// bundle entries that exist on CurseForge only.
	CurseForge func() (*curseforge.Client, error)

	// Progress, if set, is told about slow steps (downloads).
	Progress func(msg string)
}

type Report struct {
	Path    string   // exported file
	Bundled []string // entries shipped as overrides (not on the target platform)
	Skipped []string // entries left out, with the reason (SkipMissing only)
}

type entry struct {
	kind pack.Kind
	mod  pack.Mod
}

func entries(p *pack.Pack) []entry {
	var out []entry
	for _, k := range pack.AllKinds {
		for _, m := range *p.Items(k) {
			out = append(out, entry{kind: k, mod: m})
		}
	}
	return out
}

func (o Options) progressf(format string, args ...any) {
	if o.Progress != nil {
		o.Progress(fmt.Sprintf(format, args...))
	}
}

// fail handles an entry that can't be included: it is either an error,
// or recorded in the report when SkipMissing is set.
func (o Options) fail(rep *Report, e entry, err error) error {
	if o.SkipMissing {
		rep.Skipped = append(rep.Skipped, fmt.Sprintf("%s: %v", e.mod.Name, err))
		return nil
	}
	return fmt.Errorf("%s: %w (use --skip-missing to leave it out)", e.mod.Name, err)
}

func validate(p *pack.Pack) error {
	if p.MCVersion == "" {
		return errors.New("mc_version is empty in packman.json")
	}
	if _, ok := mrLoaderKey[p.Loader]; !ok {
		return fmt.Errorf("unknown loader %q in packman.json", p.Loader)
	}
	if p.LoaderVersion == "" {
		return errors.New("loader_version is empty in packman.json (both formats need it)")
	}
	return nil
}

var (
	nonAlnum   = regexp.MustCompile(`[^a-z0-9]+`)
	nonVersion = regexp.MustCompile(`[^A-Za-z0-9._+-]+`)
)

// baseName builds "my-modpack-1.0.0" from the pack's name and version.
func baseName(p *pack.Pack) string {
	n := strings.Trim(nonAlnum.ReplaceAllString(strings.ToLower(p.Name), "-"), "-")
	if n == "" {
		n = "modpack"
	}
	if v := nonVersion.ReplaceAllString(p.Version, "-"); v != "" {
		n += "-" + v
	}
	return n
}

func outPath(o Options, p *pack.Pack, ext string) string {
	return filepath.Join(o.OutDir, baseName(p)+ext)
}
