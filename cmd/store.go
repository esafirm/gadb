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
	"log"
	"os/exec"
	"runtime"

	"github.com/esafirm/gadb/config"
	"github.com/spf13/cobra"
)

// storeCmd represents the store command
var storeCmd = &cobra.Command{
	Use:   "store [package]",
	Short: "Open playstore page",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			openStore(args[0])
		} else {
			openStore("")
		}
	},
}

func openStore(packageName string) {
	if len(packageName) == 0 {
		config, err := config.ReadConfig()
		if err == nil {
			openbrowser(config.PackageName)
		} else {
			fmt.Println("Project not initialized or corrupt config file")
		}
	} else {
		openbrowser(packageName)
	}
}

func openbrowser(packageName string) {
	url := "https://play.google.com/store/apps/details?id=" + packageName
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}
}

func init() {
	rootCmd.AddCommand(storeCmd)
}
