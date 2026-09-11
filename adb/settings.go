package adb

// SettingsGet reads a settings value: adb shell settings get <namespace> <key>
func SettingsGet(namespace string, key string) CommandReturn {
	return runOnly("adb", "shell", "settings", "get", namespace, key)
}

// SettingsPut writes a settings value: adb shell settings put <namespace> <key> <value>
func SettingsPut(namespace string, key string, value string) CommandReturn {
	return runWithPrint("adb", "shell", "settings", "put", namespace, key, value)
}
