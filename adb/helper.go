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
