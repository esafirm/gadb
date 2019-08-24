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
	"github.com/spf13/cobra"
)

// clearCmd represents the clear command
var clearCmd = &cobra.Command{
	Use:   "clear or clear <application id> ",
	Short: "Trigger clear data to your",
	Run: func(cmd *cobra.Command, args []string) {
		packageName := args[0]
		clearAppData(getPackageName(packageName))
	},
}

func clearAppData(packageName string) {
	if len(packageName) == 0 {
		config, err := config.ReadConfig()
		if err == nil {
			adb.ClearData(config.PackageName)
		}
	} else {
		adb.ClearData(packageName)
	}
}

func getPackageName(packageName string) string {
	if len(packageName) == 0 {
		config, err := config.ReadConfig()
		if err != nil {
			return ""
		}
		return config.PackageName
	}
	return packageName
}

func init() {
	rootCmd.AddCommand(clearCmd)
}
