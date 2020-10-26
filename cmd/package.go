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

	analyzer "github.com/esafirm/gadb/apkanalyzer"

	"github.com/spf13/cobra"
)

var pakcageCmd = &cobra.Command{
	Use:   "package [apk]",
	Short: "Print package name from the APK",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("You must provide the APK")
			return
		}

		pakckageName, err := analyzer.PackageName(args[0])
		if err != nil {
			fmt.Printf("Package name: %s", pakckageName)
		} else {
			panic(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(pakcageCmd)
}
