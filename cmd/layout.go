package cmd

import "github.com/spf13/cobra"

var layoutCmd = &cobra.Command{
	Use:   "layout",
	Short: "Get the device UI layout tree",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		args := []string{"layout"}
		if layoutDevice != "" {
			args = append(args, "--device", layoutDevice)
		}
		for _, flag := range []struct {
			name    string
			enabled bool
		}{{"--full", layoutFull}, {"--flat", layoutFlat}, {"--no-idle", layoutNoIdle}, {"--pretty", layoutPretty}} {
			if flag.enabled {
				args = append(args, flag.name)
			}
		}
		if layoutOutput != "" {
			args = append(args, "--output", layoutOutput)
		}
		return runToolCommand(cmd, "android", args...)
	},
}

var layoutDevice string
var layoutFull, layoutFlat, layoutNoIdle, layoutPretty bool
var layoutOutput string

func init() {
	layoutCmd.Flags().StringVarP(&layoutDevice, "device", "s", "", "Device serial number")
	layoutCmd.Flags().BoolVar(&layoutFull, "full", false, "Include non-interactive and hidden elements")
	layoutCmd.Flags().BoolVar(&layoutFlat, "flat", false, "Return a flat list instead of a tree")
	layoutCmd.Flags().BoolVar(&layoutNoIdle, "no-idle", false, "Don't wait for layout idle state")
	layoutCmd.Flags().BoolVar(&layoutPretty, "pretty", false, "Pretty-print JSON")
	layoutCmd.Flags().StringVarP(&layoutOutput, "output", "o", "", "Write layout to file or directory (default stdout)")
	rootCmd.AddCommand(layoutCmd)
}
