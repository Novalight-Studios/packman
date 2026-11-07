package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Alfakynz/packman/internal/curseforge"
	"github.com/Alfakynz/packman/internal/export"
	"github.com/Alfakynz/packman/internal/pack"
)

var exportOpts struct {
	format      string
	output      string
	skipMissing bool
}

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Exports the pack as a Modrinth (.mrpack) and a CurseForge (.zip) modpack",
	Long: "Generates the import files of both platforms from packman.json.\n\n" +
		"Everything that exists on the target platform is referenced there. Anything that\n" +
		"exists only on the other platform is downloaded and bundled in the pack as an\n" +
		"override, along with every file of the overrides/ folder.\n" +
		"Bundling a CurseForge-only file needs your CurseForge API key (see `packman auth`).",
	Example: "  packman export\n" +
		"  packman export --format modrinth\n" +
		"  packman export -o build --skip-missing",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		var formats []string
		switch exportOpts.format {
		case "all":
			formats = []string{"modrinth", "curseforge"}
		case "modrinth", "curseforge":
			formats = []string{exportOpts.format}
		default:
			return fmt.Errorf("invalid --format %q (all | modrinth | curseforge)", exportOpts.format)
		}

		p, err := pack.Load(".")
		if err != nil {
			return err
		}
		out, errOut := cmd.OutOrStdout(), cmd.ErrOrStderr()

		// The CurseForge client is only created if something needs it.
		var cfClient *curseforge.Client
		var cfErr error
		cfLoaded := false
		opts := export.Options{
			Dir:         ".",
			OutDir:      exportOpts.output,
			SkipMissing: exportOpts.skipMissing,
			CurseForge: func() (*curseforge.Client, error) {
				if !cfLoaded {
					cfClient, cfErr = newCurseForgeClient()
					cfLoaded = true
				}
				return cfClient, cfErr
			},
			Progress: func(msg string) { fmt.Fprintln(errOut, "…", msg) },
		}

		var errs []error
		bundledAny := false
		for _, format := range formats {
			var rep *export.Report
			if format == "modrinth" {
				rep, err = export.Modrinth(p, opts)
			} else {
				rep, err = export.CurseForge(p, opts)
			}
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", format, err))
				continue
			}

			fmt.Fprintf(out, "Exported %s\n", rep.Path)
			if len(rep.Bundled) > 0 {
				bundledAny = true
				fmt.Fprintf(out, "  bundled as overrides: %s\n", strings.Join(rep.Bundled, ", "))
			}
			for _, s := range rep.Skipped {
				fmt.Fprintf(errOut, "! skipped %s\n", s)
			}
		}

		if bundledAny {
			fmt.Fprintln(errOut, "Note: bundled files are redistributed inside the pack, make sure their licenses allow it.")
		}
		return errors.Join(errs...)
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)
	f := exportCmd.Flags()
	f.StringVar(&exportOpts.format, "format", "all", "all | modrinth | curseforge")
	f.StringVarP(&exportOpts.output, "output", "o", "dist", "output folder")
	f.BoolVar(&exportOpts.skipMissing, "skip-missing", false, "leave out entries that can't be included instead of failing")
}
