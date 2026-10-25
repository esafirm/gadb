// Copyright © 2019 Esa Firman <esafirm21@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/chzyer/readline"
	adb "github.com/esafirm/gadb/adb"
	"github.com/esafirm/gadb/avd"
	"github.com/esafirm/gadb/config"
	"github.com/esafirm/gadb/slim"
	"github.com/esafirm/gadb/utils"
	"github.com/fatih/color"
	pui "github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var (
	slimRAM        int
	slimGPU        string
	slimKeepAudio  bool
	slimKeepCamera bool
	slimRestore    bool

	createAPI        int
	createDevice     string
	createRAM        int
	createGPU        string
	createKeepAudio  bool
	createKeepCamera bool

	onAggressive bool
	onKeep       []string
	onSkip       []string
	onNoAnim     bool

	startRAM        int
	startGPU        string
	startHeadless   bool
	startCold       bool
	startNoLowRAM   bool
	startNoSlim     bool
	startKeepAudio  bool
	startKeepCamera bool
	startAggressive bool
	startKeep       []string
	startSkip       []string
	startNoAnim     bool

	snapshotLive       bool
	snapshotHeadless   bool
	snapshotYes        bool
	snapshotRAM        int
	snapshotGPU        string
	snapshotKeepAudio  bool
	snapshotKeepCamera bool
	snapshotAggressive bool
	snapshotKeep       []string
	snapshotSkip       []string
	snapshotNoAnim     bool

	unbakeYes bool
)

var slimCmd = &cobra.Command{
	Use:   "slim",
	Short: "Keep Android emulators light, fast and reproducible",
	Long: `Keep Android emulators light, fast and reproducible.

check     audit the image, config and live guest (read-only)
create    build a new AVD from a 4 KB Google APIs image
tune      set guest RAM, host GPU, audio and camera in config.ini
start     tune, boot, wait and prune in one command
on        prune packages and settings on a running emulator
off       restore packages, settings and config.ini
status    show what is applied and whether the guest drifted
snapshot  bake a clean gadb_clean snapshot for fast boots
unbake    delete that snapshot

Run with no subcommand for an interactive menu. Use the subcommands and flags
for scripting.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !readline.IsTerminal(int(os.Stdin.Fd())) {
			return cmd.Help()
		}
		return runSlimMenu()
	},
}

var slimCheckCmd = &cobra.Command{
	Use:           "check [avd]",
	Short:         "Audit an AVD without changing anything",
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			avds, err := avd.List()
			if err != nil {
				return err
			}
			switch len(avds) {
			case 0:
				return errors.New("no AVDs found")
			case 1:
				// fall through to the detailed report for the only AVD
			default:
				return printCheckTable(avds)
			}
		}

		target, err := selectSlimAVD(args, "Select an AVD to check")
		if err != nil {
			return err
		}
		report, err := avd.Check(target)
		if err != nil {
			return err
		}
		printCheckReport(report)
		return nil
	},
}

var slimTuneCmd = &cobra.Command{
	Use:           "tune [avd]",
	Short:         "Tune an AVD's host configuration",
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		target, err := selectSlimAVD(args, "Select an AVD to tune")
		if err != nil {
			return err
		}

		if slimRestore {
			result, err := avd.RestoreConfig(target)
			if err != nil {
				return err
			}
			printRestoreResult(result)
			return nil
		}

		result, err := avd.Tune(target, avd.TuneOptions{
			RAM:        slimRAM,
			GPU:        slimGPU,
			KeepAudio:  slimKeepAudio,
			KeepCamera: slimKeepCamera,
		})
		if err != nil {
			return err
		}
		printTuneResult(result)
		return nil
	},
}

var slimCreateCmd = &cobra.Command{
	Use:           "create <name>",
	Short:         "Create a slim-ready AVD from a 4 KB Google APIs image",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := avd.Create(avd.CreateOptions{
			Name:       args[0],
			API:        createAPI,
			Device:     createDevice,
			RAM:        createRAM,
			GPU:        createGPU,
			KeepAudio:  createKeepAudio,
			KeepCamera: createKeepCamera,
		})
		if err != nil {
			return err
		}
		printCreateResult(result)
		return nil
	},
}

var slimOnCmd = &cobra.Command{
	Use:           "on [serial|avd]",
	Short:         "Prune packages and tune settings on a running emulator",
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateSlimSkip(onSkip); err != nil {
			return err
		}

		target, serial, running, err := resolveSlimTarget(args, true, "Select a running emulator to slim")
		if err != nil {
			return err
		}
		if !running {
			return fmt.Errorf("AVD %q is not running", target.Name)
		}

		dev, err := slim.NewADBDevice(serial)
		if err != nil {
			return err
		}

		result, err := slim.On(target, dev, slim.Options{
			Aggressive: onAggressive,
			Keep:       slimKeepPackages(onKeep),
			Skip:       onSkip,
			NoAnim:     onNoAnim,
		})
		if err != nil {
			return err
		}
		printOnResult(result)
		return nil
	},
}

var slimOffCmd = &cobra.Command{
	Use:           "off [serial|avd]",
	Short:         "Restore packages, settings and config.ini",
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		target, serial, running, err := resolveSlimTarget(args, false, "Select an AVD to restore")
		if err != nil {
			return err
		}

		var dev slim.Device
		if running {
			adbDevice, err := slim.NewADBDevice(serial)
			if err != nil {
				return err
			}
			dev = adbDevice
		}

		result, err := slim.Off(target, dev, running)
		if err != nil {
			return err
		}
		printOffResult(result)
		return nil
	},
}

var slimStatusCmd = &cobra.Command{
	Use:           "status [serial|avd]",
	Short:         "Show what is applied and whether the guest still matches",
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		target, serial, running, err := resolveSlimTarget(args, false, "Select an AVD to inspect")
		if err != nil {
			return err
		}

		var dev slim.Device
		if running {
			adbDevice, err := slim.NewADBDevice(serial)
			if err != nil {
				return err
			}
			dev = adbDevice
		}

		status, err := slim.Status(target, dev)
		if err != nil {
			return err
		}
		printSlimStatus(status)
		return nil
	},
}

var slimStartCmd = &cobra.Command{
	Use:           "start [avd]",
	Short:         "Tune, boot and slim an AVD",
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateSlimSkip(startSkip); err != nil {
			return err
		}

		target, err := selectSlimAVD(args, "Select an AVD to start")
		if err != nil {
			return err
		}
		if serial, running := avd.RunningSerial(target.Name); running {
			return fmt.Errorf("AVD %q is already running (%s); use gadb slim on %s",
				target.Name, serial, target.Name)
		}
		tune, ram, gpu := slimTuneState(cmd, target, startRAM, startGPU)
		return runSlimStart(target, tune, ram, gpu)
	},
}

var slimSnapshotCmd = &cobra.Command{
	Use:           "snapshot [avd]",
	Aliases:       []string{"bake"},
	Short:         "Bake a clean golden snapshot for fast slim boots",
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateSlimSkip(snapshotSkip); err != nil {
			return err
		}

		target, err := selectSlimAVD(args, "Select an AVD to snapshot")
		if err != nil {
			return err
		}
		if snapshotLive {
			for _, flag := range []string{"headless", "ram", "gpu", "keep-audio", "keep-camera"} {
				if cmd.Flags().Changed(flag) {
					return fmt.Errorf("--%s cannot be combined with --live", flag)
				}
			}
			return runLiveSnapshot(target)
		}
		tune, ram, gpu := slimTuneState(cmd, target, snapshotRAM, snapshotGPU)
		return runColdSnapshot(target, tune, ram, gpu)
	},
}

var slimUnbakeCmd = &cobra.Command{
	Use:           "unbake [avd]",
	Short:         "Delete the golden snapshot",
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		target, err := selectSlimAVD(args, "Select an AVD to unbake")
		if err != nil {
			return err
		}

		if serial, running := avd.RunningSerial(target.Name); running {
			if !unbakeYes && !utils.ShowYesOrNoConfirmation(
				fmt.Sprintf("Stop %s (%s) to delete its snapshot?", target.Name, serial)) {
				return fmt.Errorf("AVD %q is running; stop it first", target.Name)
			}
			if err := avd.StopSerial(serial, 30*time.Second); err != nil {
				return err
			}
		}

		existed := target.HasSnapshot(avd.CleanSnapshotName)
		if err := avd.DeleteSnapshot(target, avd.CleanSnapshotName); err != nil {
			return err
		}
		manifest, err := avd.LoadManifest(target)
		if err != nil {
			return err
		}
		if manifest != nil {
			manifest.Snapshot = nil
			if manifest.Empty() {
				if err := avd.RemoveManifest(target); err != nil {
					return err
				}
			} else if err := avd.SaveManifest(target, manifest); err != nil {
				return err
			}
		}

		fmt.Printf("%s %s\n", color.New(color.Bold).Sprint("AVD"), color.New(color.Bold).Sprint(target.Name))
		if existed {
			fmt.Printf("  %s snapshot removed\n", color.GreenString("✓"))
		} else {
			fmt.Printf("  %s no snapshot to remove\n", color.HiBlackString("-"))
		}
		return nil
	},
}

func runSlimStart(target avd.AVD, tune bool, ram int, gpu string) error {
	var image *avd.Image
	if tune {
		result, err := avd.Tune(target, avd.TuneOptions{
			RAM:        ram,
			GPU:        gpu,
			KeepAudio:  startKeepAudio,
			KeepCamera: startKeepCamera,
		})
		if err != nil {
			return err
		}
		image = result.Image
	} else {
		image = imageForAVD(target)
		target.ClearHardwareFile()
	}

	fmt.Printf("%s %s\n", color.New(color.Bold).Sprint("AVD"), color.New(color.Bold).Sprint(target.Name))
	if image != nil {
		fmt.Printf("  %-10s %s (%d KB pages)\n", "image", imageTitle(image), image.PageSizeKB)
	}
	warnUnsuitableImage(image)

	manifest, err := avd.LoadManifest(target)
	if err != nil {
		return err
	}
	snapshot := avd.CheckSnapshot(target, manifest)
	useSnapshot := snapshot.Valid && !startCold
	switch {
	case startCold && snapshot.Valid:
		fmt.Printf("  %s cold boot requested, ignoring the golden snapshot\n", color.HiBlackString("-"))
	case snapshot.Exists && !snapshot.Valid:
		fmt.Printf("  %s snapshot is stale (%s), cold booting\n", color.YellowString("!"), snapshot.Reason)
	case !snapshot.Exists:
		fmt.Printf("  %s no golden snapshot, cold booting (run gadb slim snapshot %s for fast boots)\n",
			color.HiBlackString("-"), target.Name)
	}

	snapshotName := ""
	if useSnapshot {
		snapshotName = avd.CleanSnapshotName
		if startNoLowRAM {
			fmt.Printf("  %s the golden snapshot was baked with -lowram\n", color.YellowString("!"))
		}
	}
	process, err := avd.Launch(target, avd.LaunchOptions{
		RAM:        ram,
		GPU:        gpu,
		Headless:   startHeadless,
		LowRAM:     !startNoLowRAM,
		NoAudio:    !startKeepAudio,
		NoBootAnim: true,
		Snapshot:   snapshotName,
	})
	if err != nil {
		return err
	}

	fmt.Printf("  %-10s %s\n", "serial", process.Serial)
	fmt.Printf("  %-10s %s\n", "log", process.LogPath)
	started := time.Now()
	if err := process.WaitForBoot(3 * time.Minute); err != nil {
		_ = process.Stop(30 * time.Second)
		return err
	}
	boot := "cold boot"
	if useSnapshot {
		boot = "golden snapshot"
	}
	fmt.Printf("  %s booted from %s in %s\n",
		color.GreenString("✓"), boot, time.Since(started).Round(time.Second))

	if !startNoSlim {
		device, err := slim.NewADBDevice(process.Serial)
		if err != nil {
			return err
		}
		result, err := slim.On(target, device, slim.Options{
			Aggressive: startAggressive,
			Keep:       slimKeepPackages(startKeep),
			Skip:       startSkip,
			NoAnim:     startNoAnim,
		})
		if err != nil {
			return err
		}
		printOnChanges(result)
		fmt.Printf("  %-10s gadb slim off %s\n", "restore", target.Name)
	}

	return nil
}

func runColdSnapshot(target avd.AVD, tune bool, ram int, gpu string) error {
	if serial, running := avd.RunningSerial(target.Name); running {
		if !snapshotYes && !utils.ShowYesOrNoConfirmation(
			fmt.Sprintf("Stop %s (%s) for a clean bake?", target.Name, serial)) {
			return fmt.Errorf("AVD %q is running; use --live or stop it first", target.Name)
		}
		if err := avd.StopSerial(serial, 30*time.Second); err != nil {
			return err
		}
	}

	var image *avd.Image
	if tune {
		result, err := avd.Tune(target, avd.TuneOptions{
			RAM:        ram,
			GPU:        gpu,
			KeepAudio:  snapshotKeepAudio,
			KeepCamera: snapshotKeepCamera,
		})
		if err != nil {
			return err
		}
		image = result.Image
	} else {
		image = imageForAVD(target)
		target.ClearHardwareFile()
	}

	fmt.Printf("%s %s\n", color.New(color.Bold).Sprint("AVD"), color.New(color.Bold).Sprint(target.Name))
	if image != nil {
		fmt.Printf("  %-10s %s (%d KB pages)\n", "image", imageTitle(image), image.PageSizeKB)
	}
	warnUnsuitableImage(image)

	process, err := avd.Launch(target, avd.LaunchOptions{
		RAM:        ram,
		GPU:        gpu,
		Headless:   snapshotHeadless,
		LowRAM:     true,
		NoAudio:    !snapshotKeepAudio,
		NoBootAnim: true,
	})
	if err != nil {
		return err
	}
	defer func() {
		if err := process.Stop(30 * time.Second); err != nil {
			fmt.Printf("  %s %s\n", color.YellowString("!"), err)
		}
	}()

	fmt.Printf("  %-10s %s\n", "serial", process.Serial)
	fmt.Printf("  %-10s %s\n", "log", process.LogPath)
	started := time.Now()
	if err := process.WaitForBoot(3 * time.Minute); err != nil {
		return err
	}
	fmt.Printf("  %s booted in %s\n", color.GreenString("✓"), time.Since(started).Round(time.Second))

	device, err := slim.NewADBDevice(process.Serial)
	if err != nil {
		return err
	}
	result, err := slim.On(target, device, slim.Options{
		Aggressive: snapshotAggressive,
		Keep:       slimKeepPackages(snapshotKeep),
		Skip:       snapshotSkip,
		NoAnim:     snapshotNoAnim,
	})
	if err != nil {
		return err
	}
	printOnChanges(result)

	time.Sleep(2 * time.Second)
	if err := saveCleanSnapshot(target, process.Serial); err != nil {
		return err
	}
	fmt.Printf("  %-10s gadb slim start %s\n", "next", target.Name)
	return nil
}

func runLiveSnapshot(target avd.AVD) error {
	serial, running := avd.RunningSerial(target.Name)
	if !running {
		return fmt.Errorf("AVD %q is not running; start it first or drop --live", target.Name)
	}

	device, err := slim.NewADBDevice(serial)
	if err != nil {
		return err
	}

	manifest, err := avd.LoadManifest(target)
	if err != nil {
		return err
	}
	if manifest == nil || len(manifest.Config) == 0 {
		fmt.Printf("  %s this AVD is not tuned, start will tune it and may mark this snapshot stale\n",
			color.YellowString("!"))
	}

	fmt.Printf("%s %s (%s)\n", color.New(color.Bold).Sprint("AVD"),
		color.New(color.Bold).Sprint(target.Name), serial)
	fmt.Printf("  %s the snapshot will include installed apps and current data\n", color.YellowString("!"))
	result, err := slim.On(target, device, slim.Options{
		Aggressive: snapshotAggressive,
		Keep:       slimKeepPackages(snapshotKeep),
		Skip:       snapshotSkip,
		NoAnim:     snapshotNoAnim,
	})
	if err != nil {
		return err
	}
	printOnChanges(result)

	time.Sleep(2 * time.Second)
	if err := saveCleanSnapshot(target, serial); err != nil {
		return err
	}
	fmt.Printf("  %-10s gadb slim start %s\n", "next", target.Name)
	return nil
}

func saveCleanSnapshot(target avd.AVD, serial string) error {
	fmt.Printf("  %s saving snapshot %s\n", color.HiBlackString("-"), avd.CleanSnapshotName)
	result := adb.RunOnlyWithTimeout(2*time.Minute, "adb",
		"-s", serial, "emu", "avd", "snapshot", "save", avd.CleanSnapshotName)
	output := strings.TrimSpace(string(result.Output))
	if result.Error != nil || strings.Contains(output, "KO") {
		clearSnapshotRef(target)
		return fmt.Errorf("snapshot save failed: %v %s", result.Error, output)
	}
	if !target.HasSnapshot(avd.CleanSnapshotName) {
		clearSnapshotRef(target)
		return fmt.Errorf("the emulator reported success but %s was not created",
			target.SnapshotDir(avd.CleanSnapshotName))
	}

	manifest, err := avd.LoadManifest(target)
	if err != nil {
		return err
	}
	if manifest == nil {
		manifest = &avd.Manifest{Version: avd.ManifestVersion, AVD: target.Name}
	}
	config, err := avd.LoadConfig(target.ConfigPath())
	if err != nil {
		return err
	}
	image, _ := avd.ImageForAVD(config)
	manifest.Snapshot = &avd.SnapshotRef{
		Name:               avd.CleanSnapshotName,
		CreatedAt:          time.Now().UTC().Format(time.RFC3339),
		ImageFingerprint:   avd.ImageFingerprint(image),
		ConfigHash:         avd.ConfigHash(config),
		PackageListVersion: slim.PackageListVersion,
	}
	if err := avd.SaveManifest(target, manifest); err != nil {
		return err
	}

	size, _ := avd.SnapshotSize(target, avd.CleanSnapshotName)
	fmt.Printf("  %s snapshot saved (%s)\n", color.GreenString("✓"), avd.HumanSize(size))
	return nil
}

func clearSnapshotRef(target avd.AVD) {
	_ = avd.DeleteSnapshot(target, avd.CleanSnapshotName)
	manifest, err := avd.LoadManifest(target)
	if err != nil || manifest == nil {
		return
	}
	manifest.Snapshot = nil
	if manifest.Empty() {
		_ = avd.RemoveManifest(target)
	} else {
		_ = avd.SaveManifest(target, manifest)
	}
}

func validateSlimSkip(groups []string) error {
	for _, group := range groups {
		if !slim.ValidSkip(group) {
			return fmt.Errorf("unknown --skip group %q (valid: %s)",
				group, strings.Join(slim.SkipGroups(), ", "))
		}
	}
	return nil
}

func slimKeepPackages(extra []string) []string {
	keep := append([]string{}, extra...)
	if pkg := config.GetPackageNameOrDefault(func() string { return "" }); pkg != "" {
		keep = append(keep, pkg)
	}
	return keep
}

func warnUnsuitableImage(image *avd.Image) {
	if image == nil {
		return
	}
	if image.PageSizeKB == 16 {
		fmt.Printf("  %s 16 KB image: the guest will still reserve 4096 MB\n", color.YellowString("!"))
	}
	if image.IsPlayStore() {
		fmt.Printf("  %s Play Store image: adb root is unavailable\n", color.YellowString("!"))
	}
}

// slimTuneState keeps an existing tune and its launch values intact unless the
// user asked for a change. Re-running Tune with the flag defaults would reset
// the RAM and invalidate a matching golden snapshot.
func slimTuneState(cmd *cobra.Command, target avd.AVD, fallbackRAM int, fallbackGPU string) (bool, int, string) {
	ram, gpu := fallbackRAM, fallbackGPU

	manifest, err := avd.LoadManifest(target)
	tuned := err == nil && manifest != nil && len(manifest.Config) > 0
	if tuned {
		if config, err := avd.LoadConfig(target.ConfigPath()); err == nil {
			if !cmd.Flags().Changed("ram") {
				if value, ok := config.Get("hw.ramSize"); ok {
					if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
						ram = parsed
					}
				}
			}
			if !cmd.Flags().Changed("gpu") {
				if value, ok := config.Get("hw.gpu.mode"); ok && value != "" {
					gpu = value
				}
			}
		}
	}

	tune := !tuned ||
		cmd.Flags().Changed("ram") ||
		cmd.Flags().Changed("gpu") ||
		cmd.Flags().Changed("keep-audio") ||
		cmd.Flags().Changed("keep-camera")

	return tune, ram, gpu
}

func imageForAVD(target avd.AVD) *avd.Image {
	config, err := avd.LoadConfig(target.ConfigPath())
	if err != nil {
		return nil
	}
	image, _ := avd.ImageForAVD(config)
	return image
}

func resolveSlimTarget(args []string, requireRunning bool, caption string) (avd.AVD, string, bool, error) {
	if len(args) > 0 {
		return resolveNamedSlimTarget(args[0], requireRunning)
	}

	running, runningErr := avd.RunningAVDs()
	if runningErr != nil && requireRunning {
		return avd.AVD{}, "", false, runningErr
	}
	if runningErr == nil && len(running) > 0 {
		if len(running) == 1 {
			target, err := avd.Find(running[0].AVD)
			return target, running[0].Serial, true, err
		}

		labels := make([]string, len(running))
		for i, candidate := range running {
			labels[i] = fmt.Sprintf("%s (%s)", candidate.AVD, candidate.Serial)
		}
		index, _, err := promptSlimSelection(caption, labels)
		if err != nil {
			return avd.AVD{}, "", false, err
		}
		chosen := running[index]
		target, err := avd.Find(chosen.AVD)
		return target, chosen.Serial, true, err
	}
	if requireRunning {
		return avd.AVD{}, "", false, errors.New("no running emulators")
	}

	installed, err := avd.List()
	if err != nil {
		return avd.AVD{}, "", false, err
	}
	switch len(installed) {
	case 0:
		return avd.AVD{}, "", false, errors.New("no AVDs found")
	case 1:
		return installed[0], "", false, nil
	default:
		names := make([]string, len(installed))
		for i, candidate := range installed {
			names[i] = candidate.Name
		}
		index, _, err := promptSlimSelection(caption, names)
		if err != nil {
			return avd.AVD{}, "", false, err
		}
		return installed[index], "", false, nil
	}
}

func resolveNamedSlimTarget(target string, requireRunning bool) (avd.AVD, string, bool, error) {
	if strings.HasPrefix(target, "emulator-") {
		name, err := avd.NameForSerial(target)
		if err != nil {
			return avd.AVD{}, "", false, err
		}
		found, err := avd.Find(name)
		return found, target, true, err
	}

	found, err := avd.Find(target)
	if err != nil {
		return avd.AVD{}, "", false, err
	}
	serial, running := avd.RunningSerial(found.Name)
	if requireRunning && !running {
		return avd.AVD{}, "", false, fmt.Errorf("AVD %q is not running", found.Name)
	}
	return found, serial, running, nil
}

func selectSlimAVD(args []string, caption string) (avd.AVD, error) {
	if len(args) > 0 {
		return avd.Find(args[0])
	}

	avds, err := avd.List()
	if err != nil {
		return avd.AVD{}, err
	}
	switch len(avds) {
	case 0:
		return avd.AVD{}, errors.New("no AVDs found")
	case 1:
		return avds[0], nil
	default:
		names := make([]string, len(avds))
		for i, candidate := range avds {
			names[i] = candidate.Name
		}
		index, _, err := promptSlimSelection(caption, names)
		if err != nil {
			return avd.AVD{}, err
		}
		return avds[index], nil
	}
}

func promptSlimSelection(caption string, items []string) (int, string, error) {
	prompt := pui.Select{
		Label: caption,
		Items: items,
		Size:  12,
	}
	return prompt.Run()
}

func printCheckTable(avds []avd.AVD) error {
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "NAME\tIMAGE\tRAM\tGPU\tVERDICT")

	for _, candidate := range avds {
		report, err := avd.Check(candidate)
		if err != nil {
			fmt.Fprintf(writer, "%s\t-\t-\t-\terror: %v\n", candidate.Name, err)
			continue
		}

		ram := report.RAM
		if report.Runtime != nil && report.Runtime.MemTotalMB > 0 {
			ram = fmt.Sprintf("%s/%d", report.RAM, report.Runtime.MemTotalMB)
		}

		gpu := report.GPU
		if report.Runtime != nil && report.Runtime.Renderer != "" && report.Runtime.Renderer != "unknown" {
			gpu = fmt.Sprintf("%s/%s", report.GPU, shortRenderer(report.Runtime.Renderer))
		}

		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n",
			candidate.Name, shortImage(report.Image), ram, gpu, strings.ToUpper(report.Verdict))
	}

	return writer.Flush()
}

func printCheckReport(report *avd.CheckReport) {
	fmt.Printf("%s %s\n", color.New(color.Bold).Sprint("AVD"), color.New(color.Bold).Sprint(report.AVD.Name))
	fmt.Printf("  %-10s %s\n", "path", report.AVD.Dir)

	if report.Image != nil {
		fmt.Printf("  %-10s %s\n", "image", imageTitle(report.Image))
		if report.Image.TagDisplay != "" {
			fmt.Printf("  %-10s %s\n", "", report.Image.TagDisplay)
		}
		if !report.Image.Complete {
			fmt.Printf("  %-10s %s\n", "", color.RedString("incomplete image"))
		}
	} else if report.ImageErr != nil {
		fmt.Printf("  %-10s %s\n", "image", color.YellowString(report.ImageErr.Error()))
	}

	ram := report.RAM + " MB"
	if report.RAMFloor > 0 {
		ram += fmt.Sprintf(" (image floor %d MB)", report.RAMFloor)
	}
	fmt.Printf("  %-10s RAM %s, GPU %s, audio %s, camera %s\n",
		"config", ram, report.GPU, report.Audio, report.Camera)

	if report.Runtime != nil {
		lowRAM := "not low-RAM"
		if report.Runtime.LowRAM {
			lowRAM = "low-RAM"
		}
		fmt.Printf("  %-10s %s: %d MB RAM, %s, %s\n",
			"runtime", report.Runtime.Serial, report.Runtime.MemTotalMB, report.Runtime.Renderer, lowRAM)
	} else {
		fmt.Printf("  %-10s %s\n", "runtime", "not running")
	}

	snapshot := "none"
	if report.Snapshot != nil && report.Snapshot.Exists {
		snapshot = report.Snapshot.Name
		if report.Snapshot.Valid {
			snapshot += " (valid)"
		} else {
			snapshot += " (stale: " + report.Snapshot.Reason + ")"
		}
	}
	fmt.Printf("  %-10s %s\n", "snapshot", snapshot)

	manifest := "none"
	switch {
	case report.ManifestErr != nil:
		manifest = "unreadable"
	case report.Manifest != nil:
		manifest = "tuned"
	}
	fmt.Printf("  %-10s %s\n", "manifest", manifest)
	fmt.Printf("  %-10s %s\n", "verdict", verdictLabel(report.Verdict))

	if len(report.Notes) > 0 {
		fmt.Println("  notes")
		for _, note := range report.Notes {
			fmt.Printf("    - %s\n", note)
		}
	}
	if len(report.Next) > 0 {
		fmt.Println("  next")
		for _, next := range report.Next {
			fmt.Printf("    - %s\n", next)
		}
	}
}

func printTuneResult(result *avd.TuneResult) {
	fmt.Printf("%s %s\n", color.New(color.Bold).Sprint("AVD"), color.New(color.Bold).Sprint(result.AVD.Name))
	if result.Image != nil {
		fmt.Printf("  %-10s %s (%d KB pages)\n", "image", imageTitle(result.Image), result.Image.PageSizeKB)
	}
	printTuneChanges(result)
}

func printTuneChanges(result *avd.TuneResult) {
	for _, change := range result.Changes {
		from := change.From
		if !change.Existed {
			from = "(unset)"
		}
		unchanged := ""
		if from == change.To {
			unchanged = color.HiBlackString(" (unchanged)")
		}
		fmt.Printf("  %s %-18s %s -> %s%s\n",
			color.GreenString("✓"), change.Key, from, change.To, unchanged)
	}
	for _, skipped := range result.Skipped {
		fmt.Printf("  %s %s\n", color.YellowString("!"), skipped)
	}

	if result.RemovedHardware {
		fmt.Printf("  %-10s %s\n", "removed", "hardware-qemu.ini (regenerated on boot)")
	}
	fmt.Printf("  %-10s %s\n", "backup", result.Backup)
	for _, warning := range result.Warnings {
		fmt.Printf("  %s %s\n", color.YellowString("!"), warning)
	}
	fmt.Printf("  %-10s gadb slim tune %s --restore\n", "restore", result.AVD.Name)
}

func printCreateResult(result *avd.CreateResult) {
	fmt.Printf("%s %s\n", color.GreenString("✓ created"), color.New(color.Bold).Sprint(result.AVD.Name))
	fmt.Printf("  %-10s %s\n", "image", result.Image.PkgID)
	fmt.Printf("  %-10s %s\n", "device", result.Device)
	for _, warning := range result.Warnings {
		fmt.Printf("  %s %s\n", color.YellowString("!"), warning)
	}
	if result.Tune != nil {
		printTuneChanges(result.Tune)
	}
	fmt.Printf("  %-10s gadb avds %s\n", "next", result.AVD.Name)
}

func printOnResult(result *slim.OnResult) {
	fmt.Printf("%s %s (%s)\n",
		color.New(color.Bold).Sprint("AVD"),
		color.New(color.Bold).Sprint(result.AVD.Name),
		result.Serial)
	printOnChanges(result)
	fmt.Printf("  %-10s gadb slim off %s\n", "restore", result.AVD.Name)
}

func printOnChanges(result *slim.OnResult) {
	if len(result.Disabled) == 0 {
		fmt.Printf("  %s no packages to disable\n", color.HiBlackString("-"))
	} else {
		fmt.Printf("  %s disabled %d packages\n", color.GreenString("✓"), len(result.Disabled))
		for _, pkg := range result.Disabled {
			fmt.Printf("    - %s\n", pkg)
		}
	}
	if len(result.Kept) > 0 {
		fmt.Printf("  %s kept %s\n", color.HiBlackString("-"), strings.Join(result.Kept, ", "))
	}
	for _, setting := range result.Settings {
		from := "(unset)"
		if setting.Existed && setting.Original != nil {
			from = *setting.Original
		}
		fmt.Printf("  %s %-32s %s -> %s\n",
			color.GreenString("✓"), setting.Key, from, setting.Tuned)
	}
	for _, failure := range result.FailedPackages {
		fmt.Printf("  %s package: %s\n", color.RedString("✗"), failure)
	}
	for _, failure := range result.FailedSettings {
		fmt.Printf("  %s setting: %s\n", color.RedString("✗"), failure)
	}
	for _, warning := range result.Warnings {
		fmt.Printf("  %s %s\n", color.YellowString("!"), warning)
	}
	if result.Trimmed {
		fmt.Printf("  %s trimmed cached processes\n", color.GreenString("✓"))
	}
}

func printOffResult(result *slim.OffResult) {
	header := result.AVD.Name
	if result.Serial != "" {
		header += " (" + result.Serial + ")"
	}
	fmt.Printf("%s %s\n", color.New(color.Bold).Sprint("AVD"), color.New(color.Bold).Sprint(header))

	if result.Running {
		fmt.Printf("  %s re-enabled %d packages\n", color.GreenString("✓"), len(result.RestoredPackages))
		fmt.Printf("  %s restored %d settings\n", color.GreenString("✓"), len(result.RestoredSettings))
	}
	if result.Config != nil && len(result.Config.Restored) > 0 {
		fmt.Printf("  %s restored %d config.ini values\n", color.GreenString("✓"), len(result.Config.Restored))
	}
	if result.Config != nil {
		for _, skipped := range result.Config.Skipped {
			fmt.Printf("  %s config: %s\n", color.YellowString("!"), skipped)
		}
	}
	for _, failure := range result.FailedPackages {
		fmt.Printf("  %s package: %s\n", color.RedString("✗"), failure)
	}
	for _, failure := range result.FailedSettings {
		fmt.Printf("  %s setting: %s\n", color.RedString("✗"), failure)
	}
	for _, warning := range result.Warnings {
		fmt.Printf("  %s %s\n", color.YellowString("!"), warning)
	}
	if result.RemovedManifest {
		fmt.Printf("  %s manifest removed\n", color.GreenString("✓"))
	} else if len(result.FailedPackages) > 0 || len(result.FailedSettings) > 0 {
		fmt.Printf("  %-10s gadb slim off %s\n", "retry", result.AVD.Name)
	} else if !result.Running {
		fmt.Printf("  %-10s start the emulator, then gadb slim off %s\n", "next", result.AVD.Name)
	}
}

func printSlimStatus(status *slim.StatusReport) {
	header := status.AVD.Name
	if status.Serial != "" {
		header += " (" + status.Serial + ")"
	}
	fmt.Printf("%s %s\n", color.New(color.Bold).Sprint("AVD"), color.New(color.Bold).Sprint(header))

	if status.Manifest == nil {
		fmt.Printf("  %s not slimmed by gadb\n", color.HiBlackString("-"))
		if status.Running {
			fmt.Printf("  %-10s gadb slim on %s\n", "next", status.AVD.Name)
		} else {
			fmt.Printf("  %-10s start the emulator, then gadb slim on %s\n", "next", status.AVD.Name)
		}
		return
	}

	applied := status.Manifest.AppliedAt
	if applied == "" {
		applied = "unknown"
	}
	fmt.Printf("  %-10s %s\n", "applied", applied)

	if status.Running {
		fmt.Printf("  %-10s %d recorded, %d disabled\n", "packages", status.Packages, status.Disabled)
		fmt.Printf("  %-10s %d recorded, %d match\n", "settings", status.Settings, status.SettingsOK)
	} else {
		fmt.Printf("  %-10s %d recorded (guest not running)\n", "packages", status.Packages)
		fmt.Printf("  %-10s %d recorded (guest not running)\n", "settings", status.Settings)
	}

	if len(status.ConfigDrift) == 0 {
		fmt.Printf("  %-10s all tuned values match\n", "config")
	} else {
		fmt.Printf("  %-10s drift: %s\n", "config", strings.Join(status.ConfigDrift, ", "))
	}
	if status.Snapshot == "" {
		fmt.Printf("  %-10s none\n", "snapshot")
	} else {
		line := status.Snapshot
		if status.SnapshotValid {
			line += " (valid)"
		} else {
			line += " (stale: " + status.SnapshotReason + ")"
		}
		fmt.Printf("  %-10s %s\n", "snapshot", line)
	}

	for _, pkg := range status.ReEnabled {
		fmt.Printf("  %s drift: %s is enabled again\n", color.YellowString("!"), pkg)
	}
	for _, pkg := range status.Missing {
		fmt.Printf("  %s missing: %s is no longer installed\n", color.YellowString("!"), pkg)
	}
	for _, setting := range status.SettingsDrift {
		fmt.Printf("  %s drift: %s does not match the manifest\n", color.YellowString("!"), setting)
	}
	if status.OlderList {
		fmt.Printf("  %s the package list changed since this manifest, run gadb slim on %s\n",
			color.YellowString("!"), status.AVD.Name)
	}
}

func printRestoreResult(result *avd.RestoreResult) {
	fmt.Printf("%s %s\n", color.New(color.Bold).Sprint("AVD"), color.New(color.Bold).Sprint(result.AVD.Name))

	for _, change := range result.Restored {
		to := change.To
		if to == "" {
			to = "(removed)"
		}
		fmt.Printf("  %s %-18s %s -> %s\n",
			color.GreenString("✓"), change.Key, change.From, to)
	}
	for _, skipped := range result.Skipped {
		fmt.Printf("  %s %s\n", color.YellowString("!"), skipped)
	}
	if len(result.Restored) == 0 && len(result.Skipped) == 0 {
		fmt.Println("  no recorded config changes")
	}

	if result.RemovedManifest {
		fmt.Printf("  %-10s removed\n", "manifest")
	}
	fmt.Printf("  %-10s %s\n", "backup", filepath.Join(result.AVD.Dir, avd.BackupName))
}

func shortImage(image *avd.Image) string {
	if image == nil {
		return "unknown"
	}
	return image.ShortLabel()
}

func shortRenderer(renderer string) string {
	switch renderer {
	case "software":
		return "sw"
	case "hardware":
		return "hw"
	default:
		return "?"
	}
}

func imageTitle(image *avd.Image) string {
	return fmt.Sprintf("%s %s %s", image.API, image.TagID, image.ABI)
}

func verdictLabel(verdict string) string {
	switch verdict {
	case "high":
		return color.RedString("high impact")
	case "medium":
		return color.YellowString("medium impact")
	case "low":
		return color.GreenString("low impact")
	default:
		return verdict
	}
}

func init() {
	rootCmd.AddCommand(slimCmd)
	slimCmd.AddCommand(slimCheckCmd, slimCreateCmd, slimOffCmd, slimOnCmd, slimSnapshotCmd,
		slimStartCmd, slimStatusCmd, slimTuneCmd, slimUnbakeCmd)

	slimTuneCmd.Flags().IntVar(&slimRAM, "ram", avd.DefaultRAM, "Guest RAM in MB")
	slimTuneCmd.Flags().StringVar(&slimGPU, "gpu", avd.DefaultGPU(), "Emulator GPU mode")
	slimTuneCmd.Flags().BoolVar(&slimKeepAudio, "keep-audio", false, "Leave audio emulation on")
	slimTuneCmd.Flags().BoolVar(&slimKeepCamera, "keep-camera", false, "Leave camera emulation on")
	slimTuneCmd.Flags().BoolVar(&slimRestore, "restore", false, "Restore the values recorded by the last tune")

	slimCreateCmd.Flags().IntVar(&createAPI, "api", 0, "API level to use (default: newest installed)")
	slimCreateCmd.Flags().StringVar(&createDevice, "device", "pixel", "avdmanager device profile")
	slimCreateCmd.Flags().IntVar(&createRAM, "ram", avd.DefaultRAM, "Guest RAM in MB")
	slimCreateCmd.Flags().StringVar(&createGPU, "gpu", avd.DefaultGPU(), "Emulator GPU mode")
	slimCreateCmd.Flags().BoolVar(&createKeepAudio, "keep-audio", false, "Leave audio emulation on")
	slimCreateCmd.Flags().BoolVar(&createKeepCamera, "keep-camera", false, "Leave camera emulation on")

	slimOnCmd.Flags().BoolVar(&onAggressive, "aggressive", false, "Also disable Play Store, Chrome and setup packages")
	slimOnCmd.Flags().StringArrayVar(&onKeep, "keep", nil, "Never disable this package (repeatable)")
	slimOnCmd.Flags().StringArrayVar(&onSkip, "skip", nil, "Leave a group alone (repeatable)")
	slimOnCmd.Flags().BoolVar(&onNoAnim, "no-anim", false, "Set the animation scales to 0")

	slimStartCmd.Flags().IntVar(&startRAM, "ram", avd.DefaultRAM, "Guest RAM in MB")
	slimStartCmd.Flags().StringVar(&startGPU, "gpu", avd.DefaultGPU(), "Emulator GPU mode")
	slimStartCmd.Flags().BoolVar(&startHeadless, "headless", false, "Run without an emulator window")
	slimStartCmd.Flags().BoolVar(&startCold, "cold", false, "Ignore the golden snapshot and cold boot")
	slimStartCmd.Flags().BoolVar(&startNoLowRAM, "no-lowram", false, "Do not pass -lowram")
	slimStartCmd.Flags().BoolVar(&startNoSlim, "no-slim", false, "Skip guest pruning after boot")
	slimStartCmd.Flags().BoolVar(&startKeepAudio, "keep-audio", false, "Leave audio emulation on")
	slimStartCmd.Flags().BoolVar(&startKeepCamera, "keep-camera", false, "Leave camera emulation on")
	slimStartCmd.Flags().BoolVar(&startAggressive, "aggressive", false, "Also disable Play Store, Chrome and setup packages")
	slimStartCmd.Flags().StringArrayVar(&startKeep, "keep", nil, "Never disable this package (repeatable)")
	slimStartCmd.Flags().StringArrayVar(&startSkip, "skip", nil, "Leave a group alone (repeatable)")
	slimStartCmd.Flags().BoolVar(&startNoAnim, "no-anim", false, "Set the animation scales to 0")

	slimSnapshotCmd.Flags().BoolVar(&snapshotLive, "live", false, "Snapshot the running emulator instead of a cold bake")
	slimSnapshotCmd.Flags().BoolVar(&snapshotHeadless, "headless", false, "Bake without an emulator window")
	slimSnapshotCmd.Flags().BoolVar(&snapshotYes, "yes", false, "Stop a running emulator without asking")
	slimSnapshotCmd.Flags().IntVar(&snapshotRAM, "ram", avd.DefaultRAM, "Guest RAM in MB")
	slimSnapshotCmd.Flags().StringVar(&snapshotGPU, "gpu", avd.DefaultGPU(), "Emulator GPU mode")
	slimSnapshotCmd.Flags().BoolVar(&snapshotKeepAudio, "keep-audio", false, "Leave audio emulation on")
	slimSnapshotCmd.Flags().BoolVar(&snapshotKeepCamera, "keep-camera", false, "Leave camera emulation on")
	slimSnapshotCmd.Flags().BoolVar(&snapshotAggressive, "aggressive", false, "Also disable Play Store, Chrome and setup packages")
	slimSnapshotCmd.Flags().StringArrayVar(&snapshotKeep, "keep", nil, "Never disable this package (repeatable)")
	slimSnapshotCmd.Flags().StringArrayVar(&snapshotSkip, "skip", nil, "Leave a group alone (repeatable)")
	slimSnapshotCmd.Flags().BoolVar(&snapshotNoAnim, "no-anim", false, "Set the animation scales to 0")

	slimUnbakeCmd.Flags().BoolVar(&unbakeYes, "yes", false, "Stop a running emulator without asking")
}
