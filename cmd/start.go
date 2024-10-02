// Copyright © 2019 Esa Firman esafirm21@gmail.com
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
	adb "github.com/esafirm/gadb/adb"
	"github.com/esafirm/gadb/config"
	"github.com/esafirm/gadb/utils"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start [package]",
	Short: "Start Android application",
	Run: func(cmd *cobra.Command, args []string) {

		packageName := extractPackageName(args)
		if packageName == "" {
			packageName = config.GetPackageNameOrDefault(func() string {
				return utils.SelectThirdPartyPackage()
			})
		}

		if packageName == "" {
			color.Red("No package name found")
			return
		}

		color.Cyan("Starting %s", packageName)
		adb.Start(packageName)
	},
}

func extractPackageName(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

func init() {
	rootCmd.AddCommand(startCmd)
}
