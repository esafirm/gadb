package adb

import (
	"fmt"
	"os/exec"
	"regexp"
)

const ANALYZER = "apkanalyzer"

type CommandReturn struct {
	Output []byte
	Error  error
}

func checkInPath() error {
	_, err := exec.LookPath(ANALYZER)
	if err != nil {
		return fmt.Errorf("%s is not in the path", ANALYZER)
	}
	return nil
}

func Manifest(apkPath string) CommandReturn {
	if err := checkInPath(); err != nil {
		return CommandReturn{nil, err}
	}

	output, err := exec.Command(ANALYZER, "manifest", "print", apkPath).CombinedOutput()
	return CommandReturn{output, err}
}

func PackageName(apkPath string) (string, error) {
	if err := checkInPath(); err != nil {
		return "", err
	}

	output, err := exec.Command(ANALYZER, "manifest", "print", apkPath).CombinedOutput()
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
