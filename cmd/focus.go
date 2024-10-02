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
	"strings"

	adb "github.com/esafirm/gadb/adb"
	color "github.com/fatih/color"
	"github.com/spf13/cobra"
)

// focusCmd represents the start command
var focusCmd = &cobra.Command{
	Use:   "focus",
	Short: "Get the info about the current focused app",
	Run: func(cmd *cobra.Command, args []string) {
		result := adb.DumpSys("window", "displays")

		if result.Error != nil {
			println("Error: ", result.Error.Error())
			return
		}

		resultString := string(result.Output[:])

		focusWindowRegex := regexp.MustCompile(`mCurrentFocus=Window{(?P<focusInfo>.*)}`)
		matches := focusWindowRegex.FindStringSubmatch(resultString)

		fmt.Printf("Focus window: %s\n", color.GreenString(getSplitLastIndex(matches[1], " ", 1)))

		focusAppRegex := regexp.MustCompile(`mFocusedApp=ActivityRecord{(?P<focusInfo>.*)}`)
		matches = focusAppRegex.FindStringSubmatch(resultString)

		fmt.Printf("Focus app: %s\n", color.GreenString(getSplitLastIndex(matches[1], " ", 2)))

		resultString = string(adb.DumpSys("activity", "top").Output[:])
		focusFragment := regexp.MustCompile(`Active Fragments:\n(.*){`)
		matches = focusFragment.FindStringSubmatch(resultString)

		// Check if there's any fragment
		if len(matches) < 2 {
			fmt.Printf("Focus fragment: %s\n", color.GreenString("-"))
		} else {
			fragment := strings.TrimSpace(matches[1])
			fmt.Printf("Focus fragment: %s\n", color.GreenString(fragment))
		}

	},
}

func getSplitLastIndex(s string, separator string, index int) string {
	splits := strings.Split(s, separator)
	return splits[len(splits)-index]
}

func init() {
	rootCmd.AddCommand(focusCmd)
}
