package adb

import (
	"fmt"
	"os/exec"
)

func runWithPrint(name string, arg ...string) CommandReturn {
	output, err := exec.Command(name, arg...).CombinedOutput()
	if err == nil {
		fmt.Println(string(output))
	}
	return CommandReturn{output, err}
}

func runOnly(name string, arg ...string) CommandReturn {
	output, err := exec.Command(name, arg...).CombinedOutput()
	return CommandReturn{output, err}
}
