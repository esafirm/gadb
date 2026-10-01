// Copyright © 2026 Esa Firman esafirm21@gmail.com
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
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	adb "github.com/esafirm/gadb/adb"
	config "github.com/esafirm/gadb/config"
	"github.com/esafirm/gadb/logtui"
	"github.com/esafirm/gadb/utils"
	color "github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	logPackage     string
	logSerial      string
	logLevel       string
	logAllPackages bool
	logFilterQuery string
	logExclude     string
)

var logCmd = &cobra.Command{
	Use:     "log [package] [filter]",
	Aliases: []string{"logs", "logcat"},
	Short:   "Interactive TUI for streaming and analyzing device logcat",
	Long: `Interactive keystroke-driven terminal UI for viewing and searching
Android device logcat in real time. Features include live filtering, level filtering,
crash & ANR detection, crash stack trace popup with AI prompt export, and rustycat-style
column layout.`,
	Run: func(cmd *cobra.Command, args []string) {
		runLogTUI(args)
	},
}

var logsCmd = &cobra.Command{
	Use:     "logs [package] [filter]",
	Aliases: []string{"logcat"},
	Short:   "Interactive TUI for streaming and analyzing device logcat",
	Long: `Interactive keystroke-driven terminal UI for viewing and searching
Android device logcat in real time. Features include live filtering, level filtering,
crash & ANR detection, crash stack trace popup with AI prompt export, and rustycat-style
column layout.`,
	Run: func(cmd *cobra.Command, args []string) {
		runLogTUI(args)
	},
}

func init() {
	rootCmd.AddCommand(logCmd)
	rootCmd.AddCommand(logsCmd)

	setupLogFlags(logCmd)
	setupLogFlags(logsCmd)
}

func setupLogFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&logPackage, "package", "k", "", "Filter logs by application ID / package name")
	cmd.Flags().StringVarP(&logSerial, "serial", "s", "", "Target specific device serial")
	cmd.Flags().StringVarP(&logLevel, "level", "l", "", "Minimum log level (V, D, I, W, E)")
	cmd.Flags().BoolVarP(&logAllPackages, "all", "a", false, "Start showing all logs (no package filter)")
	cmd.Flags().StringVarP(&logFilterQuery, "filter", "f", "", "Initial search / include filter")
	cmd.Flags().StringVarP(&logExclude, "exclude", "x", "", "Initial exclude filter")
}

// parseLogArgs resolves package name, filter query, and allPackages mode from args and flags
func parseLogArgs(args []string, flagPkg, flagFilter string, flagAll bool) (pkg string, filter string, all bool) {
	pkg = flagPkg
	filter = flagFilter
	all = flagAll

	if flagAll {
		// User explicitly asked for all packages; any positional args are filter terms
		if filter == "" && len(args) > 0 {
			filter = strings.Join(args, " ")
		}
		return "", filter, true
	}

	if pkg != "" {
		// Package was specified via -k/--package; any positional args are filter terms
		if filter == "" && len(args) > 0 {
			filter = strings.Join(args, " ")
		}
		return pkg, filter, false
	}

	// No package flag and not --all: first arg is package name (if provided)
	if len(args) > 0 {
		pkg = args[0]
		if filter == "" && len(args) > 1 {
			filter = strings.Join(args[1:], " ")
		}
		if pkg == "" {
			all = true
		} else {
			all = false
		}
		return pkg, filter, all
	}

	// No args, no -k, no -a: check configuration
	configPkg := config.GetPackageNameOrDefault(func() string {
		return ""
	})
	if configPkg != "" {
		pkg = configPkg
		all = false
	} else {
		all = true
	}

	return pkg, filter, all
}

// RunLog runs the interactive log TUI with positional arguments
func RunLog(args ...string) {
	runLogTUI(args)
}

func runLogTUI(args []string) {
	// 1. Resolve connected device
	serial := logSerial
	if serial == "" {
		devices, err := adb.GetDeviceList()
		if err != nil {
			color.Red("Failed to list devices: %v", err)
			os.Exit(1)
		}

		if len(devices) == 0 {
			color.Red("No device connected. Please connect a device or start an emulator.")
			os.Exit(1)
		} else if len(devices) == 1 {
			serial = devices[0].Serial
		} else {
			// Prompt device selection
			var options []string
			for _, d := range devices {
				label := fmt.Sprintf("%s (%s)", d.Serial, d.Model)
				options = append(options, label)
			}
			idx, _ := utils.ShowSelection("Select target device", options)
			serial = devices[idx].Serial
		}
	}

	// 2. Resolve package name, filter query, and allPackages mode
	pkg, filter, allPackages := parseLogArgs(args, logPackage, logFilterQuery, logAllPackages)

	// 3. Create Model and launch Bubble Tea program
	cfg := logtui.Config{
		DeviceSerial: serial,
		PackageName:  pkg,
		Level:        logLevel,
		AllPackages:  allPackages,
		FilterQuery:  filter,
		ExcludeQuery: logExclude,
	}

	model := logtui.NewModel(cfg)
	program := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running log TUI: %v\n", err)
		os.Exit(1)
	}
}
