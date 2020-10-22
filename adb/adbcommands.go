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

func runOnly(name string, arg ...string) CommandReturn {
	output, err := exec.Command(name, arg...).CombinedOutput()
	return CommandReturn{output, err}
}

// Uninstall application with following package name
func Uninstall(packageName string) CommandReturn {
	return runWithPrint("adb", "uninstall", packageName)
}

// Install APK in the path
func Install(apkPath string) CommandReturn {
	return runWithPrint("adb", "install", apkPath)
}

// InstallTo install APK to the specific device
func InstallTo(deviceID string, apkPath string) CommandReturn {
	return runWithPrint("adb", "-s", deviceID, "install", apkPath)
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

// ClearData clear all application data in device storage
func ClearData(packageName string) CommandReturn {
	return runWithPrint("adb", "shell", "pm", "clear", packageName)
}

// ConnectedDevices return all devices that connected to ADB
func ConnectedDevices() CommandReturn {
	return runWithPrint("adb", "devices")
}

// AvdList print all available AVD(s)
func AvdList() CommandReturn {
	return runOnly("emulator", "-list-avds")
}

// AvdRun start the AVD with the passed name
func AvdRun(avdName string) {
	cmd := exec.Command("emulator", "@"+avdName)
	cmd.Start()
}

// Wipe AVD emulator data
func AvdWipe(avdName string) {
	runWithPrint("emulator", "@"+avdName, "-wipe-data{")
}

func DumpSys(moreCommand string) CommandReturn {
	return runWithPrint("adb shell dumpsys " + moreCommand)
}

// Forward forward tcp port
func Forward(port string) CommandReturn {
	return runWithPrint("adb", "forward", "tcp:"+port, "tcp:"+port)
}

// ForwardList return all forward list
func ForwardList() CommandReturn {
	return runOnly("adb", "forward", "--list")
}
