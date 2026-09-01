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
	"encoding/json"
	"fmt"

	adb "github.com/esafirm/gadb/adb"
	color "github.com/fatih/color"
	"github.com/spf13/cobra"
)

var specJSON bool

// specCmd represents the spec command
var specCmd = &cobra.Command{
	Use:   "spec",
	Short: "Print connected device specs (model, OS version, density, screen size)",
	Run: func(cmd *cobra.Command, args []string) {
		result := adb.Spec()

		if specJSON {
			printSpecJSON(result)
			return
		}

		if result.Error != nil {
			color.Red("Error: %v", result.Error)
			return
		}

		spec := result.Spec
		fmt.Printf("Model: %s\n", color.GreenString(spec.Model))
		fmt.Printf("Manufacturer: %s\n", color.GreenString(spec.Manufacturer))
		fmt.Printf("OS Version: %s\n", color.GreenString("Android "+spec.OSVersion+" (API "+spec.SDKVersion+")"))
		fmt.Printf("Density: %s\n", color.GreenString(spec.DensityDpi+" dpi"))
		fmt.Printf("Screen size: %s\n", color.GreenString(
			spec.WidthPx+"x"+spec.HeightPx+" px ("+spec.WidthDp+"x"+spec.HeightDp+" dp)",
		))
	},
}

func printSpecJSON(result adb.DeviceSpecCommandReturn) {
	if result.Error != nil {
		output, _ := json.MarshalIndent(map[string]string{"error": result.Error.Error()}, "", "  ")
		fmt.Println(string(output))
		return
	}

	output, _ := json.MarshalIndent(result.Spec, "", "  ")
	fmt.Println(string(output))
}

func init() {
	rootCmd.AddCommand(specCmd)
	specCmd.Flags().BoolVarP(&specJSON, "json", "j", false, "Print result as JSON")
}
