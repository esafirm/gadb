package adb

import "strings"

// Split and format raw package list from ADB command return
func splitPackageList(rawString string, trimmedPrefix string) []string {
	clean := strings.ReplaceAll(rawString, "\r", "")
	lines := strings.Split(clean, "\n")

	var packages []string
	for _, p := range lines {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		p = strings.TrimPrefix(p, trimmedPrefix)
		p = strings.TrimSpace(p)
		if p != "" {
			packages = append(packages, p)
		}
	}

	return packages
}

func splitAvdList(rawString string) []string {
	clean := strings.ReplaceAll(rawString, "\r", "")
	avds := strings.Split(clean, "\n")

	returnedAvds := []string{}

	for _, avd := range avds {
		avd = strings.TrimSpace(avd)
		if avd == "" {
			continue
		}
		if !strings.Contains(avd, "INFO") && !strings.Contains(avd, "|") {
			returnedAvds = append(returnedAvds, avd)
		}
	}

	return returnedAvds
}
