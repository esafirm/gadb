package adb

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
)

const ANALYZER = "apkanalyzer"

type CommandReturn struct {
	Output []byte
	Error  error
}

func checkInPath() {
	_, err := exec.LookPath(ANALYZER)
	if err != nil {
		fmt.Println(ANALYZER + " is not in the path")
		os.Exit(1)
	}
}

func Manifest(apkPath string) CommandReturn {
	checkInPath()

	output, err := exec.Command(ANALYZER, "manifest", "print", apkPath).CombinedOutput()
	return CommandReturn{output, err}
}

func PackageName(apkPath string) (string, error) {
	checkInPath()

	output, err := exec.Command(ANALYZER, "manifest", "print", apkPath).CombinedOutput()
	if err != nil {
		return "", err
	}

	r, _ := regexp.Compile("package=\"(.*)\"")
	result := r.FindStringSubmatch(string(output))

	return result[1], nil
}
