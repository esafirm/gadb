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
	"os"

	adb "github.com/esafirm/gadb/adb"
	httpmock "github.com/esafirm/gadb/httpmock"
	"github.com/spf13/cobra"
)

var dir string
var file string
var prefix string

var mockCmd = &cobra.Command{
	Use:   "mock",
	Short: "Set mock for OkHttp interceptor",
	Args:  cobra.MaximumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		forwardList := adb.ForwardList()

		// Handle blank
		if len(forwardList.Output) <= 1 {
			adb.Forward(httpmock.DEFAULT_PORT)
		}

		println("Preparing…")

		if &dir == nil {
			currentDir, _ := os.Getwd()
			dir = currentDir
		}

		var mockStrings []string
		if &file == nil {
			_, mockStrings = httpmock.ReadMockFiles(dir, &prefix)
		} else {
			_, mockStrings = httpmock.ReadMockFile(file)
		}

		httpmock.Connect(mockStrings)
	},
}

func init() {
	rootCmd.AddCommand(mockCmd)
	mockCmd.Flags().StringVarP(&dir, "directory", "d", "", "Set mock file to all json in passed directory")
	mockCmd.Flags().StringVarP(&file, "file", "f", "", "Set mock file to passed file")
	mockCmd.Flags().StringVarP(&prefix, "prefix", "p", "", "Set prefix for mock file")
}
