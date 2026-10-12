package adb

import "testing"

func TestStayAwakeCommands(t *testing.T) {
	// Verify helper functions don't panic
	res := StayAwake(false)
	if res.Error != nil {
		t.Logf("StayAwake(false) error (acceptable if device not connected): %v", res.Error)
	}

	res = AlwaysOnWhenConnectedToPower(false)
	if res.Error != nil {
		t.Logf("AlwaysOnWhenConnectedToPower(false) error: %v", res.Error)
	}
}
