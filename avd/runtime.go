package avd

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/esafirm/gadb/adb"
)

// runningEmulator maps an AVD name to its live emulator serial. It is a
// variable so tests can stay hermetic.
var runningEmulator = findRunningEmulator

// Runtime is what a running emulator actually reports, as opposed to what
// config.ini says. Launch flags can override the config.
type Runtime struct {
	Serial     string
	MemTotalMB int
	Renderer   string // "hardware", "software" or "unknown"
	LowRAM     bool
}

// RunningAVD is a live emulator mapped to its AVD name.
type RunningAVD struct {
	AVD    string
	Serial string
}

// RunningSerial returns the live serial for an AVD.
func RunningSerial(name string) (string, bool) {
	return runningEmulator(name)
}

// NameForSerial resolves an emulator serial to its AVD name.
func NameForSerial(serial string) (string, error) {
	name := emulatorAVDName(serial)
	if name == "" {
		return "", fmt.Errorf("cannot resolve %s to an AVD", serial)
	}
	return name, nil
}

// RunningAVDs lists running emulators that can be mapped back to an AVD.
func RunningAVDs() ([]RunningAVD, error) {
	devices, err := adb.GetDeviceList()
	if err != nil {
		return nil, err
	}

	running := make([]RunningAVD, 0, len(devices))
	for _, device := range devices {
		if device.State != "device" || !strings.HasPrefix(device.Serial, "emulator-") {
			continue
		}
		name := emulatorAVDName(device.Serial)
		if name == "" {
			continue
		}
		running = append(running, RunningAVD{AVD: name, Serial: device.Serial})
	}

	sort.Slice(running, func(i, j int) bool {
		return running[i].AVD < running[j].AVD
	})
	return running, nil
}

func findRunningEmulator(avdName string) (string, bool) {
	running, err := RunningAVDs()
	if err != nil {
		return "", false
	}
	for _, candidate := range running {
		if strings.EqualFold(candidate.AVD, avdName) {
			return candidate.Serial, true
		}
	}
	return "", false
}

func emulatorAVDName(serial string) string {
	result := adb.RunOnlyWithTimeout(10*time.Second, "adb", "-s", serial, "emu", "avd", "name")
	if result.Error != nil {
		return ""
	}
	for _, line := range strings.Split(string(result.Output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "OK" {
			continue
		}
		return line
	}
	return ""
}

func probeRuntime(serial string) *Runtime {
	runtime := &Runtime{Serial: serial}

	if result := adb.ShellCommand(serial, 10*time.Second, "grep MemTotal /proc/meminfo"); result.Error == nil {
		runtime.MemTotalMB = parseMemTotalMB(string(result.Output))
	}
	if result := adb.ShellCommand(serial, 20*time.Second, "dumpsys SurfaceFlinger 2>/dev/null | grep 'GLES:' | head -1"); result.Error == nil {
		runtime.Renderer = classifyRenderer(string(result.Output))
	}
	if result := adb.ShellCommand(serial, 10*time.Second, "getprop ro.config.low_ram"); result.Error == nil {
		runtime.LowRAM = strings.TrimSpace(string(result.Output)) == "true"
	}

	return runtime
}

func parseMemTotalMB(output string) int {
	fields := strings.Fields(output)
	for i, field := range fields {
		if field != "kB" || i == 0 {
			continue
		}
		kb, err := strconv.Atoi(fields[i-1])
		if err == nil {
			return kb / 1024
		}
	}
	return 0
}

func classifyRenderer(output string) string {
	lower := strings.ToLower(output)
	switch {
	case strings.Contains(lower, "swiftshader"),
		strings.Contains(lower, "lavapipe"),
		strings.Contains(lower, "llvmpipe"),
		strings.Contains(lower, "software"):
		return "software"
	case strings.Contains(lower, "angle"),
		strings.Contains(lower, "metal"),
		strings.Contains(lower, "adreno"),
		strings.Contains(lower, "mali"),
		strings.Contains(lower, "apple"):
		return "hardware"
	default:
		return "unknown"
	}
}
