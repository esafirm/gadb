package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
)

var fileName = "gadb.setting.json"

// Config that exported to json file
type Config struct {
	PackageName string `json:"packageName"`
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
