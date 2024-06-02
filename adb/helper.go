package adb

import "strings"

// Split and format raw package list from ADB command return
func splitPackageList(rawString string, trimmedPrefix string) []string {
	packages := strings.Split(strings.TrimSpace(rawString), "\n")

	// remove "package:" prefix
	for i, p := range packages {
		packages[i] = strings.TrimPrefix(p, trimmedPrefix)
	}

	return packages
}

func splitAvdList(rawString string) []string {
	avds := strings.Split(strings.TrimSpace(rawString), "\n")

	returnedAvds := []string{}

	for _, avd := range avds {
		if !strings.Contains(avd, "INFO") && !strings.Contains(avd, "|") {
			returnedAvds = append(returnedAvds, avd)
		}
	}

	return returnedAvds
}
