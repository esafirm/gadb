package slim

import (
	"fmt"
	"strings"
	"time"

	"github.com/esafirm/gadb/adb"
)

// Device is the guest surface On, Off and Status need. ADBDevice is the real
// implementation, tests use an in-memory fake.
type Device interface {
	Serial() string
	InstalledPackages() (map[string]bool, error)
	DisabledPackages() (map[string]bool, error)
	DisablePackage(pkg string) error
	EnablePackage(pkg string) error
	GetSetting(namespace, key string) (*string, error)
	PutSetting(namespace, key, value string) error
	DeleteSetting(namespace, key string) error
	TrimMemory()
}

// ADBDevice talks to one verified emulator over adb.
type ADBDevice struct {
	serial string
}

// NewADBDevice verifies that serial is an emulator and returns a device.
func NewADBDevice(serial string) (*ADBDevice, error) {
	check, err := adb.VerifyEmulator(serial)
	if err != nil {
		return nil, err
	}
	if !check.Confirmed {
		return nil, fmt.Errorf("%s is not a verified emulator", serial)
	}
	return &ADBDevice{serial: serial}, nil
}

// Serial is the adb serial of this device.
func (d *ADBDevice) Serial() string { return d.serial }

// InstalledPackages lists every installed package.
func (d *ADBDevice) InstalledPackages() (map[string]bool, error) {
	return d.packageSet("pm list packages")
}

// DisabledPackages lists packages disabled for user 0.
func (d *ADBDevice) DisabledPackages() (map[string]bool, error) {
	return d.packageSet("pm list packages -d")
}

func (d *ADBDevice) packageSet(command string) (map[string]bool, error) {
	result := adb.ShellCommand(d.serial, 60*time.Second, command)
	if result.Error != nil {
		return nil, result.Error
	}
	return parsePackages(string(result.Output)), nil
}

func parsePackages(output string) map[string]bool {
	packages := make(map[string]bool)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "package:"))
		if line != "" {
			packages[line] = true
		}
	}
	return packages
}

// DisablePackage disables a package for the current user.
func (d *ADBDevice) DisablePackage(pkg string) error {
	result := adb.ShellCommand(d.serial, 30*time.Second, "pm disable-user --user 0 "+pkg)
	output := strings.TrimSpace(string(result.Output))
	if result.Error != nil || !strings.Contains(output, "disabled-user") {
		return fmt.Errorf("%s: %s", pkg, output)
	}
	return nil
}

// EnablePackage re-enables a disabled package.
func (d *ADBDevice) EnablePackage(pkg string) error {
	result := adb.ShellCommand(d.serial, 30*time.Second, "pm enable "+pkg)
	output := strings.TrimSpace(string(result.Output))
	if result.Error != nil || !strings.Contains(output, "enabled") {
		return fmt.Errorf("%s: %s", pkg, output)
	}
	return nil
}

// GetSetting returns nil when the setting is unset.
func (d *ADBDevice) GetSetting(namespace, key string) (*string, error) {
	result := adb.ShellCommand(d.serial, 15*time.Second, "settings get "+namespace+" "+key)
	if result.Error != nil {
		return nil, result.Error
	}
	value := strings.TrimSpace(string(result.Output))
	if value == "" || value == "null" {
		return nil, nil
	}
	return &value, nil
}

// PutSetting writes a setting.
func (d *ADBDevice) PutSetting(namespace, key, value string) error {
	result := adb.ShellCommand(d.serial, 15*time.Second,
		"settings put "+namespace+" "+key+" "+shellQuote(value))
	return result.Error
}

// DeleteSetting removes a setting so it falls back to its default.
func (d *ADBDevice) DeleteSetting(namespace, key string) error {
	result := adb.ShellCommand(d.serial, 15*time.Second, "settings delete "+namespace+" "+key)
	return result.Error
}

// TrimMemory asks Android to drop cached processes and trim heaps. Failures are
// best effort: the package and setting changes are already applied.
func (d *ADBDevice) TrimMemory() {
	adb.ShellCommand(d.serial, 30*time.Second, "am kill-all")
	adb.ShellCommand(d.serial, 60*time.Second, "am trim-memory --all COMPLETE")
	adb.ShellCommand(d.serial, 15*time.Second, "sync")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}
