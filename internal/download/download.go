// Package download fetches files into a local cache.
package download

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func cacheDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "packman", "downloads")
	return dir, os.MkdirAll(dir, 0o755)
}

// Cached returns the path of a local copy of the file at url.
//
// The copy lives in the user cache directory, named after its sha1, so a
// file is downloaded once. The download is streamed to disk (resource packs
// and shaders can be big) and checked against sha1Hex when it is not empty.
func Cached(url, sha1Hex string) (string, error) {
	dir, err := cacheDir()
	if err != nil {
		return "", err
	}

	want := strings.ToLower(strings.TrimSpace(sha1Hex))
	if want != "" {
		cached := filepath.Join(dir, want)
		if _, err := os.Stat(cached); err == nil {
			return cached, nil
		}
	}

	client := &http.Client{Timeout: 15 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed (%s): %s", resp.Status, url)
	}

	tmp, err := os.CreateTemp(dir, "download-*")
	if err != nil {
		return "", err
	}
	h := sha1.New()
	_, err = io.Copy(io.MultiWriter(tmp, h), resp.Body)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmp.Name())
		return "", err
	}

	got := hex.EncodeToString(h.Sum(nil))
	if want != "" && got != want {
		os.Remove(tmp.Name())
		return "", fmt.Errorf("sha1 mismatch for %s", url)
	}

	final := filepath.Join(dir, got)
	if err := os.Rename(tmp.Name(), final); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	return final, nil
}
