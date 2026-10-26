package cmd

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/esafirm/gadb/avd"
	"github.com/esafirm/gadb/utils"
	"github.com/fatih/color"
	pui "github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var (
	killAll bool
	killYes bool
)

var avdsKillCmd = &cobra.Command{
	Use:           "kill [avd|serial]",
	Short:         "Stop running emulators, showing how each one was started",
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	Long: `Show every running emulator with its launch config, then stop one.

The config comes from the live qemu process, so it shows what the emulator was
actually started with: no-window, GPU mode, audio, memory, snapshot, and so on.

Without an argument, pick an emulator from the list. Use --all to stop every
running emulator. Offline emulators that adb cannot reach are killed through
their host process.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if killAll && len(args) > 0 {
			return errors.New("--all cannot be combined with an AVD name or serial")
		}

		running, err := avd.RunningEmulators()
		if err != nil {
			return err
		}
		if len(running) == 0 {
			color.Yellow("No emulators are running.")
			return nil
		}

		printRunningEmulators(running)

		targets := running
		switch {
		case len(args) > 0:
			target, err := findRunningEmulator(running, args[0])
			if err != nil {
				return err
			}
			targets = []avd.RunningEmulator{target}
		case !killAll:
			target, err := pickRunningEmulator(running)
			if err != nil {
				return err
			}
			targets = []avd.RunningEmulator{target}
		case !killYes:
			if !utils.ShowYesOrNoConfirmation(fmt.Sprintf("Stop all %d running emulators?", len(targets))) {
				color.Yellow("Aborted — nothing was stopped.")
				return nil
			}
		}

		fmt.Println()
		failures := 0
		for _, target := range targets {
			if err := target.Stop(30 * time.Second); err != nil {
				failures++
				color.Red("  ✗ %s: %v", target.Label(), err)
				continue
			}
			color.Green("  ✓ stopped %s", target.Label())
		}
		if failures > 0 {
			return fmt.Errorf("%d emulator(s) failed to stop", failures)
		}
		return nil
	},
}

func printRunningEmulators(running []avd.RunningEmulator) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "SERIAL\tAVD\tSTATE\tPID\tCONFIG")
	for _, emulator := range running {
		serial := emulator.Serial
		if serial == "" {
			serial = "-"
		}
		pid := "-"
		if emulator.PID > 0 {
			pid = strconv.Itoa(emulator.PID)
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n",
			serial, emulator.AVD, emulator.State, pid, emulator.Config.Summary())
	}
	_ = writer.Flush()
}

func findRunningEmulator(running []avd.RunningEmulator, query string) (avd.RunningEmulator, error) {
	for _, candidate := range running {
		if strings.EqualFold(candidate.Serial, query) || strings.EqualFold(candidate.AVD, query) {
			return candidate, nil
		}
	}
	return avd.RunningEmulator{}, fmt.Errorf("no running emulator matches %q", query)
}

func pickRunningEmulator(running []avd.RunningEmulator) (avd.RunningEmulator, error) {
	items := make([]string, len(running))
	for i, candidate := range running {
		items[i] = candidate.Label()
	}

	prompt := pui.Select{
		Label: "Select emulator to stop",
		Items: items,
		Size:  12,
	}
	index, _, err := prompt.Run()
	if err != nil {
		return avd.RunningEmulator{}, err
	}
	return running[index], nil
}

func init() {
	avdsCmd.AddCommand(avdsKillCmd)
	avdsKillCmd.Flags().BoolVar(&killAll, "all", false, "Stop every running emulator")
	avdsKillCmd.Flags().BoolVarP(&killYes, "yes", "y", false, "Do not ask before stopping all emulators")
}
