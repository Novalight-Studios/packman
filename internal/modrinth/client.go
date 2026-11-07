// Package modrinth is a minimal client for the Modrinth API v2 (no API key needed).
package modrinth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const baseURL = "https://api.modrinth.com/v2"

// ErrNotFound is returned when a project does not exist on Modrinth.
var ErrNotFound = errors.New("not found on Modrinth")

type Client struct {
	http      *http.Client
	userAgent string
}

// New creates a client. Modrinth requires an identifiable User-Agent,
// e.g. "Alfakynz/packman/1.0.0".
func New(userAgent string) *Client {
	return &Client{
		http:      &http.Client{Timeout: 30 * time.Second},
		userAgent: userAgent,
	}
}

type Project struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	ProjectType string `json:"project_type"` // mod | resourcepack | shader | modpack | ...
	ClientSide  string `json:"client_side"`  // required | optional | unsupported
	ServerSide  string `json:"server_side"`
}

type Hashes struct {
	Sha1   string `json:"sha1"`
	Sha512 string `json:"sha512"`
}

type File struct {
	Hashes   Hashes `json:"hashes"`
	URL      string `json:"url"`
	Filename string `json:"filename"`
	Primary  bool   `json:"primary"`
	Size     int64  `json:"size"`
}

type Version struct {
	ID            string       `json:"id"`
	ProjectID     string       `json:"project_id"`
	VersionNumber string       `json:"version_number"`
	VersionType   string       `json:"version_type"` // release | beta | alpha
	Files         []File       `json:"files"`
	Dependencies  []Dependency `json:"dependencies"`
}

// Dependency is a link from a version to another project.
type Dependency struct {
	ProjectID      string `json:"project_id"`      // empty for an external dependency
	DependencyType string `json:"dependency_type"` // required | optional | incompatible | embedded
}

// RequiredProjects returns the IDs of the projects this version requires.
func (v Version) RequiredProjects() []string {
	var ids []string
	for _, d := range v.Dependencies {
		if d.DependencyType == "required" && d.ProjectID != "" {
			ids = append(ids, d.ProjectID)
		}
	}
	return ids
}

// PrimaryFile returns the file flagged as primary (or the first one).
func (v Version) PrimaryFile() (File, bool) {
	for _, f := range v.Files {
		if f.Primary {
			return f, true
		}
	}
	if len(v.Files) > 0 {
		return v.Files[0], true
	}
	return File{}, false
}

// Project fetches a project by slug or ID.
func (c *Client) Project(slugOrID string) (*Project, error) {
	var p Project
	if err := c.get("/project/"+url.PathEscape(slugOrID), nil, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// LatestVersion returns the newest version compatible with the given
// Minecraft version and loader, preferring stable releases over alpha/beta.
func (c *Client) LatestVersion(projectID, mcVersion, loader string) (*Version, error) {
	q := url.Values{}
	q.Set("game_versions", fmt.Sprintf("[%q]", mcVersion))
	if loader != "" { // resource packs and shaders have no mod loader
		q.Set("loaders", fmt.Sprintf("[%q]", loader))
	}

	var versions []Version
	if err := c.get("/project/"+url.PathEscape(projectID)+"/version", q, &versions); err != nil {
		return nil, err
	}
	if len(versions) == 0 {
		if loader == "" {
			return nil, fmt.Errorf("no version for Minecraft %s", mcVersion)
		}
		return nil, fmt.Errorf("no version for Minecraft %s with %s", mcVersion, loader)
	}
	// The API returns the newest first.
	for i := range versions {
		if versions[i].VersionType == "release" {
			return &versions[i], nil
		}
	}
	return &versions[0], nil
}

func (c *Client) get(path string, q url.Values, out any) error {
	u := baseURL + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", c.userAgent)

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
	case http.StatusTooManyRequests:
		return errors.New("Modrinth rate limit reached, try again in a minute")
	default:
		return fmt.Errorf("Modrinth answered %s", resp.Status)
	}
}

// VersionByHash finds the version a file belongs to from its sha1.
// This is how a mod found on another platform is matched on Modrinth.
func (c *Client) VersionByHash(sha1 string) (*Version, error) {
	q := url.Values{}
	q.Set("algorithm", "sha1")
	var v Version
	if err := c.get("/version_file/"+url.PathEscape(sha1), q, &v); err != nil {
		return nil, err
	}
	return &v, nil
}
