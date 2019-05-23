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
	"github.com/spf13/cobra"
)

var clearData bool

// restartCmd represents the restart command
var restartCmd = &cobra.Command{
	Use:   "restart [package] [flags]",
	Args:  cobra.MinimumNArgs(1),
	Short: "Restart application",
	Run: func(cmd *cobra.Command, args []string) {
		packageName := args[0]
		if clearData {
			adb.ClearData(packageName)
		}
		adb.Restart(packageName)
	},
}

func init() {
	rootCmd.AddCommand(restartCmd)
	restartCmd.Flags().BoolVarP(&clearData, "clear", "c", false, "Restart and clear application data")
}
