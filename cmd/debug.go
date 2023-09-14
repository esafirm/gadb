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
	"time"

	adb "github.com/esafirm/gadb/adb"
	"github.com/spf13/cobra"
)

// persistent mode to be on/off
var persistent bool

// if true, clear the persisten debug flag
var clear bool

// if true, restart the application
var isRestart bool

var debugCmd = &cobra.Command{
	Use:   "debug",
	Args:  cobra.MinimumNArgs(1),
	Short: "gadb debug [PACKAGE] [OPTIONS]",
	Run: func(cmd *cobra.Command, args []string) {
		packageName := args[0]
		if clear {
			clearDebug(packageName)
		} else {
			debug(persistent, packageName)
		}
	},
}

func clearDebug(packageName string) {
	adb.DebugCancel(packageName)
}

func debug(isPersistent bool, packageName string) {
	adb.Stop(packageName)
	time.Sleep(3 * time.Second)
	adb.Debug(persistent, packageName)

	if isRestart {
		adb.Restart(packageName)
	}
}

func init() {
	rootCmd.AddCommand(debugCmd)
	debugCmd.Flags().BoolVarP(&persistent, "persistent", "p", true, "Set waiting for debug mode until nodebug is triggered")
	debugCmd.Flags().BoolVarP(&clear, "clear", "c", false, "Clear waiting debug status")
	debugCmd.Flags().BoolVarP(&isRestart, "restart", "r", true, "Restart the application")
}
