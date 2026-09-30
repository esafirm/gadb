package adb

import (
	"testing"
)

func TestParseDeviceListOutput(t *testing.T) {
	// Test that GetDeviceList doesn't panic when adb is available
	devices, err := GetDeviceList()
	if err != nil {
		t.Logf("GetDeviceList error (acceptable if adb not installed): %v", err)
		return
	}
	t.Logf("Found %d devices", len(devices))
	for _, d := range devices {
		t.Logf("Device: %+v", d)
	}
}
