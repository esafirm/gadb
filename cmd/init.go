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
	"os"

	"github.com/esafirm/gadb/config"
	promptui "github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create gadb configuration",
	Run: func(cmd *cobra.Command, args []string) {
		validateDirectory()
	},
}

func validateDirectory() {
	_, err := os.Stat("settings.gradle")
	os.IsNotExist(err)

	if err != nil {
		validation := func(input string) error {
			if input == "y" || input == "Y" || input == "n" || input == "N" {
				return nil
			}
			return errors.New("Answer not valid")
		}

		prompt := promptui.Prompt{
			Label:    "It looks like this directory doesn't belong to android project, continue? [Y,n]",
			Validate: validation,
		}

		result, _ := prompt.Run()

		if result == "Y" || result == "y" {
			askData()
		}
	} else {
		askData()
	}
}

func askData() {
	questions := []string{
		"Project pacakge",
	}
	answers := []string{}

	questionIndex := 0
	for questionIndex < len(questions) {
		result, err := askQuestion(questions[questionIndex])

		if err == nil {
			answers = append(answers, result)
			questionIndex++
		}
	}

	config.WriteConfig(
		config.Config{
			PackageName: answers[0],
		},
	)
}

func askQuestion(questionLabel string) (string, error) {
	notEmptyValidation := func(input string) error {
		if len(input) == 0 {
			return errors.New("answers cannot be empty")
		}
		return nil
	}

	prompt := promptui.Prompt{
		Label:    questionLabel,
		Validate: notEmptyValidation,
	}

	result, err := prompt.Run()

	if err != nil {
		return "", errors.New("promp failed")
	}

	return result, nil
}

func init() {
	rootCmd.AddCommand(initCmd)
}
