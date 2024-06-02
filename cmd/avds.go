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

	adb "github.com/esafirm/gadb/adb"
	pui "github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// Option: run wipe command to the selected AVD
var isWipe bool

// Option: run AVD in cold boot state
var isColdBoot bool

var avdsCmd = &cobra.Command{
	Use:   "avds [emulator name]",
	Short: "List all available AVD(s) and run it",
	Run: func(cmd *cobra.Command, args []string) {
		if isWipe {
			showAvdsWipe()
			return
		}

		if len(args) == 0 {
			showAvdSelection()
		} else {
			adb.AvdRun(args[0], isColdBoot)
		}
	},
}

func showAvdsWipe() {
	result := showAvdsPrompt("Select AVD to be wiped")
	adb.AvdWipe(result)
}

func showAvdSelection() {
	result := showAvdsPrompt("Select AVD to run")
	fmt.Printf("Launching %s AVD", result)
	adb.AvdRun(result, isColdBoot)
}

func showAvdsPrompt(caption string) string {
	commandResult := adb.AvdListFormatted()
	if commandResult.Error != nil {
		panic(commandResult.Error)
	}

	prompt := pui.Select{
		Label: caption,
		Items: commandResult.AvdList,
	}

	_, result, err := prompt.Run()
	if err != nil {
		panic(err)
	}

	return result
}

func init() {
	rootCmd.AddCommand(avdsCmd)
	avdsCmd.Flags().BoolVarP(&isWipe, "wipe", "w", false, "Wipe AVD data")
	avdsCmd.Flags().BoolVarP(&isColdBoot, "cold", "c", false, "Set running AVD in cold state")
}
