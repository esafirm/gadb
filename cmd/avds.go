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
	adb "github.com/esafirm/gadb/adb"
	"github.com/spf13/cobra"
)

var avdsCmd = &cobra.Command{
	Use:   "avds [emulator name]",
	Short: "List all available AVD(s) and run it",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			adb.AvdList()
		} else {
			adb.AvdRun(args[0])
		}
	},
}

func init() {
	rootCmd.AddCommand(avdsCmd)
}
