package avd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/esafirm/gadb/adb"
)

// LaunchConfig is the part of an emulator command line that changes how the
// emulator runs. It is read from the live qemu process, so it reflects the
// flags actually used at launch rather than what config.ini says.
type LaunchConfig struct {
	Known      bool
	NoWindow   bool
	GPU        string
	NoAudio    bool
	NoBootAnim bool
	LowRAM     bool
	MemoryMB   int
	ReadOnly   bool
	Snapshot   string
	ColdBoot   bool
}

// Summary renders the config as a compact, human-readable list.
func (c LaunchConfig) Summary() string {
	if !c.Known {
		return "unknown"
	}

	window := "window"
	if c.NoWindow {
		window = "no-window"
	}
	parts := []string{window}
	if c.GPU != "" {
		parts = append(parts, "gpu="+c.GPU)
	}
	if c.NoAudio {
		parts = append(parts, "no-audio")
	}
	if c.NoBootAnim {
		parts = append(parts, "no-boot-anim")
	}
	if c.LowRAM {
		parts = append(parts, "lowram")
	}
	if c.MemoryMB > 0 {
		parts = append(parts, fmt.Sprintf("%d MB", c.MemoryMB))
	}
	if c.ReadOnly {
		parts = append(parts, "read-only")
	}
	if c.Snapshot != "" {
		parts = append(parts, "snapshot="+c.Snapshot)
	}
	if c.ColdBoot {
		parts = append(parts, "cold-boot")
	}
	return strings.Join(parts, ", ")
}

// RunningEmulator is a live emulator found through adb, the host process list,
// or both. State is the adb state ("device", "offline", ...) and is empty for
// a process adb does not know about yet.
type RunningEmulator struct {
	AVD    string
	Serial string
	State  string
	PID    int
	Config LaunchConfig
}

// Label names the emulator for prompts and result lines.
func (e RunningEmulator) Label() string {
	switch {
	case e.AVD != "" && e.AVD != "unknown" && e.Serial != "":
		return fmt.Sprintf("%s (%s)", e.AVD, e.Serial)
	case e.Serial != "":
		return e.Serial
	case e.AVD != "" && e.AVD != "unknown":
		return fmt.Sprintf("%s (pid %d)", e.AVD, e.PID)
	default:
		return fmt.Sprintf("pid %d", e.PID)
	}
}

// Stop shuts the emulator down. It asks adb to shut down an online emulator,
// then falls back to the host process so offline or stuck emulators still die.
func (e RunningEmulator) Stop(timeout time.Duration) error {
	// An emulator adb can see is stopped through its console; offline ones get
	// a short attempt, since the console can still answer while the guest is
	// stuck. A process adb has not registered yet is killed directly.
	if e.Serial != "" && e.State != "starting" {
		wait := timeout
		if e.State != "device" {
			wait = 5 * time.Second
		}
		if err := StopSerial(e.Serial, wait); err == nil {
			return nil
		}
	}

	if e.PID <= 0 {
		if e.Serial != "" && !serialVisible(e.Serial) {
			return nil
		}
		return fmt.Errorf("%s did not stop after %s", e.Label(), timeout)
	}

	process, err := os.FindProcess(e.PID)
	if err == nil {
		err = process.Kill()
	}
	if err != nil && !errors.Is(err, os.ErrProcessDone) {
		return fmt.Errorf("cannot kill %s: %w", e.Label(), err)
	}
	if !waitProcessGone(e.PID, 5*time.Second) {
		return fmt.Errorf("%s (pid %d) is still running", e.Label(), e.PID)
	}
	return nil
}

// RunningEmulators lists every live emulator, including ones that are offline
// or not visible to adb yet, with the launch flags read from the host process.
func RunningEmulators() ([]RunningEmulator, error) {
	devices, err := adb.GetDeviceList()
	if err != nil {
		return nil, err
	}
	return mergeRunningEmulators(devices, listEmulatorProcesses(), emulatorAVDName), nil
}

func mergeRunningEmulators(devices []adb.Device, processes []hostEmulator, resolveName func(string) string) []RunningEmulator {
	bySerial := make(map[string]hostEmulator, len(processes))
	byAVD := make(map[string]hostEmulator, len(processes))
	for _, process := range processes {
		if process.Serial != "" {
			bySerial[process.Serial] = process
		}
		if process.AVD != "" {
			byAVD[strings.ToLower(process.AVD)] = process
		}
	}

	used := make(map[int]bool, len(processes))
	running := make([]RunningEmulator, 0, len(devices)+len(processes))

	for _, device := range devices {
		if !strings.HasPrefix(device.Serial, "emulator-") {
			continue
		}

		entry := RunningEmulator{Serial: device.Serial, State: device.State}
		process, found := bySerial[device.Serial]
		if !found {
			entry.AVD = resolveName(device.Serial)
			if candidate, ok := byAVD[strings.ToLower(entry.AVD)]; ok && entry.AVD != "" {
				process, found = candidate, true
			}
		}
		if found {
			if entry.AVD == "" {
				entry.AVD = process.AVD
			}
			entry.PID = process.PID
			entry.Config = process.Config
			used[process.PID] = true
		}
		if entry.AVD == "" {
			entry.AVD = "unknown"
		}
		running = append(running, entry)
	}

	for _, process := range processes {
		if used[process.PID] {
			continue
		}
		running = append(running, RunningEmulator{
			AVD:    process.AVD,
			Serial: process.Serial,
			State:  "starting",
			PID:    process.PID,
			Config: process.Config,
		})
	}

	sort.SliceStable(running, func(i, j int) bool {
		return emulatorPort(running[i].Serial) < emulatorPort(running[j].Serial)
	})
	return running
}

type hostEmulator struct {
	PID    int
	AVD    string
	Serial string
	Config LaunchConfig
}

// listEmulatorProcesses scans the host process list. It returns nothing when
// ps is unavailable, which is the case on Windows; adb still lists emulators
// there, just without their launch flags.
func listEmulatorProcesses() []hostEmulator {
	output, err := exec.Command("ps", "-axo", "pid=,command=").Output()
	if err != nil {
		return nil
	}

	var processes []hostEmulator
	for _, line := range strings.Split(string(output), "\n") {
		if process, ok := parseProcessLine(line); ok {
			processes = append(processes, process)
		}
	}
	return processes
}

func parseProcessLine(line string) (hostEmulator, bool) {
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return hostEmulator{}, false
	}

	pid, err := strconv.Atoi(fields[0])
	if err != nil {
		return hostEmulator{}, false
	}
	if !isEmulatorBinary(fields[1]) {
		return hostEmulator{}, false
	}

	name, port, config := parseLaunchArgs(fields[2:])
	if name == "" && port == 0 {
		return hostEmulator{}, false
	}
	config.Known = true

	serial := ""
	if port >= 5554 {
		serial = "emulator-" + strconv.Itoa(port)
	}
	return hostEmulator{PID: pid, AVD: name, Serial: serial, Config: config}, true
}

func isEmulatorBinary(path string) bool {
	return strings.HasPrefix(strings.ToLower(filepath.Base(path)), "qemu-system")
}

func parseLaunchArgs(args []string) (string, int, LaunchConfig) {
	config := LaunchConfig{}
	name := ""
	port := 0
	for i := 0; i < len(args); i++ {
		arg := args[i]
		key, value, hasValue := strings.Cut(arg, "=")
		next := func() string {
			if hasValue {
				return value
			}
			if i+1 < len(args) {
				i++
				return args[i]
			}
			return ""
		}

		switch key {
		case "-avd":
			name = next()
		case "-port":
			port, _ = strconv.Atoi(next())
		case "-gpu":
			config.GPU = next()
		case "-memory":
			config.MemoryMB, _ = strconv.Atoi(next())
		case "-snapshot":
			config.Snapshot = next()
		case "-no-window":
			config.NoWindow = true
		case "-no-audio":
			config.NoAudio = true
		case "-no-boot-anim":
			config.NoBootAnim = true
		case "-lowram":
			config.LowRAM = true
		case "-read-only":
			config.ReadOnly = true
		case "-no-snapshot", "-no-snapshot-load":
			config.ColdBoot = true
		}
		if strings.HasPrefix(arg, "@") && name == "" {
			name = strings.TrimPrefix(arg, "@")
		}
	}
	return name, port, config
}

func emulatorPort(serial string) int {
	if !strings.HasPrefix(serial, "emulator-") {
		return 1 << 30
	}
	port, err := strconv.Atoi(strings.TrimPrefix(serial, "emulator-"))
	if err != nil {
		return 1 << 30
	}
	return port
}

func waitProcessGone(pid int, timeout time.Duration) bool {
	if runtime.GOOS == "windows" {
		return true
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !processRunning(pid) {
			return true
		}
		time.Sleep(250 * time.Millisecond)
	}
	return !processRunning(pid)
}

func processRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}
