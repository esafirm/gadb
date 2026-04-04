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
	"os"
	"strings"

	"github.com/esafirm/gadb/config"
	color "github.com/fatih/color"
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
	// Ask for package name
	packagePrompt := promptui.Prompt{
		Label:    "Project package",
		Validate: func(input string) error {
			if len(input) == 0 {
				return errors.New("package name cannot be empty")
			}
			return nil
		},
	}

	packageName, err := packagePrompt.Run()
	if err != nil {
		color.Red("Failed to get package name: %v", err)
		return
	}

	// Ask if user wants to configure AI
	aiPrompt := promptui.Prompt{
		Label:     "Configure AI for crash analysis? [Y/n]",
		IsConfirm: true,
	}

	configureAI, _ := aiPrompt.Run()

	// Build config
	cfg := config.Config{
		PackageName: packageName,
	}

	// Configure AI if requested
	if configureAI == "" || strings.ToLower(configureAI) == "y" {
		// Provider selection
		providerSelect := promptui.Select{
			Label: "Select AI Provider",
			Items: []string{"gemini", "anthropic", "openai"},
		}

		_, provider, err := providerSelect.Run()
		if err != nil {
			color.Yellow("Skipping AI configuration due to error: %v", err)
		} else {
			var apiKey string

			// API Key input (required for Anthropic/OpenAI, optional for Gemini)
			if provider != "gemini" {
				apiKeyPrompt := promptui.Prompt{
					Label: "Enter API Key",
					Mask:  '*',
					Validate: func(input string) error {
						if len(input) == 0 {
							return errors.New("API key cannot be empty")
						}
						return nil
					},
				}

				apiKey, err = apiKeyPrompt.Run()
				if err != nil {
					color.Yellow("Skipping AI configuration")
					return
				}
			} else {
				color.HiBlack("Note: Gemini uses OAuth authentication - no API key needed")
				color.HiBlack("Make sure you're logged in with 'gemini auth login'")
			}

			// Set default model based on provider
			var defaultModel string
			switch provider {
			case "anthropic":
				defaultModel = "claude-3-sonnet-20240229"
			case "openai":
				defaultModel = "gpt-4"
			case "gemini":
				defaultModel = "gemini-2.0-flash-exp"
			}

			cfg.AI = config.AIConfig{
				Provider:    provider,
				APIKey:      apiKey,
				Model:       defaultModel,
				MaxTokens:   1000,
				Temperature: 0.7,
			}

			color.Green("✓ AI configuration added")
		}
	}

	// Write config
	config.WriteConfig(cfg)

	color.Green("✓ Configuration saved successfully!")
	fmt.Println()
	if cfg.AI.APIKey != "" || cfg.AI.Provider == "gemini" {
		color.Green("✓ AI is configured - you can now use 'gadb analyze --ai'")
	} else {
		color.HiBlack("Run 'gadb config --ai' to configure AI for crash analysis")
	}
}

func init() {
	rootCmd.AddCommand(initCmd)
}
