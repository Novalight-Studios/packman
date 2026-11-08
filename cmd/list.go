package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Novalight-Studios/packman/internal/export"
	"github.com/Novalight-Studios/packman/internal/pack"
)

var listOpts struct {
	platform string
	kind     string
	output   string
}

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "Lists the content of the pack as Markdown, with a link for each project",
	Long: "Generates a Markdown list of every mod, resource pack and shader of the pack,\n" +
		"sorted by name, with a link to each project.\n\n" +
		"--platform chooses which pages the links point to. An entry that isn't on that\n" +
		"platform links to the other one. CurseForge page URLs need your API key; without\n" +
		"it, links use the project ID (www.curseforge.com/projects/<id>), which redirects.\n\n" +
		"The list is printed, or written to a file with --output.",
	Example: "  packman list\n" +
		"  packman list --platform curseforge\n" +
		"  packman list --type mod -o MODS.md\n" +
		"  packman list -o PACK_CONTENT.md",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		kind, err := parseKind(listOpts.kind)
		if err != nil {
			return err
		}
		if listOpts.platform != "modrinth" && listOpts.platform != "curseforge" {
			return fmt.Errorf("invalid --platform %q (modrinth | curseforge)", listOpts.platform)
		}
		p, err := pack.Load(".")
		if err != nil {
			return err
		}
		out, errOut := cmd.OutOrStdout(), cmd.ErrOrStderr()

		opts := export.ListOptions{Platform: listOpts.platform}
		if kind != "" {
			opts.Kinds = []pack.Kind{kind}
		}
		if listOpts.platform == "curseforge" {
			if cf, err := newCurseForgeClient(); err == nil {
				opts.CurseForge = cf
			} else {
				fmt.Fprintln(errOut, "! No CurseForge API key: links use project IDs instead of page names (see `packman auth curseforge`)")
			}
		}

		rep, err := export.ContentList(p, opts)
		if err != nil {
			return err
		}
		for _, w := range rep.Warnings {
			fmt.Fprintf(errOut, "! %s\n", w)
		}
		if n := len(rep.Fallbacks); n > 0 {
			other := "Modrinth"
			if listOpts.platform == "modrinth" {
				other = "CurseForge"
			}
			names, more := rep.Fallbacks, ""
			if n > 10 {
				names, more = names[:10], ", …"
			}
			fmt.Fprintf(errOut, "! %d entry(ies) not on %s link to %s instead: %s%s\n",
				n, listOpts.platform, other, strings.Join(names, ", "), more)
		}

		if rep.Markdown == "" {
			fmt.Fprintln(errOut, "Nothing to list")
			return nil
		}
		if listOpts.output == "" {
			fmt.Fprint(out, rep.Markdown)
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(listOpts.output), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(listOpts.output, []byte(rep.Markdown), 0o644); err != nil {
			return err
		}
		fmt.Fprintf(out, "Wrote %s (%d entries)\n", listOpts.output, rep.Count)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	f := listCmd.Flags()
	f.StringVarP(&listOpts.platform, "platform", "p", "modrinth", "which pages to link to: modrinth | curseforge")
	f.StringVarP(&listOpts.kind, "type", "t", "all", typeFlagHelp)
	f.StringVarP(&listOpts.output, "output", "o", "", "write the list to this file instead of printing it")
}
