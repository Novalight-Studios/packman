package cmd

// Hides the auto-generated `completion` command from the help.
func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}
