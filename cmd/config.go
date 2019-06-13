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
	fmt.Println(data)
	dataJSON, err := json.Marshal(data)
	fmt.Println(err)
	fmt.Println(string(dataJSON))

	_ = ioutil.WriteFile(fileName, dataJSON, 0644)

	fmt.Println("Project initialized!")
}

func readConfig() (Config, error) {
	file, _ := ioutil.ReadFile(fileName)
	config := Config{}
	err := json.Unmarshal(file, &config)

	return config, err
}