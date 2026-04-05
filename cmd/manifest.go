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
	"regexp"

	analyzer "github.com/esafirm/gadb/apkanalyzer"

	"github.com/spf13/cobra"
)

var onlyHighlights bool

var manifestCmd = &cobra.Command{
	Use:   "manifest [apk]",
	Short: "Print the AndroidManifest.xml and highlights from the APK",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("You must provide the APK")
			return
		}

		commandReturn := analyzer.Manifest(args[0])
		if commandReturn.Error != nil {
			fmt.Println(commandReturn.Error)
			return
		}

		manifestContent := string(commandReturn.Output)
		highlights := extractHighlightsFromManifest(manifestContent)

		keys := []string{"Package Name", "Version Name", "Version Code", "Min SDK", "Target SDK"}
		for _, key := range keys {
			if value, ok := highlights[key]; ok {
				fmt.Printf("%s: %s\n", key, value)
			}
		}

		if !onlyHighlights {
			fmt.Println("\nManifest:")
			fmt.Println(manifestContent)
		}
	},
}

func extractHighlightsFromManifest(manifest string) map[string]string {
	highlights := make(map[string]string)

	packageRegex := regexp.MustCompile(`package="([^"]*)"`)
	versionCodeRegex := regexp.MustCompile(`android:versionCode="([^"]*)"`)
	versionNameRegex := regexp.MustCompile(`android:versionName="([^"]*)"`)
	minSdkRegex := regexp.MustCompile(`android:minSdkVersion="([^"]*)"`)
	targetSdkRegex := regexp.MustCompile(`android:targetSdkVersion="([^"]*)"`)

	if match := packageRegex.FindStringSubmatch(manifest); len(match) > 1 {
		highlights["Package Name"] = match[1]
	}
	if match := versionCodeRegex.FindStringSubmatch(manifest); len(match) > 1 {
		highlights["Version Code"] = match[1]
	}
	if match := versionNameRegex.FindStringSubmatch(manifest); len(match) > 1 {
		highlights["Version Name"] = match[1]
	}
	if match := minSdkRegex.FindStringSubmatch(manifest); len(match) > 1 {
		highlights["Min SDK"] = match[1]
	}
	if match := targetSdkRegex.FindStringSubmatch(manifest); len(match) > 1 {
		highlights["Target SDK"] = match[1]
	}

	return highlights
}

func init() {
	manifestCmd.Flags().BoolVarP(&onlyHighlights, "highlights", "i", false, "Only show highlights (e.g., package name)")
	rootCmd.AddCommand(manifestCmd)
}
