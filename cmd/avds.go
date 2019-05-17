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
	"os/exec"

	"github.com/spf13/cobra"
)

var avdsCmd = &cobra.Command{
	Use:   "avds",
	Short: "List and run AVD",
	Long: `Usage:
	gadb avds				List AVD(s)
	gabd avds <emulator name>		Run AVD`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			listAvds()
		} else {
			runAvd(args[0])
		}
	},
}

func runAvd(emulatorName string) {
	cmd := exec.Command("emulator", "@"+emulatorName)
	cmd.Start()
}

func listAvds() {
	runCommandWithOutput("emulator", "-list-avds")
}

func runCommandWithOutput(name string, arg ...string) {
	output, err := exec.Command(name, arg...).CombinedOutput()
	if err == nil {
		fmt.Println(string(output))
	}
}

func init() {
	rootCmd.AddCommand(avdsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// avdsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// avdsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
