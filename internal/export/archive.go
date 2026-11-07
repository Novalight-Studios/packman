package export

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// archive is a zip being written. It remembers the names already used so
// the same entry is never written twice.
type archive struct {
	zw    *zip.Writer
	names map[string]bool
}

// build writes an archive to out. The file is written under a temporary name
// and only moved into place on success, so a failed export leaves nothing behind.
func build(out string, fill func(a *archive) error) (err error) {
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	tmp := out + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			f.Close()
			os.Remove(tmp)
		}
	}()

	a := &archive{zw: zip.NewWriter(f), names: map[string]bool{}}
	if err = fill(a); err != nil {
		return err
	}
	if err = a.zw.Close(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, out)
}

func (a *archive) has(name string) bool {
	return a.names[name]
}

func (a *archive) addJSON(name string, v any) error {
	w, err := a.zw.Create(name)
	if err != nil {
		return err
	}
	a.names[name] = true
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// addFile copies the local file src into the archive as name.
func (a *archive) addFile(name, src string) error {
	if a.names[name] {
		return fmt.Errorf("duplicate entry %s", name)
	}
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	w, err := a.zw.Create(name)
	if err != nil {
		return err
	}
	a.names[name] = true
	_, err = io.Copy(w, f)
	return err
}

// addOverrides copies every file of <dir>/overrides into the archive under
// "overrides/", keeping the folder structure.
func (a *archive) addOverrides(dir string) error {
	root := filepath.Join(dir, overridesDir)
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil
	}
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		switch d.Name() {
		case ".DS_Store", "Thumbs.db", "desktop.ini":
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		return a.addFile(overridesDir+"/"+filepath.ToSlash(rel), p)
	})
}

// cleanRel validates a relative path coming from a platform API.
func cleanRel(p string) (string, error) {
	c := path.Clean(strings.ReplaceAll(p, "\\", "/"))
	if c == "." || c == ".." || strings.HasPrefix(c, "../") || path.IsAbs(c) {
		return "", fmt.Errorf("unsafe path %q", p)
	}
	return c, nil
}

// safeFileName keeps only the file name of a name coming from a platform API.
func safeFileName(name string) (string, error) {
	base := path.Base(strings.ReplaceAll(name, "\\", "/"))
	if base == "." || base == ".." || base == "/" || base == "" {
		return "", fmt.Errorf("invalid file name %q", name)
	}
	return base, nil
}
