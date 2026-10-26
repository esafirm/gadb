package cmd

import (
	"testing"

	"github.com/esafirm/gadb/avd"
)

func TestFindRunningEmulator(t *testing.T) {
	running := []avd.RunningEmulator{
		{AVD: "Pixel_7_API_34", Serial: "emulator-5554"},
		{AVD: "Pixel_3a_API_30", Serial: "emulator-5556"},
	}

	tests := []struct {
		query string
		want  string
	}{
		{query: "emulator-5554", want: "emulator-5554"},
		{query: "pixel_3a_api_30", want: "emulator-5556"},
	}

	for _, test := range tests {
		got, err := findRunningEmulator(running, test.query)
		if err != nil {
			t.Fatalf("findRunningEmulator(%q) error: %v", test.query, err)
		}
		if got.Serial != test.want {
			t.Fatalf("findRunningEmulator(%q) = %s, want %s", test.query, got.Serial, test.want)
		}
	}

	if _, err := findRunningEmulator(running, "Pixel_9"); err == nil {
		t.Fatal("findRunningEmulator() found an emulator that is not running")
	}
}
