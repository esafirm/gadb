package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
)

var fileName = "gadb.setting.json"

// AIConfig holds AI/LLM configuration
type AIConfig struct {
	Provider    string `json:"provider"`     // "anthropic" or "openai"
	APIKey      string `json:"apiKey"`       // API key for the provider
	Model       string `json:"model"`        // Model name (e.g., "claude-3-sonnet", "gpt-4")
	Endpoint    string `json:"endpoint"`     // Custom endpoint (optional)
	MaxTokens   int    `json:"maxTokens"`    // Maximum tokens in response
	Temperature float64 `json:"temperature"` // Temperature for generation (0.0-1.0)
}

// Config that exported to json file
type Config struct {
	PackageName string   `json:"packageName"`
	AI          AIConfig `json:"ai"`
}

func WriteConfig(data Config) {
	dataJSON, err := json.Marshal(data)
	_ = os.WriteFile(fileName, dataJSON, fs.FileMode(0644))

	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("Project initialized!")
	}
}

func ReadConfig() (Config, error) {
	if !isConfigExist() {
		return Config{}, errors.New("Config file doesn't exist. Create one using gadb init")
	}
	file, _ := os.ReadFile(fileName)
	config := Config{}
	err := json.Unmarshal(file, &config)

	return config, err
}

func isConfigExist() bool {
	_, err := os.Stat(fileName)
	return !os.IsNotExist(err)
}

// GetPackageNameOrDefault try to get the package name from config file or return default value
func GetPackageNameOrDefault(defaultValue func() string) string {
	return getConfigWithFallback(func(config Config) string {
		return config.PackageName
	}, defaultValue)
}

type fetcher func(Config) string

func getConfigWithFallback(f fetcher, fallback func() string) string {
	config, err := ReadConfig()
	if err != nil {
		return fallback()
	}

	fetched := f(config)
	if fetched == "" {
		return fallback()
	}
	return fetched
}

// GetAIConfig returns the AI configuration
func GetAIConfig() (AIConfig, error) {
	config, err := ReadConfig()
	if err != nil {
		return AIConfig{}, fmt.Errorf("config file doesn't exist. Create one using gadb init")
	}

	// Set defaults if not configured
	if config.AI.Provider == "" {
		config.AI.Provider = "anthropic"
	}
	if config.AI.Model == "" {
		if config.AI.Provider == "anthropic" {
			config.AI.Model = "claude-3-sonnet-20240229"
		} else {
			config.AI.Model = "gpt-4"
		}
	}
	if config.AI.MaxTokens == 0 {
		config.AI.MaxTokens = 1000
	}
	if config.AI.Temperature == 0 {
		config.AI.Temperature = 0.7
	}

	return config.AI, nil
}

// IsAIConfigured checks if AI is properly configured
func IsAIConfigured() bool {
	aiConfig, err := GetAIConfig()
	if err != nil {
		return false
	}
	return aiConfig.APIKey != ""
}

// SetAIConfig updates the AI configuration
func SetAIConfig(aiConfig AIConfig) error {
	config, err := ReadConfig()
	if err != nil {
		config = Config{}
	}

	config.AI = aiConfig

	dataJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	err = os.WriteFile(fileName, dataJSON, fs.FileMode(0644))
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
