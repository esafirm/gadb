package cmd

import "github.com/spf13/cobra"
import "os/exec"

var runCmd = &cobra.Command{
	Use:                "run [apks_or_reference] [-- build_args]",
	Short:              "Build, install, and launch an Android app",
	Long:               "Forwards arguments to `android run`. Use a build reference or APK path, and `--` to pass build arguments verbatim.\n\n" + "Usage: android run [options] [<apks_or_reference>] [-- <build_args>]",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if isBackendHelp(args) {
			if _, err := exec.LookPath("android"); err != nil {
				return cmd.Help()
			}
		}
		return runToolCommand(cmd, "android", append([]string{"run"}, args...)...)
	},
}

func init() { rootCmd.AddCommand(runCmd) }
