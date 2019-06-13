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

	"github.com/spf13/cobra"
)

// storeCmd represents the store command
var storeCmd = &cobra.Command{
	Use:   "store",
	Short: "A brief description of your command",
	Run: func(cmd *cobra.Command, args []string) {
		openStore()
	},
}

func openStore() {
	config, err := readConfig()
	if err == nil {
		openbrowser("https://play.google.com/store/apps/details?id=" + config.PackageName)
	} else {
		fmt.Println("Project not initialized or corrupt config file")
	}

}

func openbrowser(url string) {
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
