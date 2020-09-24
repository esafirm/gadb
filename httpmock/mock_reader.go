package httpmock

import (
	"encoding/json"
	"io/ioutil"
	"strings"
)

type MockContext struct {
	Path   string `json:"path"`
	Body   string `json:"mock"`
	Method string `json:"method"`
}

func ReadMockFile(path string) ([]MockContext, []string) {
	files := []string{path}
	return read(files)
}

func ReadMockFiles(dir string, prefix *string) ([]MockContext, []string) {
	files := listFiles(dir, *prefix)
	return read(files)
}

func read(files []string) ([]MockContext, []string) {
	jsons := getJsons(files)
	contexts := getContexts(jsons)

	return contexts, jsons
}

func getContexts(jsons []string) []MockContext {
	var contexts []MockContext
	for _, jsonString := range jsons {
		var context MockContext
		json.Unmarshal([]byte(jsonString), &context)
		contexts = append(contexts, context)
	}
	return contexts
}

func getJsons(files []string) []string {
	var jsons []string
	for _, f := range files {
		jsonFile, err := ioutil.ReadFile(f)
		CheckErr(err)

		jsons = append(jsons, SpaceMap(string(jsonFile)))
	}
	return jsons
}

func listFiles(dir string, prefix string) []string {
	var files []string
	fileInfos, err := ioutil.ReadDir(dir)
	CheckErr(err)

	for _, info := range fileInfos {
		if strings.HasPrefix(info.Name(), prefix) {
			files = append(files, info.Name())
		}
	}
	return files
}
