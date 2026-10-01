package adb

import (
	"bufio"
	"bytes"
	"strings"
)

// Device represents a connected Android device
type Device struct {
	Serial      string
	State       string
	Product     string
	Model       string
	DeviceName  string
	TransportID string
}

// GetDeviceList parses `adb devices -l` and returns structured devices
func GetDeviceList() ([]Device, error) {
	cmdRes := RunOnly("adb", "devices", "-l")
	if cmdRes.Error != nil {
		return nil, cmdRes.Error
	}

	var devices []Device
	scanner := bufio.NewScanner(bytes.NewReader(cmdRes.Output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "List of devices") || strings.HasPrefix(line, "* daemon") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		dev := Device{
			Serial: fields[0],
			State:  fields[1],
		}

		for _, f := range fields[2:] {
			parts := strings.SplitN(f, ":", 2)
			if len(parts) == 2 {
				switch parts[0] {
				case "product":
					dev.Product = parts[1]
				case "model":
					dev.Model = parts[1]
				case "device":
					dev.DeviceName = parts[1]
				case "transport_id":
					dev.TransportID = parts[1]
				}
			}
		}

		// Fallback for model if not populated
		if dev.Model == "" {
			dev.Model = dev.Serial
		}

		devices = append(devices, dev)
	}

	return devices, nil
}

// GetPidsForPackage returns running PIDs for a package on a specific device
func GetPidsForPackage(serial string, packageName string) ([]string, error) {
	if packageName == "" {
		return nil, nil
	}

	args := []string{}
	if serial != "" {
		args = append(args, "-s", serial)
	}
	args = append(args, "shell", "pidof", packageName)

	cmdRes := RunOnly("adb", args...)
	if cmdRes.Error != nil {
		// pidof may return non-zero if process not running, or may not exist on older devices
		// Fallback to ps
		return getPidsWithPS(serial, packageName)
	}

	output := strings.TrimSpace(string(cmdRes.Output))
	if output == "" {
		return nil, nil
	}

	return strings.Fields(output), nil
}

func getPidsWithPS(serial string, packageName string) ([]string, error) {
	args := []string{}
	if serial != "" {
		args = append(args, "-s", serial)
	}
	args = append(args, "shell", "ps", "-A")

	cmdRes := RunOnly("adb", args...)
	if cmdRes.Error != nil {
		// Try ps without -A for older Android
		args[len(args)-1] = "ps"
		cmdRes = RunOnly("adb", args...)
		if cmdRes.Error != nil {
			return nil, cmdRes.Error
		}
	}

	var pids []string
	scanner := bufio.NewScanner(bytes.NewReader(cmdRes.Output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, packageName) {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				// field[1] is PID in standard ps output
				pids = append(pids, fields[1])
			}
		}
	}

	return pids, nil
}

// ClearLogcat clears the device logcat buffer
func ClearLogcat(serial string) error {
	args := []string{}
	if serial != "" {
		args = append(args, "-s", serial)
	}
	args = append(args, "logcat", "-c")
	cmdRes := RunOnly("adb", args...)
	return cmdRes.Error
}

// GetDeviceProperty retrieves a system property via getprop
func GetDeviceProperty(serial, prop string) string {
	args := []string{}
	if serial != "" {
		args = append(args, "-s", serial)
	}
	args = append(args, "shell", "getprop", prop)
	cmdRes := RunOnly("adb", args...)
	if cmdRes.Error != nil {
		return ""
	}
	return strings.TrimSpace(string(cmdRes.Output))
}

// GetInstalledPackages returns 3rd party packages installed on the target device
func GetInstalledPackages(serial string) ([]string, error) {
	args := []string{}
	if serial != "" {
		args = append(args, "-s", serial)
	}
	args = append(args, "shell", "pm", "list", "packages", "-3")
	cmdRes := RunOnly("adb", args...)
	if cmdRes.Error != nil {
		return nil, cmdRes.Error
	}
	return splitPackageList(string(cmdRes.Output), "package:"), nil
}
