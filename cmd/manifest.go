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
	"os/exec"

	"github.com/spf13/cobra"
)

const ANALYZER = "apkanalyzer"

var manifestCmd = &cobra.Command{
	Use:   "manifest [apk]",
	Short: "Print the AndroidManifest.xml from the APK",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("You must provide the APK")
			return
		}

		_, err := exec.LookPath(ANALYZER)
		if err != nil {
			fmt.Println(ANALYZER + " is not in the path")
		}

		output, err := exec.Command(ANALYZER, "manifest", "print", args[0]).CombinedOutput()
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(string(output))
		}
	},
}

func init() {
	rootCmd.AddCommand(manifestCmd)
}
