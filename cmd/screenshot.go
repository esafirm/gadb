package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var screenshotDevice, screenshotOutput string
var screenshotAnnotated bool

var screenshotCmd = &cobra.Command{
	Use:   "screenshot",
	Short: "Capture the device screen as PNG",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if screenshotAnnotated {
			args := []string{"screen", "capture", "--annotate"}
			if screenshotDevice != "" {
				args = append(args, "--device", screenshotDevice)
			}
			dir := ""
			if screenshotOutput != "" {
				dir = filepath.Dir(screenshotOutput)
			}
			tmp, err := os.CreateTemp(dir, ".gadb-screenshot-*.png")
			if err != nil {
				return err
			}
			tmpPath := tmp.Name()
			if err := tmp.Close(); err != nil {
				os.Remove(tmpPath)
				return err
			}
			defer os.Remove(tmpPath)
			args = append(args, "--output", tmpPath)
			stdout := cmd.OutOrStdout()
			cmd.SetOut(cmd.ErrOrStderr())
			runErr := runToolCommand(cmd, "android", args...)
			cmd.SetOut(stdout)
			if runErr != nil {
				return runErr
			}
			data, err := os.ReadFile(tmpPath)
			if err != nil {
				return fmt.Errorf("android screenshot output unavailable: %w", err)
			}
			if !bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")) {
				return fmt.Errorf("android screenshot output is not a PNG")
			}
			if screenshotOutput == "" {
				_, err = cmd.OutOrStdout().Write(data)
				return err
			}
			if err := os.Rename(tmpPath, screenshotOutput); err != nil {
				return fmt.Errorf("write screenshot: %w", err)
			}
			return nil
		}
		args := []string{}
		if screenshotDevice != "" {
			args = append(args, "-s", screenshotDevice)
		}
		args = append(args, "exec-out", "screencap", "-p")
		if screenshotOutput == "" {
			return runToolCommand(cmd, "adb", args...)
		}
		dir := filepath.Dir(screenshotOutput)
		f, err := os.CreateTemp(dir, ".gadb-screenshot-*")
		if err != nil {
			return err
		}
		tmp := f.Name()
		if err := f.Close(); err != nil {
			os.Remove(f.Name())
			return err
		}
		defer os.Remove(tmp)
		child, err := exec.LookPath("adb")
		if err != nil {
			return err
		}
		process := exec.Command(child, args...)
		out, err := os.Create(tmp)
		if err != nil {
			return err
		}
		process.Stdout, process.Stderr, process.Stdin = out, cmd.ErrOrStderr(), cmd.InOrStdin()
		err = process.Run()
		closeErr := out.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		data, err := os.ReadFile(tmp)
		if err != nil {
			return err
		}
		if !bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")) {
			return fmt.Errorf("adb screenshot output is not a PNG")
		}
		if err := os.Rename(tmp, screenshotOutput); err != nil {
			return fmt.Errorf("write screenshot: %w", err)
		}
		return nil
	},
}

func init() {
	screenshotCmd.Flags().StringVarP(&screenshotOutput, "output", "o", "", "Write PNG to this path (default stdout)")
	screenshotCmd.Flags().StringVarP(&screenshotDevice, "device", "s", "", "Device serial number")
	screenshotCmd.Flags().BoolVar(&screenshotAnnotated, "annotate", false, "Capture with numbered UI element annotations (requires Android CLI)")
	rootCmd.AddCommand(screenshotCmd)
}
