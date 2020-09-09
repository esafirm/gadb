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
	httpmock "github.com/esafirm/gadb/httpmock"
	"github.com/spf13/cobra"
	"os"
)

var mockCmd = &cobra.Command{
	Use:   "mock",
	Short: "Set mock for OkHttp interceptor",
	Run: func(cmd *cobra.Command, args []string) {

		var prefix *string
		if len(args) == 0 {
			prefix = nil
		} else {
			prefix = &args[0]
		}

		forwardList := adb.ForwardList()

		// Handle blank
		if len(forwardList.Output) <= 1 {
			adb.Forward(httpmock.DEFAULT_PORT)
		}

		println("Preparing…")
		dir, _ := os.Getwd()
		_, mockStrings := httpmock.ReadMockFiles(dir, prefix)

		println("Connecting…")
		httpmock.Connect(mockStrings)
	},
}

func init() {
	rootCmd.AddCommand(mockCmd)
}
