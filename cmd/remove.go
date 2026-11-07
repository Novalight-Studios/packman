package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Novalight-Studios/packman/internal/pack"
)

var removeOpts struct {
	kind string
}

var removeCmd = &cobra.Command{
	Use:     "remove <name>...",
	Aliases: []string{"rm"},
	Short:   "Removes mods, resource packs or shaders from the pack",
	Long: "Removes entries from packman.json, looked up by slug or name among your mods,\n" +
		"resource packs and shaders. Use --type if the same name exists in several types.",
	Example: "  packman remove sodium\n" +
		"  packman remove sodium lithium\n" +
		"  packman remove complementary-reimagined --type shader",
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		kind, err := parseKind(removeOpts.kind)
		if err != nil {
			return err
		}
		p, err := pack.Load(".")
		if err != nil {
			return err
		}

		var errs []error
		changed := false
		for _, name := range args {
			r, err := p.Remove(kind, name)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			changed = true
			fmt.Fprintf(cmd.OutOrStdout(), "- %s [%s]\n", r.Mod.Name, r.Kind.Label())
		}

		if changed {
			if err := pack.Save(".", *p); err != nil {
				return err
			}
		}
		return errors.Join(errs...)
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
	removeCmd.Flags().StringVarP(&removeOpts.kind, "type", "t", "all", typeFlagHelp)
}
