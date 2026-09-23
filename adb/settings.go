package adb

import "strings"

// SettingsGet reads a settings value: adb shell settings get <namespace> <key>
func SettingsGet(namespace string, key string) CommandReturn {
	return runOnly("adb", "shell", "settings", "get", namespace, key)
}

// SettingsPut writes a settings value: adb shell settings put <namespace> <key> <value>
func SettingsPut(namespace string, key string, value string) CommandReturn {
	return runOnly("adb", "shell", "settings", "put", namespace, key, value)
}

// AirplaneMode sets the airplane mode state: adb shell settings put global airplane_mode_on <0|1>
func AirplaneMode(enable bool) CommandReturn {
	val := "0"
	if enable {
		val = "1"
	}
	return SettingsPut("global", "airplane_mode_on", val)
}

// AirplaneModeToggle toggles the device airplane mode state
func AirplaneModeToggle() CommandReturn {
	res := SettingsGet("global", "airplane_mode_on")
	if res.Error != nil {
		return res
	}
	val := strings.TrimSpace(string(res.Output))
	if val == "1" {
		return AirplaneMode(false)
	}
	return AirplaneMode(true)
}
