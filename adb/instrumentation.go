package adb

import (
	"fmt"
	"strings"
)

type InstrumentationsCommandReturn struct {
	PackageList []InstrumentationPackage
	Error       error
}

// A pair of the instrumentation package and its target
type InstrumentationPackage struct {
	PackageName string
	Target      string
}

// List down installed instrumentation packages in a formatted way
func ListPackagesInstrumentationFormatted() InstrumentationsCommandReturn {
	commandReturn := runOnly("adb", "shell", "pm", "list", "instrumentation")
	if commandReturn.Error != nil {
		return InstrumentationsCommandReturn{Error: commandReturn.Error}
	}
	return InstrumentationsCommandReturn{PackageList: splitToInstrumentationPackages(
		string(commandReturn.Output),
		"instrumentation:",
	)}
}

// Start instrumentation test
func StartInstrumentation(instrumentationPackage string, args ...string) CommandReturn {
	args = append([]string{"shell", "am", "instrument", "-w", instrumentationPackage}, args...)
	fmt.Println("Running instrumentation test with args:", args)
	return runWithPrint("adb", args...)
}

// Create debug extra for instrumentation command
func ExtraDebug() []string {
	return []string{"-e", "debug", "true"}
}

func ExtraFilter(key string, value string) []string {
	return []string{"-e", key, value}
}

func splitToInstrumentationPackages(rawString string, trimmedPrefix string) []InstrumentationPackage {
	packages := strings.Split(strings.TrimSpace(rawString), "\n")

	// remove "package:" prefix
	for i, p := range packages {
		packages[i] = strings.TrimPrefix(p, trimmedPrefix)
	}

	// Create new slices with certain length
	result := make([]InstrumentationPackage, len(packages))
	for index, p := range packages {
		slices := strings.Split(strings.TrimSpace(p), " ")
		instrumentationPackage := InstrumentationPackage{
			PackageName: strings.TrimSpace(slices[0]),
			Target:      strings.TrimPrefix(strings.Trim(slices[1], "()"), "target="),
		}
		result[index] = instrumentationPackage
	}

	return result
}

func (s InstrumentationPackage) String() string {
	return fmt.Sprintf("%s - %s", s.PackageName, s.Target)
}
