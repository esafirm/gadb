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
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	adb "github.com/esafirm/gadb/adb"
	"github.com/esafirm/gadb/config"
	"github.com/esafirm/gadb/utils"
	color "github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	cleanupSerial string
	cleanupDryRun bool
	cleanupYes    bool
	cleanupSdcard bool
	cleanupApps   bool
	cleanupKeep   []string
)

// cleanupCmd represents the cleanup command
var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Free disk space on an emulator (emulator-only, verified first)",
	Long: `Clean up disposable files on a running Android emulator to free disk space.

Only devices that are verified to be emulators are cleaned — physical devices
are never touched.

Cleaned by default:
  Perfetto traces   /data/misc/perfetto-traces
  Crash tombstones  /data/tombstones
  System debris     package_cache, ANR traces, dropbox, textclassifier,
                    resource cache (all regenerated on demand)
  Temp files        /data/local/tmp (LLDB/Studio tooling and avdslim state kept)
  App caches        pm trim-caches

Optional:
  --sdcard          also empty Download, Pictures and DCIM
  --apps            interactively pick third-party apps to uninstall

Root is used when available (su 0 first, otherwise adb root, restored
afterwards). Without root, Perfetto traces, tombstones and system debris
are skipped.`,
	Run: func(cmd *cobra.Command, args []string) {
		runCleanup()
	},
}

func runCleanup() {
	serial, check := resolveCleanupEmulator()
	color.Green("✓ %s is an emulator", serial)
	if check.Model != "" && check.Model != serial {
		fmt.Printf("  %s\n", color.HiBlackString("model: "+check.Model))
	}
	if len(check.Signals) > 0 {
		fmt.Printf("  %s\n", color.HiBlackString("verified via "+strings.Join(check.Signals, ", ")))
	}

	rootMode, startedAdbd := adb.DetectRootMode(serial)
	canRoot := rootMode != adb.RootUnavailable
	switch {
	case startedAdbd:
		defer adb.RestoreAdbd(serial)
		fmt.Printf("  %s\n", color.HiBlackString("adbd restarted as root (restored on exit)"))
	case rootMode == adb.RootViaAdbd:
		fmt.Printf("  %s\n", color.HiBlackString("using existing root shell"))
	case rootMode == adb.RootUnavailable:
		color.Yellow("! No root access: Perfetto traces and tombstones will be skipped")
	}

	totalKB, _, beforeAvailKB, spaceErr := adb.DataPartitionSpace(serial)
	if spaceErr == nil {
		fmt.Printf("\n/data free: %s of %s\n", adb.FormatKB(beforeAvailKB), adb.FormatKB(totalKB))
	}

	perfettoKB, tombstoneKB := int64(-1), int64(-1)
	debrisKB := int64(0)
	if canRoot {
		if kb, err := adb.PathSizeKB(serial, rootMode, adb.PerfettoTracesPath); err == nil {
			perfettoKB = kb
		}
		if kb, err := adb.PathSizeKB(serial, rootMode, adb.TombstonesPath); err == nil {
			tombstoneKB = kb
		}
		for _, target := range adb.SystemDebrisTargets {
			if kb, err := adb.PathSizeKB(serial, rootMode, target); err == nil {
				debrisKB += kb
			}
		}
	}

	tempEntries, tempErr := adb.ListTempEntries(serial, rootMode)
	if tempErr != nil {
		color.Yellow("! Cannot list %s: %v", adb.TempDirPath, tempErr)
	}
	tempCount, tempKB := 0, int64(0)
	var keptNames []string
	for _, entry := range tempEntries {
		if entry.Kept {
			keptNames = append(keptNames, entry.Name)
			continue
		}
		tempCount++
		if entry.SizeKB > 0 {
			tempKB += entry.SizeKB
		}
	}

	sdcardKB := int64(0)
	if cleanupSdcard {
		for _, target := range adb.SharedStorageCleanupTargets {
			if kb, err := adb.PathSizeKB(serial, adb.RootUnavailable, target); err == nil {
				sdcardKB += kb
			}
		}
	}

	var selectedApps []adb.PackageInfo
	appCandidates := 0
	if cleanupApps {
		selectedApps, appCandidates = pickCleanupApps(serial, rootMode, cleanupDryRun)
	}

	printCleanupPlan(cleanupPlan{
		serial:        serial,
		canRoot:       canRoot,
		perfettoKB:    perfettoKB,
		tombstoneKB:   tombstoneKB,
		debrisKB:      debrisKB,
		tempErr:       tempErr,
		tempCount:     tempCount,
		tempKB:        tempKB,
		keptNames:     keptNames,
		sdcardKB:      sdcardKB,
		appsDryRun:    cleanupDryRun,
		appCandidates: appCandidates,
		selectedApps:  selectedApps,
	})

	if cleanupDryRun {
		color.Yellow("\nDry run — nothing was deleted.")
		return
	}

	if !cleanupYes {
		fmt.Println()
		if !utils.ShowYesOrNoConfirmation("Proceed with cleanup?") {
			color.Yellow("Aborted — nothing was deleted.")
			return
		}
	}

	fmt.Println()
	color.Cyan("Cleaning %s…", serial)

	failures := 0
	stepFailed := func(label string, res adb.CommandReturn) {
		failures++
		color.Red("  ✗ %s", label)
		printCommandFailure(res)
	}

	if canRoot {
		if res := adb.ClearDirContents(serial, rootMode, adb.PerfettoTracesPath); res.Error != nil {
			stepFailed("Perfetto traces", res)
		} else {
			color.Green("  ✓ Perfetto traces")
		}
		if res := adb.ClearDirContents(serial, rootMode, adb.TombstonesPath); res.Error != nil {
			stepFailed("Crash tombstones", res)
		} else {
			color.Green("  ✓ Crash tombstones")
		}

		var debrisFailures []string
		for _, target := range adb.SystemDebrisTargets {
			if res := adb.ClearDirContents(serial, rootMode, target); res.Error != nil {
				debrisFailures = append(debrisFailures, target)
			}
		}
		if len(debrisFailures) > 0 {
			stepFailed(fmt.Sprintf("System debris (failed: %s)", strings.Join(debrisFailures, ", ")), adb.CommandReturn{})
		} else {
			color.Green("  ✓ System debris")
		}
	}

	if tempErr == nil {
		if res := adb.ClearTempDir(serial, rootMode, tempEntries); res.Error != nil {
			stepFailed("Temp files", res)
		} else {
			color.Green("  ✓ Temp files (%d removed)", tempCount)
		}
	}

	if res := adb.ShellCommand(serial, 120*time.Second, "pm trim-caches 999999999999"); res.Error != nil {
		stepFailed("App caches", res)
	} else {
		color.Green("  ✓ App caches trimmed")
	}

	if cleanupSdcard {
		var sdcardFailures []string
		for _, target := range adb.SharedStorageCleanupTargets {
			if res := adb.ClearDirContents(serial, adb.RootUnavailable, target); res.Error != nil {
				sdcardFailures = append(sdcardFailures, target)
			}
		}
		if len(sdcardFailures) > 0 {
			failures++
			color.Red("  ✗ Shared storage (failed: %s)", strings.Join(sdcardFailures, ", "))
		} else {
			color.Green("  ✓ Shared storage")
		}
	}

	for _, app := range selectedApps {
		res := adb.UninstallFrom(serial, app.Name)
		if res.Error != nil {
			failures++
			color.Red("  ✗ Uninstall %s", app.Name)
			printCommandFailure(res)
			continue
		}
		color.Green("  ✓ Uninstalled %s (%s)", app.Name, adb.FormatKB(app.SizeKB))
	}

	fmt.Println()
	if failures > 0 {
		color.Yellow("Finished with %d failed step(s).", failures)
	} else {
		color.Green("Cleanup finished.")
	}

	if spaceErr == nil {
		if _, _, afterAvailKB, err := adb.DataPartitionSpace(serial); err == nil {
			delta := afterAvailKB - beforeAvailKB
			sign := "+"
			if delta < 0 {
				sign = "-"
				delta = -delta
			}
			fmt.Printf("/data free: %s → %s (%s%s)\n",
				adb.FormatKB(beforeAvailKB), adb.FormatKB(afterAvailKB), sign, adb.FormatKB(delta))
		}
	}
}

type cleanupPlan struct {
	serial        string
	canRoot       bool
	perfettoKB    int64
	tombstoneKB   int64
	debrisKB      int64
	tempErr       error
	tempCount     int
	tempKB        int64
	keptNames     []string
	sdcardKB      int64
	appsDryRun    bool
	appCandidates int
	selectedApps  []adb.PackageInfo
}

func printCleanupPlan(plan cleanupPlan) {
	fmt.Println()
	color.Cyan("Cleanup plan for %s", plan.serial)

	item := func(label, value string) {
		fmt.Printf("  • %-30s %s\n", label, value)
	}

	if plan.canRoot {
		item("Perfetto traces", adb.FormatKB(plan.perfettoKB))
		item("Crash tombstones", adb.FormatKB(plan.tombstoneKB))
		item("System debris", adb.FormatKB(plan.debrisKB))
		fmt.Printf("      %s\n", color.HiBlackString(strings.Join(adb.SystemDebrisNames(), ", ")))
	} else {
		item("Perfetto traces", color.YellowString("skipped (no root)"))
		item("Crash tombstones", color.YellowString("skipped (no root)"))
		item("System debris", color.YellowString("skipped (no root)"))
	}

	if plan.tempErr == nil {
		item("Temp files", fmt.Sprintf("%s (%d files)", adb.FormatKB(plan.tempKB), plan.tempCount))
		if len(plan.keptNames) > 0 {
			fmt.Printf("      %s\n", color.HiBlackString("keeping "+strings.Join(plan.keptNames, ", ")))
		}
	}

	item("App caches", color.HiBlackString("pm trim-caches"))

	if cleanupSdcard {
		item("Shared storage", fmt.Sprintf("%s (Download, Pictures, DCIM)", adb.FormatKB(plan.sdcardKB)))
	}

	if cleanupApps {
		switch {
		case plan.appsDryRun:
			item("Uninstall apps", color.HiBlackString(fmt.Sprintf("%d candidates (no selection in dry run)", plan.appCandidates)))
		case len(plan.selectedApps) > 0:
			item("Uninstall apps", fmt.Sprintf("%d selected", len(plan.selectedApps)))
			for _, app := range plan.selectedApps {
				fmt.Printf("      - %s (%s)\n", app.Name, adb.FormatKB(app.SizeKB))
			}
		default:
			item("Uninstall apps", color.HiBlackString("none selected"))
		}
	}
}

func pickCleanupApps(serial string, rootMode adb.RootMode, dryRun bool) ([]adb.PackageInfo, int) {
	pkgs, err := adb.ListThirdPartyPackagesWithSize(serial, rootMode)
	if err != nil {
		color.Yellow("! Cannot list installed packages: %v", err)
		return nil, 0
	}

	configuredPackage := ""
	if cfg, cfgErr := config.ReadConfig(); cfgErr == nil {
		configuredPackage = cfg.PackageName
	}

	var candidates, protected []adb.PackageInfo
	for _, pkg := range pkgs {
		if adb.IsProtectedPackage(pkg.Name, configuredPackage, cleanupKeep) {
			protected = append(protected, pkg)
			continue
		}
		candidates = append(candidates, pkg)
	}

	sort.Slice(candidates, func(i, j int) bool { return candidates[i].SizeKB > candidates[j].SizeKB })
	sort.Slice(protected, func(i, j int) bool { return protected[i].Name < protected[j].Name })

	if len(protected) > 0 {
		names := make([]string, 0, len(protected))
		for _, pkg := range protected {
			names = append(names, pkg.Name)
		}
		fmt.Println(color.HiBlackString("Keeping (protected): " + strings.Join(names, ", ")))
	}

	if len(candidates) == 0 {
		color.Yellow("No removable third-party apps found.")
		return nil, 0
	}

	if dryRun {
		fmt.Println("Third-party apps available to uninstall:")
		for _, pkg := range candidates {
			fmt.Printf("  - %s (%s)\n", pkg.Name, adb.FormatKB(pkg.SizeKB))
		}
		return nil, len(candidates)
	}

	var selected []adb.PackageInfo
	for len(candidates) > 0 {
		items := make([]string, 0, len(candidates)+1)
		items = append(items, "✓ Done selecting")
		for _, pkg := range candidates {
			items = append(items, fmt.Sprintf("%s (%s)", pkg.Name, adb.FormatKB(pkg.SizeKB)))
		}

		index, _ := utils.ShowSelection("Select an app to uninstall", items)
		if index == 0 {
			break
		}

		selected = append(selected, candidates[index-1])
		candidates = append(candidates[:index-1], candidates[index:]...)
	}
	return selected, len(selected)
}

func resolveCleanupEmulator() (string, adb.EmulatorCheck) {
	if cleanupSerial != "" {
		check, err := adb.VerifyEmulator(cleanupSerial)
		if err != nil {
			color.Red("Cannot verify %s: %v", cleanupSerial, err)
			os.Exit(1)
		}
		requireEmulator(check)
		return cleanupSerial, check
	}

	color.Cyan("Looking for connected emulators…")
	devices, err := adb.GetDeviceList()
	if err != nil {
		color.Red("Cannot list devices: %v", err)
		os.Exit(1)
	}

	var emulators []adb.EmulatorCheck
	var physical []string
	for _, device := range devices {
		if device.State != "device" {
			continue
		}
		check, verifyErr := adb.VerifyEmulator(device.Serial)
		if verifyErr != nil {
			continue
		}
		if check.Confirmed {
			emulators = append(emulators, check)
		} else {
			physical = append(physical, device.Serial)
		}
	}

	if len(physical) > 0 {
		color.Yellow("Skipping non-emulator device(s): %s", strings.Join(physical, ", "))
	}

	switch len(emulators) {
	case 0:
		color.Red("No emulator found. Nothing to clean.")
		os.Exit(1)
	case 1:
		return emulators[0].Serial, emulators[0]
	}

	options := make([]string, 0, len(emulators))
	for _, e := range emulators {
		options = append(options, fmt.Sprintf("%s (%s)", e.Serial, e.Model))
	}
	index, _ := utils.ShowSelection("Select emulator to clean", options)
	return emulators[index].Serial, emulators[index]
}

func requireEmulator(check adb.EmulatorCheck) {
	if check.Confirmed {
		return
	}
	color.Red("Refusing to clean %s — not verified as an emulator.", check.Serial)
	color.Yellow("Physical devices are never cleaned by this command.")
	if len(check.Signals) > 0 {
		color.Yellow("The only evidence found was: %s", strings.Join(check.Signals, ", "))
	}
	os.Exit(1)
}

func printCommandFailure(res adb.CommandReturn) {
	output := strings.TrimSpace(string(res.Output))
	if output == "" {
		if res.Error != nil {
			fmt.Printf("      %s\n", res.Error)
		}
		return
	}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			fmt.Printf("      %s\n", line)
		}
	}
}

func init() {
	rootCmd.AddCommand(cleanupCmd)
	cleanupCmd.Flags().StringVarP(&cleanupSerial, "serial", "s", "", "Target emulator serial (default: auto-detect or prompt)")
	cleanupCmd.Flags().BoolVarP(&cleanupDryRun, "dry-run", "n", false, "Show what would be freed without deleting anything")
	cleanupCmd.Flags().BoolVarP(&cleanupYes, "yes", "y", false, "Skip the confirmation prompt")
	cleanupCmd.Flags().BoolVar(&cleanupSdcard, "sdcard", false, "Also empty Download, Pictures and DCIM on shared storage")
	cleanupCmd.Flags().BoolVar(&cleanupApps, "apps", false, "Interactively select third-party apps to uninstall")
	cleanupCmd.Flags().StringSliceVar(&cleanupKeep, "keep", nil, "Never uninstall packages matching this prefix (repeatable)")
}
