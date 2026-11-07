package cmd

import (
	"fmt"
	"strings"

	"github.com/Novalight-Studios/packman/internal/pack"
)

const typeFlagHelp = "all | mod | resourcepack | shader"

// parseKind converts a --type value. "" and "all" mean "any type".
func parseKind(s string) (pack.Kind, error) {
	switch strings.ToLower(s) {
	case "", "all":
		return "", nil
	case "mod", "mods":
		return pack.KindMod, nil
	case "resourcepack", "resourcepacks", "resource-pack", "resource-packs":
		return pack.KindResourcePack, nil
	case "shader", "shaders", "shaderpack", "shaderpacks":
		return pack.KindShader, nil
	}
	return "", fmt.Errorf("invalid --type %q (%s)", s, typeFlagHelp)
}
