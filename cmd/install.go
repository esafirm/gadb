// Copyright © 2019 NAME HERE <EMAIL ADDRESS>
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
	"time"

	adb "github.com/esafirm/gadb/adb"
	analyzer "github.com/esafirm/gadb/apkanalyzer"
	color "github.com/fatih/color"
	pui "github.com/manifoldco/promptui"

	"github.com/esafirm/gadb/utils"
	"github.com/spf13/cobra"
)

var isYes bool
var isVerbose bool
var isNoTimeout bool

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install [apk_path]",
	Short: `Install APK to single or multiple`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			showHelpAndExit(cmd, "APK path is required")
		} else {
			runCommand(args[0])
		}
	},
}

func showHelpAndExit(cmd *cobra.Command, errorMsg string) {
	if errorMsg != "" {
		fmt.Fprintln(os.Stderr, errorMsg)
	}
	cmd.Help()
	os.Exit(1)
}

func runCommand(apkPath string) {
	color.Cyan("Installing %s…", apkPath)

	// Check if the file exist
	if _, err := os.Stat(apkPath); os.IsNotExist(err) {
		color.Red("File %s not found\n", apkPath)
		os.Exit(1)
	}

	var comamndReturn adb.CommandReturn
	if isNoTimeout {
		comamndReturn = adb.ReInstallWithTimeout(apkPath, -1*time.Second)
	} else {
		comamndReturn = adb.ReInstall(apkPath)
	}
	output := string(comamndReturn.Output)

	if comamndReturn.Error != nil {
		printCommandError(comamndReturn)
		color.Yellow("")

		if canRecoverVersionDowngrade(apkPath, output) {
			runCommand(apkPath)
			return
		}
		if shouldShowDevicePicker(output) {
			showDevicePicker(apkPath)
			return
		}

		color.Red("Cannot install %s", apkPath)
		os.Exit(1)
	}
}

// printCommandError shows adb's failure line, or the underlying error when adb
// gave no failure detail (timeout, killed process, connection problem).
func printCommandError(commandReturn adb.CommandReturn) {
	output := string(commandReturn.Output)

	if isVerbose {
		color.Red("Error: %v", commandReturn.Error)
		color.Yellow("Output: %s", output)
		return
	}

	if message := extractErrorMessage(output); message != "" {
		color.Yellow("%s", message)
		return
	}

	color.Yellow("%v", commandReturn.Error)
	if trimmed := strings.TrimSpace(output); trimmed != "" {
		color.Yellow("%s", trimmed)
	}
}

func showDevicePicker(apkPath string) {
	deviceChoice := getDeviceChoice()
	if len(deviceChoice) <= 1 {
		color.Red("No devices connected")
		os.Exit(1)
	}
	prompt := pui.Select{
		Label: "Select Target:",
		Items: deviceChoice,
	}

	_, result, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		os.Exit(1)
	}

	deviceID := extractDeviceID(result)

	if deviceID == "All" {
		allSucceeded := true
		for _, rawId := range deviceChoice[1:] {
			if !installTo(extractDeviceID(rawId), apkPath) {
				allSucceeded = false
			}
		}
		if !allSucceeded {
			os.Exit(1)
		}
	} else if !installTo(deviceID, apkPath) {
		os.Exit(1)
	}
}

func installTo(deviceID string, apkPath string) bool {
	fmt.Println("Installing to " + deviceID)
	var comamndReturn adb.CommandReturn
	if isNoTimeout {
		comamndReturn = adb.InstallToWithTimeout(deviceID, apkPath, -1*time.Second)
	} else {
		comamndReturn = adb.InstallTo(deviceID, apkPath)
	}

	if comamndReturn.Error != nil {
		printCommandError(comamndReturn)
		color.Red("Cannot install %s to %s", apkPath, deviceID)
		return false
	}
	return true
}

func getDeviceChoice() []string {
	rawList := adb.ConnectedDevices()
	arrayOfChoice := strings.Split(string(rawList.Output), "\n")
	choiceSize := len(arrayOfChoice)

	if choiceSize == 1 {
		return []string{}
	}

	deviceChoice := []string{"All"}
	for i, v := range arrayOfChoice {
		if i == 0 {
			continue
		}
		if len(strings.TrimSpace(v)) == 0 {
			continue
		}
		deviceChoice = append(deviceChoice, v)
	}

	return deviceChoice
}

func shouldShowDevicePicker(output string) bool {
	return strings.Contains(output, "more than one device/emulator")
}

func canRecoverVersionDowngrade(apkPath string, text string) bool {
	if !strings.Contains(text, "INSTALL_FAILED_VERSION_DOWNGRADE") {
		return false
	}

	packageName, err := analyzer.PackageName(apkPath)
	if err != nil {
		return false
	}

	var isConfirmed bool = isYes
	if !isConfirmed {
		message := fmt.Sprintf("%s already exist, do you want to uninstall first?", color.CyanString(packageName))
		isConfirmed = utils.ShowYesOrNoConfirmation(message)
	}

	if isConfirmed {
		return uninstall(packageName)
	}
	return false
}

func uninstall(packageName string) bool {
	fmt.Printf("Uninstalling %s…\n", color.CyanString(packageName))
	commandReturn := adb.Uninstall(packageName)
	if commandReturn.Error != nil {
		printCommandError(commandReturn)
		return false
	}
	return true
}

func extractDeviceID(deviceInfo string) string {
	return strings.Split(deviceInfo, "\t")[0]
}

func extractErrorMessage(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "adb: failed") || strings.Contains(line, "Failure [") {
			return line
		}
	}
	return ""
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().BoolVarP(&isYes, "yes", "y", false, "Set auto confirm")
	installCmd.Flags().BoolVarP(&isVerbose, "verbose", "v", false, "Show verbose error message")
	installCmd.Flags().BoolVarP(&isNoTimeout, "no-timeout", "", false, "Disable timeout for long running install")
}
