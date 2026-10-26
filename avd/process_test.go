package avd

import (
	"reflect"
	"testing"

	"github.com/esafirm/gadb/adb"
)

func TestParseProcessLine(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		wantOK     bool
		wantPID    int
		wantAVD    string
		wantSerial string
		wantConfig LaunchConfig
	}{
		{
			name:       "headless emulator",
			line:       "  4321 /Users/me/Library/Android/sdk/emulator/qemu/darwin-aarch64/qemu-system-aarch64 -avd Pixel_7_API_34 -port 5554 -no-window -gpu host -no-audio -memory 4096 -no-snapshot-load",
			wantOK:     true,
			wantPID:    4321,
			wantAVD:    "Pixel_7_API_34",
			wantSerial: "emulator-5554",
			wantConfig: LaunchConfig{
				Known:    true,
				NoWindow: true,
				GPU:      "host",
				NoAudio:  true,
				MemoryMB: 4096,
				ColdBoot: true,
			},
		},
		{
			name:       "windowed emulator with equals arguments",
			line:       "  99 /sdk/emulator/qemu/linux-x86_64/qemu-system-x86_64 -avd=Pixel_3a_API_30 -port=5556 -gpu=swiftshader_indirect -no-boot-anim -lowram -read-only -snapshot gadb_clean",
			wantOK:     true,
			wantPID:    99,
			wantAVD:    "Pixel_3a_API_30",
			wantSerial: "emulator-5556",
			wantConfig: LaunchConfig{
				Known:      true,
				GPU:        "swiftshader_indirect",
				NoBootAnim: true,
				LowRAM:     true,
				ReadOnly:   true,
				Snapshot:   "gadb_clean",
			},
		},
		{
			name:       "windowed emulator with no extra flags",
			line:       "  7 /sdk/emulator/qemu/linux-x86_64/qemu-system-x86_64 -avd Pixel_6 -port 5558",
			wantOK:     true,
			wantPID:    7,
			wantAVD:    "Pixel_6",
			wantSerial: "emulator-5558",
			wantConfig: LaunchConfig{Known: true},
		},
		{
			name:       "emulator launcher style with positional avd",
			line:       "  48756 /Users/me/Library/Android/sdk/emulator/qemu/darwin-aarch64/qemu-system-aarch64-headless @a10 -no-window -no-audio -no-snapshot-load -no-boot-anim",
			wantOK:     true,
			wantPID:    48756,
			wantAVD:    "a10",
			wantSerial: "",
			wantConfig: LaunchConfig{
				Known:      true,
				NoWindow:   true,
				NoAudio:    true,
				NoBootAnim: true,
				ColdBoot:   true,
			},
		},
		{
			name:   "not an emulator",
			line:   "  99 /usr/bin/something -avd Pixel_3a_API_30 -port 5556",
			wantOK: false,
		},
		{
			name:   "crash reporter",
			line:   "  99 /sdk/emulator/crashpad_handler --database=/tmp/crash -avd Pixel_3a_API_30",
			wantOK: false,
		},
		{
			name:   "no emulator arguments",
			line:   "  99 /sdk/emulator/qemu/linux-x86_64/qemu-system-x86_64 -help",
			wantOK: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := parseProcessLine(test.line)
			if ok != test.wantOK {
				t.Fatalf("parseProcessLine() ok = %v, want %v (got %#v)", ok, test.wantOK, got)
			}
			if !test.wantOK {
				return
			}
			if got.PID != test.wantPID || got.AVD != test.wantAVD || got.Serial != test.wantSerial {
				t.Fatalf("parseProcessLine() = %#v, want pid %d avd %q serial %q",
					got, test.wantPID, test.wantAVD, test.wantSerial)
			}
			if !reflect.DeepEqual(got.Config, test.wantConfig) {
				t.Fatalf("config = %#v, want %#v", got.Config, test.wantConfig)
			}
		})
	}
}

func TestLaunchConfigSummary(t *testing.T) {
	tests := []struct {
		name   string
		config LaunchConfig
		want   string
	}{
		{name: "unknown", config: LaunchConfig{}, want: "unknown"},
		{name: "windowed", config: LaunchConfig{Known: true}, want: "window"},
		{
			name: "full",
			config: LaunchConfig{
				Known:      true,
				NoWindow:   true,
				GPU:        "host",
				NoAudio:    true,
				NoBootAnim: true,
				LowRAM:     true,
				MemoryMB:   4096,
				ReadOnly:   true,
				Snapshot:   "gadb_clean",
				ColdBoot:   true,
			},
			want: "no-window, gpu=host, no-audio, no-boot-anim, lowram, 4096 MB, read-only, snapshot=gadb_clean, cold-boot",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.config.Summary(); got != test.want {
				t.Fatalf("Summary() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestMergeRunningEmulators(t *testing.T) {
	devices := []adb.Device{
		{Serial: "emulator-5556", State: "offline"},
		{Serial: "emulator-5554", State: "device"},
		{Serial: "R58M12345", State: "device"},
	}
	processes := []hostEmulator{
		{
			PID:    100,
			AVD:    "Pixel_7_API_34",
			Serial: "emulator-5554",
			Config: LaunchConfig{Known: true, NoWindow: true, GPU: "host"},
		},
		{
			PID:    200,
			AVD:    "Pixel_3a_API_30",
			Config: LaunchConfig{Known: true, GPU: "swiftshader_indirect"},
		},
		{
			PID:    300,
			AVD:    "Orphan_API_29",
			Serial: "emulator-5558",
			Config: LaunchConfig{Known: true, NoWindow: true},
		},
	}

	var resolved []string
	resolveName := func(serial string) string {
		resolved = append(resolved, serial)
		if serial == "emulator-5556" {
			return "Pixel_3a_API_30"
		}
		return ""
	}

	got := mergeRunningEmulators(devices, processes, resolveName)

	if !reflect.DeepEqual(resolved, []string{"emulator-5556"}) {
		t.Fatalf("resolved serials = %#v, want only emulator-5556", resolved)
	}

	want := []RunningEmulator{
		{
			AVD:    "Pixel_7_API_34",
			Serial: "emulator-5554",
			State:  "device",
			PID:    100,
			Config: LaunchConfig{Known: true, NoWindow: true, GPU: "host"},
		},
		{
			AVD:    "Pixel_3a_API_30",
			Serial: "emulator-5556",
			State:  "offline",
			PID:    200,
			Config: LaunchConfig{Known: true, GPU: "swiftshader_indirect"},
		},
		{
			AVD:    "Orphan_API_29",
			Serial: "emulator-5558",
			State:  "starting",
			PID:    300,
			Config: LaunchConfig{Known: true, NoWindow: true},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mergeRunningEmulators() =\n%#v\nwant\n%#v", got, want)
	}
}

func TestMergeRunningEmulatorsWithoutHostProcesses(t *testing.T) {
	devices := []adb.Device{
		{Serial: "emulator-5554", State: "device"},
		{Serial: "R58M12345", State: "device"},
	}
	resolveName := func(string) string { return "Pixel_7_API_34" }

	got := mergeRunningEmulators(devices, nil, resolveName)
	want := []RunningEmulator{
		{AVD: "Pixel_7_API_34", Serial: "emulator-5554", State: "device"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mergeRunningEmulators() = %#v, want %#v", got, want)
	}
}
