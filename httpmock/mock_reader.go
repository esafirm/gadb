package httpmock

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"strings"
)

type MockContext struct {
	Path   string      `json:"path" yaml:"path"`
	Method string      `json:"method" yaml:"method"`
	Code   string      `json:"code" yaml:"code"`
	Body   interface{} `json:"body" yaml:"body"`
	Header interface{} `json:"header" yaml:"header"`
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
	contents := getContents(files)
	contexts := getContexts(contents)

	println(contents[0])

	return contexts, contents
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

func getContents(files []string) []string {
	var contents []string
	for _, f := range files {
		ext := strings.ToLower(filepath.Ext(f))
		file, err := ioutil.ReadFile(f)
		CheckErr(err)

		var content string
		if ext == ".yml" || ext == ".yaml" {
			content = SpaceMap(YamlToJsonString(file))
		} else {
			content = SpaceMap(string(file))
		}
		contents = append(contents, content)
	}
	return contents
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
