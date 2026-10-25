package avd

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/esafirm/gadb/adb"
)

// FindEmulator locates the emulator binary on PATH, then in the SDK.
func FindEmulator() string {
	names := []string{"emulator"}
	if runtime.GOOS == "windows" {
		names = []string{"emulator.exe", "emulator"}
	}
	for _, name := range names {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}

	sdk := SDKDir()
	if sdk == "" {
		return ""
	}
	for _, name := range names {
		path := filepath.Join(sdk, "emulator", name)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}
	return ""
}

// FreeEmulatorPort returns an even port the emulator can bind.
func FreeEmulatorPort() (int, error) {
	for port := 5554; port <= 5682; port += 2 {
		listener, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
		if err != nil {
			continue
		}
		listener.Close()
		return port, nil
	}
	return 0, errors.New("no free emulator port between 5554 and 5682")
}

// LaunchOptions controls a slim emulator launch.
type LaunchOptions struct {
	RAM        int
	GPU        string
	Headless   bool
	LowRAM     bool
	NoAudio    bool
	NoBootAnim bool
	Snapshot   string // golden snapshot name, empty for a cold boot
}

// EmulatorProcess is a detached emulator with its output in LogPath.
type EmulatorProcess struct {
	AVD     AVD
	Serial  string
	Port    int
	LogPath string
	Cmd     *exec.Cmd

	done    chan struct{}
	mu      sync.Mutex
	exitErr error
}

// Launch starts a detached emulator and returns before it has booted.
func Launch(a AVD, opts LaunchOptions) (*EmulatorProcess, error) {
	emulator := FindEmulator()
	if emulator == "" {
		return nil, errors.New("emulator not found on PATH or in the Android SDK")
	}
	if opts.RAM <= 0 {
		opts.RAM = DefaultRAM
	}
	if opts.GPU == "" {
		opts.GPU = DefaultGPU()
	}

	port, err := FreeEmulatorPort()
	if err != nil {
		return nil, err
	}
	args := launchArgs(a, opts, port)

	logFile, err := os.CreateTemp("", "gadb-slim-"+safeName(a.Name)+"-*.log")
	if err != nil {
		return nil, err
	}
	logPath := logFile.Name()

	cmd := exec.Command(emulator, args...)
	cmd.Stdout, cmd.Stderr = logFile, logFile
	detach(cmd)
	if err := cmd.Start(); err != nil {
		logFile.Close()
		os.Remove(logPath)
		return nil, fmt.Errorf("cannot start emulator: %w", err)
	}
	logFile.Close()

	process := &EmulatorProcess{
		AVD:     a,
		Serial:  "emulator-" + strconv.Itoa(port),
		Port:    port,
		LogPath: logPath,
		Cmd:     cmd,
		done:    make(chan struct{}),
	}
	go func() {
		err := cmd.Wait()
		process.mu.Lock()
		process.exitErr = err
		process.mu.Unlock()
		close(process.done)
	}()
	return process, nil
}

func launchArgs(a AVD, opts LaunchOptions, port int) []string {
	args := []string{
		"-avd", a.Name,
		"-memory", strconv.Itoa(opts.RAM),
		"-gpu", opts.GPU,
		"-port", strconv.Itoa(port),
	}
	if opts.LowRAM {
		args = append(args, "-lowram")
	}
	if opts.Headless {
		args = append(args, "-no-window")
	}
	if opts.NoAudio {
		args = append(args, "-no-audio")
	}
	if opts.NoBootAnim {
		args = append(args, "-no-boot-anim")
	}
	if opts.Snapshot != "" {
		args = append(args, "-snapshot", opts.Snapshot, "-no-snapshot-save")
	} else {
		args = append(args, "-no-snapshot-load")
	}
	return args
}

// Exited reports whether the emulator process has exited.
func (p *EmulatorProcess) Exited() bool {
	select {
	case <-p.done:
		return true
	default:
		return false
	}
}

// ExitError is the process exit error, if any.
func (p *EmulatorProcess) ExitError() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.exitErr
}

// WaitForBoot polls sys.boot_completed and fails fast when the emulator exits.
func (p *EmulatorProcess) WaitForBoot(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if p.Exited() {
			if err := p.ExitError(); err != nil {
				return fmt.Errorf("emulator exited during boot: %v\n%s", err, TailLog(p.LogPath, 40))
			}
			return fmt.Errorf("emulator exited during boot\n%s", TailLog(p.LogPath, 40))
		}
		result := adb.ShellCommand(p.Serial, 10*time.Second, "getprop sys.boot_completed")
		if result.Error == nil && strings.TrimSpace(string(result.Output)) == "1" {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("timed out after %s waiting for %s to boot\n%s",
		timeout, p.Serial, TailLog(p.LogPath, 40))
}

// Stop asks the emulator to shut down, then kills it if it does not.
func (p *EmulatorProcess) Stop(timeout time.Duration) error {
	if p == nil || p.Cmd == nil || p.Cmd.Process == nil || p.Exited() {
		return nil
	}
	adb.RunOnlyWithTimeout(15*time.Second, "adb", "-s", p.Serial, "emu", "kill")
	select {
	case <-p.done:
		return nil
	case <-time.After(timeout):
		_ = p.Cmd.Process.Kill()
		select {
		case <-p.done:
		case <-time.After(5 * time.Second):
		}
		return fmt.Errorf("%s did not stop after %s", p.Serial, timeout)
	}
}

// StopSerial stops an emulator by serial and waits for it to disappear from adb.
func StopSerial(serial string, timeout time.Duration) error {
	adb.RunOnlyWithTimeout(15*time.Second, "adb", "-s", serial, "emu", "kill")
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !serialVisible(serial) {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("%s did not stop after %s", serial, timeout)
}

func serialVisible(serial string) bool {
	devices, err := adb.GetDeviceList()
	if err != nil {
		return true // cannot tell, keep waiting
	}
	for _, device := range devices {
		if device.Serial == serial {
			return true
		}
	}
	return false
}

// TailLog returns the last lines of an emulator log.
func TailLog(path string, lines int) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return ""
	}
	const maxRead = 16 * 1024
	offset := int64(0)
	if info.Size() > maxRead {
		offset = info.Size() - maxRead
	}
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return ""
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return ""
	}

	parts := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(parts) > lines {
		parts = parts[len(parts)-lines:]
	}
	return strings.Join(parts, "\n")
}

func safeName(name string) string {
	var out strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '.', r == '_', r == '-':
			out.WriteRune(r)
		default:
			out.WriteByte('_')
		}
	}
	return out.String()
}
