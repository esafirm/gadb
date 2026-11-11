package cmd

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
)

// runToolCommand streams output, including binary screenshots, without shell parsing.
func runToolCommand(cmd *cobra.Command, name string, args ...string) error {
	path, err := exec.LookPath(name)
	if err != nil {
		if name == "android" {
			return fmt.Errorf("Android CLI is required for this command: install it from https://developer.android.com/cli: %w", err)
		}
		return err
	}
	child := exec.Command(path, args...)
	child.Stdin = cmd.InOrStdin()
	child.Stdout = cmd.OutOrStdout()
	child.Stderr = cmd.ErrOrStderr()
	return child.Run()
}
