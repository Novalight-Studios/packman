package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Novalight-Studios/packman/internal/modrinth"
	"github.com/Novalight-Studios/packman/internal/pack"
)

var updateOpts struct {
	kind   string
	dryRun bool
	noDeps bool
}

var updateCmd = &cobra.Command{
	Use:   "update [name]...",
	Short: "Updates mods, resource packs and shaders to their latest version",
	Long: "Moves entries to the latest version compatible with the Minecraft version and\n" +
		"loader of the pack. Without arguments, everything is checked.\n" +
		"Required dependencies missing from the pack are added too (use --no-deps to skip them).\n\n" +
		"Changed the Minecraft version or the loader in packman.json? Run `packman update`\n" +
		"to bring every entry along; the ones with no compatible version are listed.\n" +
		"Entries are looked up by ID, so a different slug on each platform is not a problem.",
	Example: "  packman update\n" +
		"  packman update sodium lithium\n" +
		"  packman update --type shader\n" +
		"  packman update --dry-run",
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		kind, err := parseKind(updateOpts.kind)
		if err != nil {
			return err
		}
		p, err := pack.Load(".")
		if err != nil {
			return err
		}
		out, errOut := cmd.OutOrStdout(), cmd.ErrOrStderr()

		// Which entries to check.
		type target struct {
			kind pack.Kind
			mod  *pack.Mod
		}
		var targets []target
		var errs []error

		if len(args) == 0 {
			kinds := pack.AllKinds
			if kind != "" {
				kinds = []pack.Kind{kind}
			}
			for _, k := range kinds {
				list := p.Items(k)
				for i := range *list {
					targets = append(targets, target{kind: k, mod: &(*list)[i]})
				}
			}
		} else {
			for _, name := range args {
				k, m, err := p.Find(kind, name)
				if err != nil {
					errs = append(errs, err)
					continue
				}
				targets = append(targets, target{kind: k, mod: m})
			}
		}
		if len(targets) == 0 {
			if len(errs) > 0 {
				return errors.Join(errs...)
			}
			fmt.Fprintln(out, "Nothing to update")
			return nil
		}

		src := pack.Sources{Modrinth: modrinth.New("Novalight-Studios/packman/" + Version)}
		needsCF := false
		for _, t := range targets {
			if t.mod.CurseForge != nil {
				needsCF = true
				break
			}
		}
		if needsCF {
			if cf, err := newCurseForgeClient(); err == nil {
				src.CurseForge = cf
			} else {
				fmt.Fprintf(errOut, "! CurseForge skipped: %v\n", err)
			}
		}

		fmt.Fprintf(errOut, "Checking %d entries…\n", len(targets))
		updated, problemCount := 0, 0
		var allDeps []pack.Dep
		for _, t := range targets {
			changes, problems, deps := pack.UpdateEntry(src, p, t.kind, t.mod)
			allDeps = append(allDeps, deps...)
			if len(changes) > 0 {
				updated++
				fmt.Fprintf(out, "~ %s [%s]\n", t.mod.Name, t.kind.Label())
				for _, c := range changes {
					fmt.Fprintf(out, "    %s\n", c)
				}
			}
			for _, pr := range problems {
				problemCount++
				fmt.Fprintf(errOut, "! %s: %s\n", t.mod.Slug, pr)
			}
		}

		// Dependencies are added once every entry has been checked: adding
		// to the pack's lists while holding pointers into them would be unsafe.
		var added []pack.Added
		if !updateOpts.noDeps {
			var problems []string
			added, problems = pack.AddDeps(src, p, allDeps, false)
			for _, a := range added {
				fmt.Fprintf(out, "+ %s [%s] (%s), required by %s\n",
					a.Mod.Name, a.Kind.Label(), strings.Join(a.Mod.Platforms(), ", "), a.Required)
			}
			for _, pr := range problems {
				problemCount++
				fmt.Fprintf(errOut, "! %s\n", pr)
			}
		}

		total := updated + len(added)
		switch {
		case total == 0 && problemCount == 0:
			fmt.Fprintln(out, "Everything is up to date")
		case total == 0:
			fmt.Fprintf(out, "Nothing updated (%d problem(s) above)\n", problemCount)
		case updateOpts.dryRun:
			fmt.Fprintf(out, "%d update(s) and %d dependency(ies) to add (dry run: nothing written)\n", updated, len(added))
		default:
			if err := pack.Save(".", *p); err != nil {
				return err
			}
			fmt.Fprintf(out, "%d updated, %d dependency(ies) added\n", updated, len(added))
		}
		return errors.Join(errs...)
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
	f := updateCmd.Flags()
	f.StringVarP(&updateOpts.kind, "type", "t", "all", typeFlagHelp)
	f.BoolVar(&updateOpts.dryRun, "dry-run", false, "show what would change without writing anything")
	f.BoolVar(&updateOpts.noDeps, "no-deps", false, "don't add missing required dependencies")
}
