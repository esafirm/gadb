package adb

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
)

const ANALYZER = "apkanalyzer"

type CommandReturn struct {
	Output []byte
	Error  error
}

// resolveAnalyzer finds apkanalyzer on PATH first, then in the Android SDK.
// cmdline-tools is not always on PATH, even when ANDROID_HOME is set.
func resolveAnalyzer() (string, error) {
	if path, err := exec.LookPath(ANALYZER); err == nil {
		return path, nil
	}

	for _, root := range []string{os.Getenv("ANDROID_HOME"), os.Getenv("ANDROID_SDK_ROOT")} {
		if root == "" {
			continue
		}
		if path := findAnalyzerInSDK(root); path != "" {
			return path, nil
		}
	}

	return "", fmt.Errorf("%s is not in the path or in ANDROID_HOME/ANDROID_SDK_ROOT", ANALYZER)
}

func findAnalyzerInSDK(root string) string {
	dirs := []string{filepath.Join(root, "cmdline-tools", "latest", "bin")}
	if entries, err := os.ReadDir(filepath.Join(root, "cmdline-tools")); err == nil {
		for _, entry := range entries {
			dirs = append(dirs, filepath.Join(root, "cmdline-tools", entry.Name(), "bin"))
		}
	}
	dirs = append(dirs, filepath.Join(root, "tools", "bin"))

	for _, dir := range dirs {
		for _, name := range []string{ANALYZER, ANALYZER + ".bat"} {
			path := filepath.Join(dir, name)
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				return path
			}
		}
	}
	return ""
}

func Manifest(apkPath string) CommandReturn {
	path, err := resolveAnalyzer()
	if err != nil {
		return CommandReturn{nil, err}
	}

	output, err := exec.Command(path, "manifest", "print", apkPath).CombinedOutput()
	return CommandReturn{output, err}
}

func PackageName(apkPath string) (string, error) {
	path, err := resolveAnalyzer()
	if err != nil {
		return "", err
	}

	output, err := exec.Command(path, "manifest", "print", apkPath).CombinedOutput()
	if err != nil {
		return "", err
	}

	r, _ := regexp.Compile(`package="([^"]*)"`)
	result := r.FindStringSubmatch(string(output))

	if len(result) > 1 {
		return result[1], nil
	}
	return "", fmt.Errorf("package name not found in manifest")
}
