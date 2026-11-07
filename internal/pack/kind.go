package pack

import "github.com/Alfakynz/packman/internal/curseforge"

// Kind is the type of content an entry of the pack is.
// The values are the same as Modrinth's project_type.
type Kind string

const (
	KindMod          Kind = "mod"
	KindResourcePack Kind = "resourcepack"
	KindShader       Kind = "shader"
)

var AllKinds = []Kind{KindMod, KindResourcePack, KindShader}

func (k Kind) Valid() bool {
	return k == KindMod || k == KindResourcePack || k == KindShader
}

// Label is the human-readable name, for messages.
func (k Kind) Label() string {
	switch k {
	case KindResourcePack:
		return "resource pack"
	case KindShader:
		return "shader"
	default:
		return string(k)
	}
}

// Dir is the Minecraft folder files of this kind go to.
func (k Kind) Dir() string {
	switch k {
	case KindResourcePack:
		return "resourcepacks"
	case KindShader:
		return "shaderpacks"
	default:
		return "mods"
	}
}

// curseForgeClass is the CurseForge class of this kind (0 if unknown).
func (k Kind) curseForgeClass() int {
	switch k {
	case KindMod:
		return curseforge.ClassMods
	case KindResourcePack:
		return curseforge.ClassResourcePacks
	case KindShader:
		return curseforge.ClassShaders
	default:
		return 0
	}
}

func kindFromClass(id int) Kind {
	switch id {
	case curseforge.ClassMods:
		return KindMod
	case curseforge.ClassResourcePacks:
		return KindResourcePack
	case curseforge.ClassShaders:
		return KindShader
	default:
		return ""
	}
}

// loader is the mod loader to filter versions by: only mods have one.
func (k Kind) loader(p *Pack) string {
	if k == KindMod {
		return p.Loader
	}
	return ""
}

// Items returns the list holding entries of the given kind.
func (p *Pack) Items(k Kind) *[]Mod {
	switch k {
	case KindResourcePack:
		return &p.ResourcesPacks
	case KindShader:
		return &p.Shaders
	default:
		return &p.Mods
	}
}
