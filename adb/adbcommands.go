package adb

import (
	"fmt"
	"os/exec"
)

// CommandReturn have output and error of the command
type CommandReturn struct {
	Output []byte
	Error  error
}

func runWithPrint(name string, arg ...string) CommandReturn {
	output, error := exec.Command(name, arg...).CombinedOutput()
	if error == nil {
		fmt.Println(string(output))
	}
	return CommandReturn{output, error}
}

// Uninstall application with following package name
func Uninstall(packageName string) CommandReturn {
	return runWithPrint("adb", "uninstall", packageName)
}

// Install APK in the path
func Install(apkPath string) CommandReturn {
	return runWithPrint("adb", "install", apkPath)
}

// Debug enable waiting debug mode in android
func Debug(isPersistent bool, packageName string) CommandReturn {
	if isPersistent {
		return runWithPrint("adb", "shell", "am", "set-debug-app", "-w", packageName)
	} else {
		return runWithPrint("adb", "shell", "am", "set-debug-app", "-w", "--persistent", packageName)
	}
}

// DebugCancel cancel the debug set on device
func DebugCancel(packageName string) CommandReturn {
	return runWithPrint("adb", "shell", "am", "clear-debug-app", packageName)
}

// Stop the assigned application
func Stop(packageName string) CommandReturn {
	return runWithPrint("adb", "shell", "am", "force-stop", packageName)
}

// Start the assigned application
func Start(packageName string) CommandReturn {
	return runWithPrint("adb", "shell", "am", "start", packageName)
}

// Restart do stop and then start the assigned application
func Restart(packageName string) CommandReturn {
	stopResult := Stop(packageName)
	if stopResult.Error != nil {
		return stopResult
	}
	return Start(packageName)
}
