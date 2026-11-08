// Package curseforge is a minimal client for the CurseForge API v1.
// It needs an API key: https://console.curseforge.com
package curseforge

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	baseURL = "https://api.curseforge.com/v1"

	// EnvKey is the environment variable holding the API key.
	EnvKey = "CURSEFORGE_API_KEY"

	gameMinecraft = 432

	// Content types ("classes") of Minecraft on CurseForge.
	ClassMods          = 6
	ClassResourcePacks = 12
	ClassShaders       = 6552

	releaseTypeRelease = 1 // 2 = beta, 3 = alpha
)

// ErrNotFound is returned when no mod matches on CurseForge.
var ErrNotFound = errors.New("not found on CurseForge")

// ErrNoKey is returned by NewFromEnv when the API key is missing.
var ErrNoKey = errors.New(EnvKey + " is not set")

var loaderTypes = map[string]int{
	"forge":    1,
	"fabric":   4,
	"quilt":    5,
	"neoforge": 6,
}

type Client struct {
	http *http.Client
	key  string
}

func New(apiKey string) *Client {
	return &Client{
		http: &http.Client{Timeout: 30 * time.Second},
		key:  apiKey,
	}
}

// NewFromEnv reads the API key from CURSEFORGE_API_KEY.
func NewFromEnv() (*Client, error) {
	key := strings.TrimSpace(os.Getenv(EnvKey))
	if key == "" {
		return nil, ErrNoKey
	}
	return New(key), nil
}

type Mod struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	ClassID int    `json:"classId"` // ClassMods, ClassResourcePacks, ClassShaders...
}

type File struct {
	ID          int       `json:"id"`
	FileName    string    `json:"fileName"`
	DownloadURL string    `json:"downloadUrl"` // empty if the author disallows third-party distribution
	FileDate    time.Time `json:"fileDate"`
	ReleaseType int       `json:"releaseType"`
	IsAvailable bool      `json:"isAvailable"`
	Dependencies []struct {
		ModID        int `json:"modId"`
		RelationType int `json:"relationType"` // 3 = required dependency
	} `json:"dependencies"`
	Hashes      []struct {
		Value string `json:"value"`
		Algo  int    `json:"algo"` // 1 = sha1, 2 = md5
	} `json:"hashes"`
}

// SHA1 returns the file's sha1, or "" if CurseForge doesn't give one.
func (f File) SHA1() string {
	for _, h := range f.Hashes {
		if h.Algo == 1 {
			return h.Value
		}
	}
	return ""
}

// Check verifies that the API key is accepted by CurseForge.
func (c *Client) Check() error {
	var v struct{}
	return c.do(http.MethodGet, "/games/"+strconv.Itoa(gameMinecraft), nil, nil, &v)
}

// FindModBySlug looks a project up by its exact slug. classID restricts the
// search to one content type (ClassMods...); 0 searches mods, resource packs
// and shaders, preferring them in that order.
func (c *Client) FindModBySlug(slug string, classID int) (*Mod, error) {
	q := url.Values{}
	q.Set("gameId", strconv.Itoa(gameMinecraft))
	q.Set("slug", slug)
	if classID != 0 {
		q.Set("classId", strconv.Itoa(classID))
	}

	var resp struct {
		Data []Mod `json:"data"`
	}
	if err := c.do(http.MethodGet, "/mods/search", q, nil, &resp); err != nil {
		return nil, err
	}

	var matches []Mod
	for _, m := range resp.Data {
		if strings.EqualFold(m.Slug, slug) {
			matches = append(matches, m)
		}
	}
	if classID != 0 { // the API already filtered by class
		if len(matches) > 0 {
			return &matches[0], nil
		}
		return nil, ErrNotFound
	}
	for _, class := range []int{ClassMods, ClassResourcePacks, ClassShaders} {
		for i := range matches {
			if matches[i].ClassID == class {
				return &matches[i], nil
			}
		}
	}
	return nil, ErrNotFound
}

// ModByID fetches a mod from its project ID.
func (c *Client) ModByID(id int) (*Mod, error) {
	var resp struct {
		Data Mod `json:"data"`
	}
	if err := c.do(http.MethodGet, "/mods/"+strconv.Itoa(id), nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// ModIDByFingerprint finds the mod a file belongs to (see Fingerprint).
func (c *Client) ModIDByFingerprint(fingerprint uint32) (int, error) {
	body := map[string][]uint32{"fingerprints": {fingerprint}}

	var resp struct {
		Data struct {
			ExactMatches []struct {
				ID   int `json:"id"`
				File struct {
					ModID int `json:"modId"`
				} `json:"file"`
			} `json:"exactMatches"`
		} `json:"data"`
	}
	path := "/fingerprints/" + strconv.Itoa(gameMinecraft)
	if err := c.do(http.MethodPost, path, nil, body, &resp); err != nil {
		return 0, err
	}
	for _, m := range resp.Data.ExactMatches {
		if m.File.ModID != 0 {
			return m.File.ModID, nil
		}
		if m.ID != 0 {
			return m.ID, nil
		}
	}
	return 0, ErrNotFound
}

// LatestFile returns the newest file compatible with the given Minecraft
// version and loader, preferring stable releases over alpha/beta.
func (c *Client) LatestFile(modID int, mcVersion, loader string) (*File, error) {
	q := url.Values{}
	q.Set("gameVersion", mcVersion)
	q.Set("pageSize", "50")
	target := "Minecraft " + mcVersion
	if loader != "" { // resource packs and shaders have no mod loader
		loaderType, ok := loaderTypes[loader]
		if !ok {
			return nil, fmt.Errorf("unknown loader %q", loader)
		}
		q.Set("modLoaderType", strconv.Itoa(loaderType))
		target += " with " + loader
	}

	var resp struct {
		Data []File `json:"data"`
	}
	if err := c.do(http.MethodGet, "/mods/"+strconv.Itoa(modID)+"/files", q, nil, &resp); err != nil {
		return nil, err
	}

	// The API doesn't guarantee any order: pick by date ourselves.
	var newestRelease, newest *File
	for i := range resp.Data {
		f := &resp.Data[i]
		if !f.IsAvailable {
			continue
		}
		if newest == nil || f.FileDate.After(newest.FileDate) {
			newest = f
		}
		if f.ReleaseType == releaseTypeRelease &&
			(newestRelease == nil || f.FileDate.After(newestRelease.FileDate)) {
			newestRelease = f
		}
	}
	switch {
	case newestRelease != nil:
		return newestRelease, nil
	case newest != nil:
		return newest, nil
	}
	return nil, fmt.Errorf("no file for %s", target)
}

func (c *Client) do(method, path string, q url.Values, body, out any) error {
	u := baseURL + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}

	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, u, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-api-key", c.key)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return json.NewDecoder(resp.Body).Decode(out)
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusUnauthorized, http.StatusForbidden:
		return errors.New("CurseForge rejected the API key (check " + EnvKey + ")")
	case http.StatusTooManyRequests:
		return errors.New("CurseForge rate limit reached, try again in a minute")
	default:
		return fmt.Errorf("CurseForge answered %s", resp.Status)
	}
}

// File fetches one file of a mod.
func (c *Client) File(modID, fileID int) (*File, error) {
	var resp struct {
		Data File `json:"data"`
	}
	path := "/mods/" + strconv.Itoa(modID) + "/files/" + strconv.Itoa(fileID)
	if err := c.do(http.MethodGet, path, nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// RequiredMods returns the IDs of the mods this file requires.
func (f File) RequiredMods() []int {
	const relationRequired = 3
	var ids []int
	for _, d := range f.Dependencies {
		if d.RelationType == relationRequired {
			ids = append(ids, d.ModID)
		}
	}
	return ids
}

// WebsiteURLs returns the page URL of several mods at once, by mod ID.
func (c *Client) WebsiteURLs(ids []int) (map[int]string, error) {
	const batch = 50
	urls := make(map[int]string, len(ids))

	for start := 0; start < len(ids); start += batch {
		end := start + batch
		if end > len(ids) {
			end = len(ids)
		}

		var resp struct {
			Data []struct {
				ID    int `json:"id"`
				Links struct {
					WebsiteURL string `json:"websiteUrl"`
				} `json:"links"`
			} `json:"data"`
		}
		body := map[string]any{"modIds": ids[start:end]}
		if err := c.do(http.MethodPost, "/mods", nil, body, &resp); err != nil {
			return nil, err
		}
		for _, m := range resp.Data {
			urls[m.ID] = m.Links.WebsiteURL
		}
	}
	return urls, nil
}
