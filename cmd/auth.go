package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/Novalight-Studios/packman/internal/config"
	"github.com/Novalight-Studios/packman/internal/curseforge"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manages the API keys used by packman",
}

var authCurseForgeOpts struct {
	remove bool
}

var authCurseForgeCmd = &cobra.Command{
	Use:   "curseforge",
	Short: "Saves (or removes) your CurseForge API key",
	Long: "Saves your CurseForge API key so you don't have to set an environment variable.\n" +
		"Get a key at https://console.curseforge.com\n\n" +
		"The key is typed hidden (never as an argument, so it stays out of your shell history)\n" +
		"and stored in your user config directory, readable by you only.\n" +
		"It can also be piped: echo \"$KEY\" | packman auth curseforge",
	Example: "  packman auth curseforge\n" +
		"  packman auth curseforge --remove",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		out := cmd.OutOrStdout()

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if authCurseForgeOpts.remove {
			cfg.CurseForgeKey = ""
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Fprintln(out, "CurseForge API key removed")
			return nil
		}

		key, err := readSecret(cmd, "CurseForge API key: ")
		if err != nil {
			return err
		}
		if key == "" {
			return errors.New("empty key")
		}
		if err := curseforge.New(key).Check(); err != nil {
			return fmt.Errorf("key not saved: %w", err)
		}

		cfg.CurseForgeKey = key
		if err := config.Save(cfg); err != nil {
			return err
		}
		path, _ := config.Path()
		fmt.Fprintf(out, "CurseForge API key saved to %s\n", path)
		if os.Getenv(curseforge.EnvKey) != "" {
			fmt.Fprintf(cmd.ErrOrStderr(), "! %s is set and takes priority over the saved key\n", curseforge.EnvKey)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authCurseForgeCmd)
	authCurseForgeCmd.Flags().BoolVar(&authCurseForgeOpts.remove, "remove", false, "remove the saved key")
}

// newCurseForgeClient looks for the API key in the environment first,
// then in the saved config.
func newCurseForgeClient() (*curseforge.Client, error) {
	if key := strings.TrimSpace(os.Getenv(curseforge.EnvKey)); key != "" {
		return curseforge.New(key), nil
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	if cfg.CurseForgeKey == "" {
		return nil, errors.New("no CurseForge API key: run `packman auth curseforge` or set " + curseforge.EnvKey)
	}
	return curseforge.New(cfg.CurseForgeKey), nil
}

// readSecret reads a hidden line from the terminal, or a plain line if stdin is piped.
func readSecret(cmd *cobra.Command, prompt string) (string, error) {
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		fmt.Fprint(cmd.ErrOrStderr(), prompt)
		b, err := term.ReadPassword(fd)
		fmt.Fprintln(cmd.ErrOrStderr())
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(b)), nil
	}
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
