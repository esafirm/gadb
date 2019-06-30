package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	file, _ := ioutil.ReadFile(fileName)
	config := Config{}
	err := json.Unmarshal(file, &config)

	return config, err
}

func GetPackageNameFromArgs(args []string) (string, error) {
	if len(args) == 0 {
		config, err := readConfig()
		if err != nil {
			return "", err
		}
		return "", config.PackageName
	}
	return "", args[0]
}
