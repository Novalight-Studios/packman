package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Novalight-Studios/packman/internal/pack"
)

var initOpts struct {
	name string
	mc string
	loader string
	loaderVersion string
  version string
}

var initCmd = &cobra.Command{
	Use: "init",
	Short: "Creates a new modpack in the current directory",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if pack.Exists(".") {
			return fmt.Errorf("%s already exists", pack.FileName)
		}
		p := pack.Pack{
			Name: initOpts.name,
      Version: initOpts.version,
			MCVersion: initOpts.mc,
			Loader: initOpts.loader,
			LoaderVersion: initOpts.loaderVersion,
			Mods: []pack.Mod{},
			ResourcesPacks: []pack.Mod{},
			Shaders: []pack.Mod{},
		}
		if err := pack.Save(".", p); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Pack %q v%s created (Minecraft %s, %s %s)\n", p.Name, p.Version, p.MCVersion, p.Loader, p.LoaderVersion)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	f := initCmd.Flags()
	f.StringVarP(&initOpts.name, "name", "n", "My Modpack", "name of the modpack")
	f.StringVar(&initOpts.mc, "mc", "1.21.1", "Minecraft version")
	f.StringVarP(&initOpts.loader, "loader", "l", "fabric", "fabric | forge | neoforge | quilt")
	f.StringVar(&initOpts.loaderVersion, "loader-version", "0.19.5", "loader version")
	f.StringVar(&initOpts.version, "version", "1.0.0", "modpack version")
}
