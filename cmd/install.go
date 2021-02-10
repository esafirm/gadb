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
	"errors"
	"fmt"
	"os"
	"strings"

	adb "github.com/esafirm/gadb/adb"
	analyzer "github.com/esafirm/gadb/apkanalyzer"
	pui "github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var isYes bool

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
		fmt.Printf("%s\n", errorMsg)
	}
	cmd.Help()
	os.Exit(0)
}

func runCommand(apkPath string) {
	comamndReturn := adb.ReInstall(apkPath)

	if comamndReturn.Error != nil {
		output := string(comamndReturn.Output)
		if canRecoverAlreadyExist(apkPath, output) {
			runCommand(apkPath)
			return
		}
		if canRecoverVersionDowngrade(apkPath, output) {
			runCommand(apkPath)
			return
		}
		if shouldShowDevicePicker(output) {
			showDevicePicker(apkPath)
			return
		}
		fmt.Println(string(comamndReturn.Output))
	}
}

func showDevicePicker(apkPath string) {
	deviceChoice := getDeviceChoice()
	prompt := pui.Select{
		Label: "Select Target:",
		Items: deviceChoice,
	}

	_, result, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}

	deviceID := strings.Split(result, "\t")[0]

	if deviceID == "All" {
		for _, id := range deviceChoice {
			installTo(id, apkPath)
		}
	} else {
		installTo(deviceID, apkPath)
	}
}

func installTo(deviceID string, apkPath string) {
	fmt.Println("Installing to " + deviceID)
	adb.InstallTo(deviceID, apkPath)
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
	packageName, err := analyzer.PackageName(apkPath)
	if err != nil {
		panic(err)
	}

	isVersionDowngradeProblem := strings.Contains(text, "INSTALL_FAILED_VERSION_DOWNGRADE")

	var isConfirmed bool = isYes
	if isVersionDowngradeProblem && !isConfirmed {
		message := fmt.Sprintf("%s already exist, do you want to uninstall first?", packageName)
		isConfirmed = confirmUninstall(message)
	}

	if isConfirmed {
		uninstall(packageName)
	}
	return false
}

func canRecoverAlreadyExist(apkPath string, text string) bool {
	var isConfirmed bool = isYes
	if !isConfirmed {
		isConfirmed = confirmUninstall("Do you want to uninstall first?")
	}

	if isConfirmed {
		var index = strings.Index(text, "re-install") + len("re-install")
		var withoutIndex = strings.Index(text, "without")
		var packageName = strings.TrimSpace(text[index:withoutIndex])

		uninstall(packageName)
		return true
	}

	return false
}

func confirmUninstall(message string) bool {
	validation := func(input string) error {
		if input == "y" || input == "Y" || input == "n" || input == "N" {
			return nil
		}
		return errors.New("Answer not valid")
	}

	prompt := pui.Prompt{
		Label:    message + " [Y,n]",
		Validate: validation,
	}

	result, _ := prompt.Run()

	if result == "Y" || result == "y" {
		return true
	}
	return false
}

func uninstall(packageName string) {
	fmt.Printf("Uninstalling %s ~\n", packageName)
	adb.Uninstall(packageName)
}

func init() {
	rootCmd.AddCommand(installCmd)
	mockCmd.Flags().BoolVarP(&isYes, "yes", "y", false, "Set auto confirm")
}
