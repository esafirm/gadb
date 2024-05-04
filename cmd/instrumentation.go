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
	"errors"
	"fmt"
	"strings"

	"github.com/esafirm/gadb/config"
	"github.com/esafirm/gadb/utils"
	"github.com/spf13/cobra"

	adb "github.com/esafirm/gadb/adb"
)

// error message constants
const (
	errPackageNotFound string = "package name is not supplied in parameter or config"
	errAdbCommand      string = "adb command error"
)

// Option: enable debug mode
var optDebug bool

// Option: enable package selection mode
var optPackageSelection bool

// insCmd represents the instrumentation command
var insCmd = &cobra.Command{
	Use:   "instrumentation or instrumentation <package name>",
	Short: "Run instrumentation test in your device",
	RunE: func(cmd *cobra.Command, args []string) error {

		var packageName string
		if len(args) > 0 {
			if optPackageSelection {
				return errors.New("cannot use package selection mode with package name")
			}
			packageName = args[0]
		} else if optPackageSelection {
			packageName = selectPackage()
		} else {
			config, err := config.ReadConfig()
			if err != nil {
				return errors.New(errPackageNotFound)
			}
			packageName = config.PackageName
		}

		return runInstrumentation(strings.TrimSpace(packageName), optDebug)
	},
}

func selectPackage() string {
	commandReturn := adb.ListPacakgesThirdPartyFormatted()
	if commandReturn.Error != nil {
		fmt.Println("Error getting package list:", commandReturn.Error)
		return ""
	}

	_, selectedPacakge := utils.ShowSelection("Select package", commandReturn.PackageList)
	return selectedPacakge
}

func selectInstrumentations(selectedPackage string) (string, error) {
	commandResult := adb.ListPackagesInstrumentationFormatted()
	if commandResult.Error != nil {
		return "", fmt.Errorf("%s: %s", errAdbCommand, commandResult.Error.Error())
	}

	for _, p := range commandResult.PackageList {
		if p.Target == selectedPackage {
			return p.PackageName, nil
		}
	}

	return "", fmt.Errorf("no instrumentation package found for %s", selectedPackage)
}

func createFilterExtras() []string {
	i, selectedItem := utils.ShowSelection("Select filter type", []string{"done ✅", "class", "package", "size"})

	// If the user selects "done", return an empty slice
	if i == 0 {
		return []string{}
	}

	if selectedItem == "size" {
		_, size := utils.ShowSelection("Select size", []string{"small", "medium", "large"})
		return adb.ExtraFilter("size", size)
	}
	if selectedItem == "class" || selectedItem == "package" {
		class := utils.ShowPrompt("Enter the class/package name")
		return adb.ExtraFilter("class", class)
	}

	return []string{}
}

func runInstrumentation(packageName string, runInDebug bool) error {
	commandReturn := adb.ListPacakgesThirdPartyFormatted()
	if commandReturn.Error != nil {
		return fmt.Errorf("%s - get package list: %s", errAdbCommand, commandReturn.Error)
	}

	installed := packageInstalled(packageName, commandReturn.PackageList)
	if !installed {
		return errors.New("Package is not installed: " + packageName)
	}

	fmt.Printf("Package exists: %s\n", focus(packageName))

	instrumentationPackage, err := selectInstrumentations(packageName)
	if err != nil {
		return err
	}

	// Create filter extras, stop if no input
	filterExtras := []string{}
	for {
		newExtra := createFilterExtras()
		if len(newExtra) == 0 {
			break
		}
		filterExtras = append(filterExtras, newExtra...)
	}

	if runInDebug {
		filterExtras = append(filterExtras, adb.ExtraDebug()...)
	}

	fmt.Println("Starting instrumentation test for", instrumentationPackage)

	result := adb.StartInstrumentation(instrumentationPackage, filterExtras...)
	if result.Error != nil {
		return fmt.Errorf("%s: %s", errAdbCommand, string(result.Output))
	}

	return nil
}

func packageInstalled(packageName string, installedPackages []string) bool {
	for _, p := range installedPackages {
		if p == packageName {
			return true
		}
	}
	return false

}

func init() {
	rootCmd.AddCommand(insCmd)
	insCmd.Flags().BoolVarP(&optDebug, "debug", "d", false, "enable debug mode")
	insCmd.Flags().BoolVarP(&optPackageSelection, "package-selection", "p", false, "enable package selection mode")
}
