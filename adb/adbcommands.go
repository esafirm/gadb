package adb

import (
	"fmt"
	"os/exec"
)

func runWithPrint(name string, arg ...string) ([]byte, error) {
	output, error := exec.Command(name, arg...).CombinedOutput()
	if error == nil {
		fmt.Println(string(output))
	}
	return output, error
}

func Uninstall(packageName string) ([]byte, error) {
	return runWithPrint("adb", "uninstall", packageName)
}

func Install(apkPath string) ([]byte, error) {
	return runWithPrint("adb", "install", apkPath)
}
