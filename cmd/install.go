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

	"github.com/spf13/cobra"
	adb "github.com/esafirm/gadb/adb"
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "install <apk_path>",
	Long:  `Install APK to single or multiple`,
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
	comamndReturn := adb.Install(apkPath)

	if comamndReturn.Error != nil {
		canRecover := recoverError(string(comamndReturn.Output))
		if canRecover {
			runCommand(apkPath)
		}
	}
}

func recoverError(text string) bool {
	isAlreadyExistProblem := strings.Contains(text, "ALREADY_EXISTS")

	if isAlreadyExistProblem {
		var index = strings.Index(text, "re-install") + len("re-install")
		var withoutIndex = strings.Index(text, "without")
		var packageName = strings.TrimSpace(text[index:withoutIndex])

		uninstall(packageName)

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

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// installCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// installCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
