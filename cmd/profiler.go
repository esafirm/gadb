package cmd

import (
	"os/exec"

	"github.com/spf13/cobra"
)

var profilerCmd = &cobra.Command{
	Use:                "profiler [command] [arguments]",
	Short:              "Profile Android apps, games, and devices",
	Long:               "Forwards profiler commands to the Android CLI. Available commands include heapdump, session, and systemtrace.\n\n" + "Usage: android profiler [command] [arguments]",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if isBackendHelp(args) {
			if _, err := exec.LookPath("android"); err != nil {
				return cmd.Help()
			}
		}
		return runToolCommand(cmd, "android", append([]string{"profiler"}, args...)...)
	},
}

func init() { rootCmd.AddCommand(profilerCmd) }
