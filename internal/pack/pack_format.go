package pack

type Pack struct {
	Name           string `json:"name"`
	Version        string `json:"version"`
	MCVersion      string `json:"mc_version"`
	Loader         string `json:"loader"`
	LoaderVersion  string `json:"loader_version"`
	Mods           []Mod  `json:"mods"`
	ResourcesPacks []Mod  `json:"resources_packs"`
	Shaders        []Mod  `json:"shaders"`
}

// Mod is an entry of the pack: a mod, a resource pack or a shader
// (the list it sits in tells which). It holds everything needed to
// generate both export formats. Each platform block is optional.
type Mod struct {
	Slug       string      `json:"slug"` // unique key, used by add/remove/update
	Name       string      `json:"name"`
	Modrinth   *Modrinth   `json:"modrinth,omitempty"`
	CurseForge *CurseForge `json:"curseforge,omitempty"`
}

// Modrinth maps 1:1 to an entry of modrinth.index.json ("files"),
// plus projectID/versionID to be able to check for updates later.
type Modrinth struct {
	ProjectID string   `json:"projectID"`
	VersionID string   `json:"versionID"`
	Path      string   `json:"path"` // e.g. "mods/sodium-fabric-0.6.0.jar"
	Hashes    Hashes   `json:"hashes"`
	Env       Env      `json:"env"`
	Downloads []string `json:"downloads"`
	FileSize  int64    `json:"fileSize"`
}

type Hashes struct {
	Sha1   string `json:"sha1"`
	Sha512 string `json:"sha512"`
}

type Env struct {
	Client string `json:"client"` // required | optional | unsupported
	Server string `json:"server"`
}

// CurseForge maps 1:1 to an entry of manifest.json ("files").
type CurseForge struct {
	ProjectID int  `json:"projectID"`
	FileID    int  `json:"fileID"`
	Required  bool `json:"required"`
	IsLocked  bool `json:"isLocked"`
}
