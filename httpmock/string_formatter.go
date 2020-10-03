package httpmock

import (
	"strings"
	"unicode"

	"github.com/ghodss/yaml"
)

func SpaceMap(str string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, str)
}

func YamlToJsonString(data []byte) string {
	data, err := yaml.YAMLToJSON(data)
	CheckErr(err)
	return string(data)
}
