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
	"fmt"

	adb "github.com/esafirm/gadb/adb"
	"github.com/esafirm/gadb/config"
	"github.com/spf13/cobra"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start [package]",
	Short: "Start Android application",
	Run: func(cmd *cobra.Command, args []string) {
		packageName, err := config.GetPackageNameFromArgs(args)
		if err == nil {
			adb.Start(packageName)
		} else {
			fmt.Println(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(focusCmd)
}
