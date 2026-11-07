package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Alfakynz/packman/internal/curseforge"
	"github.com/Alfakynz/packman/internal/modrinth"
	"github.com/Alfakynz/packman/internal/pack"
)

var addOpts struct {
	source         string
	kind           string
	modrinthSlug   string
	curseforgeSlug string
	noDeps         bool
}

var addCmd = &cobra.Command{
	Use:   "add <name>...",
	Short: "Adds mods, resource packs or shaders (from Modrinth and CurseForge) to the pack",
	Long: "Adds content by slug, looking it up on Modrinth and CurseForge. The type\n" +
		"(mod, resource pack or shader) is detected automatically; use --type to restrict it.\n" +
		"Required dependencies are added too (use --no-deps to skip them).\n" +
		"CurseForge needs an API key: run `packman auth curseforge` (or set " + curseforge.EnvKey + ");\n" +
		"without it, only Modrinth is used.\n\n" +
		"If something has a different slug on each platform, packman tries to match it\n" +
		"through the file itself. When that isn't enough, give the slug explicitly.",
	Example: "  packman add sodium\n" +
		"  packman add sodium lithium iris complementary-reimagined\n" +
		"  packman add faithful-32x --type resourcepack\n" +
		"  packman add sodium --source modrinth --no-deps\n" +
		"  packman add ferrite-core --curseforge-slug ferritecore-fabric",
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		kind, err := parseKind(addOpts.kind)
		if err != nil {
			return err
		}
		p, err := pack.Load(".")
		if err != nil {
			return err
		}
		out, errOut := cmd.OutOrStdout(), cmd.ErrOrStderr()

		if (addOpts.modrinthSlug != "" || addOpts.curseforgeSlug != "") && len(args) != 1 {
			return errors.New("--modrinth-slug and --curseforge-slug need exactly one name")
		}

		var src pack.Sources
		switch addOpts.source {
		case "all", "modrinth", "curseforge":
		default:
			return fmt.Errorf("invalid --source %q (all | modrinth | curseforge)", addOpts.source)
		}
		if addOpts.source != "curseforge" {
			src.Modrinth = modrinth.New("Alfakynz/packman/" + Version)
		}
		if addOpts.source != "modrinth" {
			cf, err := newCurseForgeClient()
			switch {
			case err == nil:
				src.CurseForge = cf
			case addOpts.source == "all":
				fmt.Fprintf(errOut, "! CurseForge skipped: %v\n", err)
			default:
				return err
			}
		}
		opts := pack.Options{Kind: kind, Modrinth: addOpts.modrinthSlug, CurseForge: addOpts.curseforgeSlug}

		var errs []error
		changed := false

		// Adds what the entry requires, if it isn't in the pack yet.
		addDeps := func(deps []pack.Dep) {
			if addOpts.noDeps || len(deps) == 0 {
				return
			}
			added, problems := pack.AddDeps(src, p, deps)
			for _, a := range added {
				changed = true
				fmt.Fprintf(out, "  + %s [%s] (%s), required by %s\n",
					a.Mod.Name, a.Kind.Label(), strings.Join(a.Mod.Platforms(), ", "), a.Required)
			}
			for _, pr := range problems {
				fmt.Fprintf(errOut, "! %s\n", pr)
			}
		}

		for _, arg := range args {
			res, err := pack.Resolve(src, p, arg, opts)
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", arg, err))
				continue
			}
			mod := res.Mod
			for _, w := range res.Warnings {
				fmt.Fprintf(errOut, "! %s: %s\n", mod.Slug, w)
			}

			// Already in the pack (same slug or same project): only fill in
			// the platforms that were missing.
			if existing := p.FindSame(res.Kind, *mod); existing != nil {
				if existing.FillMissing(*mod) {
					changed = true
					fmt.Fprintf(out, "~ %s [%s] (now on: %s)\n", existing.Slug, res.Kind.Label(), strings.Join(existing.Platforms(), ", "))
				} else {
					fmt.Fprintf(out, "= %s [%s] (already present)\n", existing.Slug, res.Kind.Label())
				}
				addDeps(res.Deps)
				continue
			}

			list := p.Items(res.Kind)
			*list = append(*list, *mod)
			changed = true
			fmt.Fprintf(out, "+ %s [%s] (%s)\n", mod.Name, res.Kind.Label(), strings.Join(mod.Platforms(), ", "))
			addDeps(res.Deps)
		}

		// Save what worked even if some entries failed.
		if changed {
			if err := pack.Save(".", *p); err != nil {
				return err
			}
		}
		return errors.Join(errs...)
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
	f := addCmd.Flags()
	f.StringVarP(&addOpts.kind, "type", "t", "all", typeFlagHelp)
	f.StringVar(&addOpts.source, "source", "all", "all | modrinth | curseforge")
	f.StringVar(&addOpts.modrinthSlug, "modrinth-slug", "", "slug on Modrinth, if different")
	f.StringVar(&addOpts.curseforgeSlug, "curseforge-slug", "", "slug on CurseForge, if different")
	f.BoolVar(&addOpts.noDeps, "no-deps", false, "don't add required dependencies")
}
