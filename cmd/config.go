package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
)

var fileName = "gadb.setting.json"

// Config that exported to json file
type Config struct {
	PackageName string `json:"packageName"`
}

func writeConfig(data Config) {
	dataJSON, err := json.Marshal(data)
	_ = ioutil.WriteFile(fileName, dataJSON, 0644)

	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("Project initialized!")
	}
}

func readConfig() (Config, error) {
	if !isConfigExist() {
		return Config{}, errors.New("Config file doesn't exist. Create one using gadb init")
	}
	file, _ := ioutil.ReadFile(fileName)
	config := Config{}
	err := json.Unmarshal(file, &config)

	return config, err
}

func isConfigExist() bool {
	_, err := os.Stat(fileName)
	return !os.IsNotExist(err)
}

// GetPackageNameFromArgs take the first index of args as package or get the packagename
// from config file
func GetPackageNameFromArgs(args []string) (string, error) {
	if len(args) == 0 {
		config, err := readConfig()
		if err != nil {
			return "", err
		}
		return config.PackageName, nil
	}
	return args[0], nil
}
