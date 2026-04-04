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

	"github.com/esafirm/gadb/config"
	color "github.com/fatih/color"
	promptui "github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var (
	configAI    bool
	configShow  bool
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage gadb configuration",
	Run: func(cmd *cobra.Command, args []string) {
		if configAI {
			configureAI()
		} else if configShow {
			showConfig()
		} else {
			cmd.Help()
		}
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.Flags().BoolVarP(&configAI, "ai", "a", false, "Configure AI settings")
	configCmd.Flags().BoolVarP(&configShow, "show", "s", false, "Show current configuration")
}

func configureAI() {
	fmt.Println()
	color.Cyan("🤖 AI Configuration Setup")
	fmt.Println()

	// Provider selection
	providerSelect := promptui.Select{
		Label: "Select AI Provider",
		Items: []string{"gemini", "anthropic", "openai"},
	}

	_, provider, err := providerSelect.Run()
	if err != nil {
		color.Red("Failed to select provider: %v", err)
		return
	}

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
			color.Red("Failed to enter API key: %v", err)
			return
		}
	} else {
		color.HiBlack("Note: Gemini uses OAuth authentication - no API key needed")
		color.HiBlack("Make sure you're logged in with 'gemini auth login'")
	}

	// Model selection (with defaults)
	var defaultModel string
	switch provider {
	case "anthropic":
		defaultModel = "claude-3-sonnet-20240229"
	case "openai":
		defaultModel = "gpt-4"
	case "gemini":
		defaultModel = "gemini-2.0-flash-exp" // Gemini Flash is faster and good for this use case
	}

	modelPrompt := promptui.Prompt{
		Label:   fmt.Sprintf("Enter Model Name (default: %s)", defaultModel),
		Default: defaultModel,
	}

	model, err := modelPrompt.Run()
	if err != nil {
		model = defaultModel
	}

	// Optional endpoint
	endpointPrompt := promptui.Prompt{
		Label:   "Enter Custom Endpoint (optional, press Enter to skip)",
		Default: "",
	}

	endpoint, err := endpointPrompt.Run()
	if err != nil {
		endpoint = ""
	}

	// Build AI config
	aiConfig := config.AIConfig{
		Provider:    provider,
		APIKey:      apiKey,
		Model:       model,
		Endpoint:    endpoint,
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	// Save configuration
	if err := config.SetAIConfig(aiConfig); err != nil {
		color.Red("Failed to save AI configuration: %v", err)
		return
	}

	color.Green("✓ AI configuration saved successfully!")
	fmt.Println()
	color.HiBlack("Provider: %s", provider)
	color.HiBlack("Model: %s", model)
	if endpoint != "" {
		color.HiBlack("Endpoint: %s", endpoint)
	}
	fmt.Println()
	color.Green("You can now use 'gadb analyze --ai' to analyze crashes with AI!")
}

func showConfig() {
	fmt.Println()
	color.Cyan("📋 Current Configuration")
	fmt.Println()

	// Try to read existing config
	cfg, err := config.ReadConfig()
	if err != nil {
		color.Yellow("No configuration file found. Run 'gadb init' to create one.")
		return
	}

	// Display package info
	if cfg.PackageName != "" {
		color.HiBlack("Package Name: %s", cfg.PackageName)
	} else {
		color.Yellow("Package Name: (not set)")
	}

	// Display AI info
	fmt.Println()
	if cfg.AI.APIKey != "" {
		color.HiBlack("AI Provider: %s", cfg.AI.Provider)
		color.HiBlack("AI Model: %s", cfg.AI.Model)
		color.HiBlack("AI Status: %s", color.GreenString("Configured"))
		if cfg.AI.Endpoint != "" {
			color.HiBlack("Custom Endpoint: %s", cfg.AI.Endpoint)
		}
	} else {
		color.HiBlack("AI Status: %s", color.YellowString("Not configured"))
		fmt.Println()
		color.HiBlack("Run 'gadb config --ai' to set up AI for crash analysis")
	}

	fmt.Println()
}
