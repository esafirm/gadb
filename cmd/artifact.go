package cmd

import (
	"os/exec"

	"github.com/spf13/cobra"
)

var artifactCmd = &cobra.Command{
	Use:                "artifact [arguments]",
	Short:              "Analyze Android APK, AAB, JAR, and AAR files",
	Long:               "Forwards arguments to `android analyze` for rich artifact analysis.\n\n" + "Usage: android analyze [-h] [--analysis=PARAM] [--format=PARAM] <path> [options]",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if isBackendHelp(args) {
			if _, err := exec.LookPath("android"); err != nil {
				return cmd.Help()
			}
		}
		return runToolCommand(cmd, "android", append([]string{"analyze"}, args...)...)
	},
}

func init() { rootCmd.AddCommand(artifactCmd) }

func isBackendHelp(args []string) bool {
	return len(args) == 1 && (args[0] == "--help" || args[0] == "-h")
}
