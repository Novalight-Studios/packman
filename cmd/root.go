package cmd

import (
	"github.com/spf13/cobra"
)

// go build -ldflags "-X github.com/Alfakynz/packman/cmd.Version=1.0.0"
var Version = "dev"

var rootCmd = &cobra.Command{
	Use: "packman",
	Short: "A modpack manager for Minecraft",
	Long: "packman creates, manages and exports Minecraft modpacks (CurseForge / Modrinth).",
	SilenceUsage: true,
}

func Execute() error {
	rootCmd.Version = Version
	return rootCmd.Execute()
}
