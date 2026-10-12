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

// StayAwake sets the stay awake while charging state (7 = all power sources, 0 = disabled):
// adb shell settings put global stay_on_while_plugged_in <0|7>
func StayAwake(enable bool) CommandReturn {
	val := "0"
	if enable {
		val = "7"
	}
	return SettingsPut("global", "stay_on_while_plugged_in", val)
}

// StayAwakeToggle toggles the device stay awake while charging state
func StayAwakeToggle() CommandReturn {
	res := SettingsGet("global", "stay_on_while_plugged_in")
	if res.Error != nil {
		return res
	}
	val := strings.TrimSpace(string(res.Output))
	if val != "0" && val != "" && val != "null" {
		return StayAwake(false)
	}
	return StayAwake(true)
}

// AlwaysOnWhenConnectedToPower sets the screen always on when connected to power state:
// alias for StayAwake
func AlwaysOnWhenConnectedToPower(enable bool) CommandReturn {
	return StayAwake(enable)
}

// AlwaysOnWhenConnectedToPowerToggle toggles the screen always on when connected to power state:
// alias for StayAwakeToggle
func AlwaysOnWhenConnectedToPowerToggle() CommandReturn {
	return StayAwakeToggle()
}
