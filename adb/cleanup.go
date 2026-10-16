package adb

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Well-known emulator locations that accumulate disposable data.
const (
	PerfettoTracesPath = "/data/misc/perfetto-traces"
	TombstonesPath     = "/data/tombstones"
	TempDirPath        = "/data/local/tmp"
)

// SharedStorageCleanupTargets are the shared-storage folders cleanup empties
// when the user opts in to shared storage cleanup.
var SharedStorageCleanupTargets = []string{
	"/sdcard/Download",
	"/sdcard/Pictures",
	"/sdcard/DCIM",
}

// SystemDebrisTargets are system scratch areas that grow with installs,
// crashes and model downloads but are always regenerated on demand. Cleanup
// empties them when root is available.
var SystemDebrisTargets = []string{
	"/data/system/package_cache",
	"/data/anr",
	"/data/system/dropbox",
	"/data/misc/textclassifier",
	"/data/resource-cache",
}

// SystemDebrisNames returns the display names (last path element) of the
// system debris targets.
func SystemDebrisNames() []string {
	names := make([]string, 0, len(SystemDebrisTargets))
	for _, target := range SystemDebrisTargets {
		names = append(names, filepath.Base(target))
	}
	return names
}

// tempDirKeep lists entries inside TempDirPath that cleanup must preserve.
// lldb-server/start_lldb_server.sh and .studio are pushed by Android Studio
// and LLDB on demand and are expensive to re-download; avdslim_state.json is
// required by avdslim to restore emulator settings; devicelab-driver.log is
// written by the devicelab test driver.
var tempDirKeep = []string{
	".studio",
	"avdslim_state.json",
	"lldb-server",
	"start_lldb_server.sh",
	"devicelab-driver.log",
}

func isTempDirKept(name string) bool {
	for _, keep := range tempDirKeep {
		if strings.HasPrefix(name, keep) {
			return true
		}
	}
	return false
}

// EmulatorCheck is the outcome of double-checking that a device is really an
// emulator before anything destructive runs against it.
type EmulatorCheck struct {
	Serial    string
	Model     string
	Confirmed bool
	Signals   []string
}

// emulatorHardwareValues are boot hardware names used by Android emulators.
var emulatorHardwareValues = map[string]bool{
	"goldfish":   true,
	"goldfish64": true,
	"ranchu":     true,
	"ranchu64":   true,
	"cuttlefish": true,
}

// VerifyEmulator double-checks that the device behind serial is an emulator.
// It combines the serial prefix with kernel/build properties reported by the
// device, so a physical device can never pass.
func VerifyEmulator(serial string) (EmulatorCheck, error) {
	check := EmulatorCheck{Serial: serial}

	devices, err := GetDeviceList()
	if err != nil {
		return check, fmt.Errorf("cannot list devices: %w", err)
	}

	state := ""
	for _, d := range devices {
		if d.Serial == serial {
			state = d.State
			check.Model = d.Model
			break
		}
	}
	if state == "" {
		return check, fmt.Errorf("device %s not found", serial)
	}
	if state != "device" {
		return check, fmt.Errorf("device %s is not ready (state: %s)", serial, state)
	}

	props := getDeviceProps(serial, "ro.kernel.qemu", "ro.boot.qemu", "ro.hardware", "ro.build.characteristics")
	check.Confirmed, check.Signals = evaluateEmulator(serial, props)
	return check, nil
}

// evaluateEmulator decides emulator-ness from the serial and getprop values.
// At least one property-level signal is required: only an emulator kernel or
// build reports QEMU/goldfish/ranchu/cuttlefish characteristics.
func evaluateEmulator(serial string, props map[string]string) (bool, []string) {
	signals := []string{}
	strong := false

	if strings.HasPrefix(serial, "emulator-") {
		signals = append(signals, "serial starts with emulator-")
	}
	if props["ro.kernel.qemu"] == "1" {
		signals = append(signals, "ro.kernel.qemu=1")
		strong = true
	}
	if props["ro.boot.qemu"] == "1" {
		signals = append(signals, "ro.boot.qemu=1")
		strong = true
	}
	if hw := strings.ToLower(strings.TrimSpace(props["ro.hardware"])); emulatorHardwareValues[hw] {
		signals = append(signals, "ro.hardware="+hw)
		strong = true
	}
	if strings.Contains(strings.ToLower(props["ro.build.characteristics"]), "emulator") {
		signals = append(signals, "ro.build.characteristics contains emulator")
		strong = true
	}

	return strong, signals
}

func getDeviceProps(serial string, names ...string) map[string]string {
	result := make(map[string]string, len(names))

	parts := make([]string, 0, len(names))
	for _, name := range names {
		parts = append(parts, "getprop "+name)
	}

	res := ShellCommand(serial, 15*time.Second, strings.Join(parts, "; "))
	if res.Error != nil {
		return result
	}

	lines := splitLines(string(res.Output))
	for i, name := range names {
		if i < len(lines) {
			result[name] = strings.TrimSpace(lines[i])
		} else {
			result[name] = ""
		}
	}
	return result
}

// deviceArgs prefixes adb arguments with -s serial when a serial is given.
func deviceArgs(serial string) []string {
	if serial == "" {
		return nil
	}
	return []string{"-s", serial}
}

// ShellCommand runs a shell command on the device and returns its output.
func ShellCommand(serial string, timeout time.Duration, command string) CommandReturn {
	args := append(deviceArgs(serial), "shell", command)
	return RunOnlyWithTimeout(timeout, "adb", args...)
}

// RootMode describes how root shell access is available on a device.
type RootMode int

const (
	// RootUnavailable means no root shell access is available.
	RootUnavailable RootMode = iota
	// RootViaSu means `su 0 <cmd>` grants root without restarting adbd.
	RootViaSu
	// RootViaAdbd means `adb root` restarted adbd as root.
	RootViaAdbd
)

// DetectRootMode probes the device for root shell access. It prefers `su`
// because it does not restart adbd and interrupt other tools. The second
// return value reports whether this call started adbd as root, in which case
// the caller should RestoreAdbd when finished.
func DetectRootMode(serial string) (RootMode, bool) {
	// adbd may already be rooted (e.g. adb root was run before).
	res := ShellCommand(serial, 15*time.Second, "id")
	if res.Error == nil && strings.Contains(string(res.Output), "uid=0") {
		return RootViaAdbd, false
	}

	res = ShellCommand(serial, 15*time.Second, "su 0 id")
	if res.Error == nil && strings.Contains(string(res.Output), "uid=0") {
		return RootViaSu, false
	}

	res = RunOnlyWithTimeout(30*time.Second, "adb", append(deviceArgs(serial), "root")...)
	out := string(res.Output)
	if strings.Contains(out, "restarting adbd as root") {
		waitForDevice(serial)
		return RootViaAdbd, true
	}
	if strings.Contains(out, "already running as root") {
		waitForDevice(serial)
		return RootViaAdbd, false
	}
	return RootUnavailable, false
}

// ShellAsRoot runs a shell command with root privileges using the detected mode.
func ShellAsRoot(serial string, mode RootMode, timeout time.Duration, command string) CommandReturn {
	switch mode {
	case RootViaSu:
		return ShellCommand(serial, timeout, "su 0 "+command)
	case RootViaAdbd:
		return ShellCommand(serial, timeout, command)
	default:
		return CommandReturn{Error: errors.New("root access is not available")}
	}
}

// RestoreAdbd returns adbd to the non-root shell after adb root was used.
func RestoreAdbd(serial string) {
	RunOnlyWithTimeout(30*time.Second, "adb", append(deviceArgs(serial), "unroot")...)
	waitForDevice(serial)
}

func waitForDevice(serial string) {
	RunOnlyWithTimeout(30*time.Second, "adb", append(deviceArgs(serial), "wait-for-device")...)
}

// PathSizeKB returns the disk usage of a path in kilobytes (du -sk).
func PathSizeKB(serial string, mode RootMode, p string) (int64, error) {
	var res CommandReturn
	if mode == RootUnavailable {
		res = ShellCommand(serial, 30*time.Second, "du -sk "+shellQuote(p))
	} else {
		res = ShellAsRoot(serial, mode, 30*time.Second, "du -sk "+shellQuote(p))
	}
	if res.Error != nil {
		return 0, res.Error
	}
	return parseSingleDuKB(string(res.Output))
}

// DataPartitionSpace returns total, used and available KB of the /data partition.
func DataPartitionSpace(serial string) (int64, int64, int64, error) {
	res := ShellCommand(serial, 15*time.Second, "df -k /data")
	if res.Error != nil {
		return 0, 0, 0, res.Error
	}
	return parseDfOutput(string(res.Output))
}

// TempEntry is an item inside /data/local/tmp.
type TempEntry struct {
	Name   string
	Path   string
	SizeKB int64 // -1 when the size could not be read
	Kept   bool
}

// ListTempEntries lists the contents of /data/local/tmp with their sizes.
// Kept marks entries that cleanup preserves.
func ListTempEntries(serial string, mode RootMode) ([]TempEntry, error) {
	res := ShellCommand(serial, 15*time.Second, "ls -1a "+shellQuote(TempDirPath))
	if res.Error != nil {
		return nil, res.Error
	}

	var names []string
	for _, line := range splitLines(string(res.Output)) {
		name := strings.TrimSpace(line)
		if name == "" || name == "." || name == ".." || strings.Contains(name, "/") {
			continue
		}
		names = append(names, name)
	}

	sizes := map[string]int64{}
	if len(names) > 0 {
		duCmd := "du -sk " + shellQuote(TempDirPath) + "/* 2>/dev/null"
		var duRes CommandReturn
		if mode == RootUnavailable {
			duRes = ShellCommand(serial, 60*time.Second, duCmd)
		} else {
			duRes = ShellAsRoot(serial, mode, 60*time.Second, duCmd)
		}
		sizes = parseDuListing(string(duRes.Output))
	}

	entries := make([]TempEntry, 0, len(names))
	for _, name := range names {
		p := TempDirPath + "/" + name
		size, ok := sizes[p]
		if !ok {
			size = -1
		}
		entries = append(entries, TempEntry{Name: name, Path: p, SizeKB: size, Kept: isTempDirKept(name)})
	}
	return entries, nil
}

// ClearTempDir removes every non-kept entry previously listed by ListTempEntries.
func ClearTempDir(serial string, mode RootMode, entries []TempEntry) CommandReturn {
	paths := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.Kept {
			paths = append(paths, e.Path)
		}
	}
	if len(paths) == 0 {
		return CommandReturn{}
	}
	return rmPaths(serial, mode, paths)
}

// ClearDirContents removes everything inside dir while keeping dir itself.
func ClearDirContents(serial string, mode RootMode, dir string) CommandReturn {
	return rmCommand(serial, mode, shellQuote(dir)+"/*")
}

func rmPaths(serial string, mode RootMode, paths []string) CommandReturn {
	quoted := make([]string, 0, len(paths))
	for _, p := range paths {
		quoted = append(quoted, shellQuote(p))
	}
	return rmCommand(serial, mode, strings.Join(quoted, " "))
}

func rmCommand(serial string, mode RootMode, targets string) CommandReturn {
	cmd := "rm -rf " + targets
	if mode == RootUnavailable {
		return ShellCommand(serial, 60*time.Second, cmd)
	}
	return ShellAsRoot(serial, mode, 60*time.Second, cmd)
}

// PackageInfo describes an installed third-party package and its footprint.
type PackageInfo struct {
	Name   string
	SizeKB int64 // apk dir + internal & external app data (best effort)
}

// ListThirdPartyPackagesWithSize lists installed third-party packages with a
// best-effort total size (apk files plus app data when readable).
func ListThirdPartyPackagesWithSize(serial string, mode RootMode) ([]PackageInfo, error) {
	res := ShellCommand(serial, 20*time.Second, "pm list packages -3 -f")
	if res.Error != nil {
		return nil, res.Error
	}

	var pkgs []PackageInfo
	var apkDirs, dataDirs, extDirs []string
	for _, line := range splitLines(string(res.Output)) {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "package:") {
			continue
		}

		value := strings.TrimPrefix(line, "package:")
		sep := strings.LastIndex(value, "=")
		if sep <= 0 || sep == len(value)-1 {
			continue
		}

		name := value[sep+1:]
		apkPath := value[:sep]
		pkgs = append(pkgs, PackageInfo{Name: name})
		apkDirs = append(apkDirs, filepath.Dir(apkPath))
		dataDirs = append(dataDirs, "/data/data/"+name)
		extDirs = append(extDirs, "/sdcard/Android/data/"+name)
	}

	if len(pkgs) == 0 {
		return pkgs, nil
	}

	apkSizes := duSizes(serial, mode, apkDirs)
	dataSizes := duSizes(serial, mode, dataDirs)
	extSizes := duSizes(serial, mode, extDirs)
	for i := range pkgs {
		pkgs[i].SizeKB = apkSizes[apkDirs[i]] + dataSizes[dataDirs[i]] + extSizes[extDirs[i]]
	}
	return pkgs, nil
}

func duSizes(serial string, mode RootMode, paths []string) map[string]int64 {
	if len(paths) == 0 {
		return nil
	}

	quoted := make([]string, 0, len(paths))
	for _, p := range paths {
		quoted = append(quoted, shellQuote(p))
	}
	cmd := "du -sk " + strings.Join(quoted, " ") + " 2>/dev/null"

	var res CommandReturn
	if mode == RootUnavailable {
		res = ShellCommand(serial, 60*time.Second, cmd)
	} else {
		res = ShellAsRoot(serial, mode, 60*time.Second, cmd)
	}
	return parseDuListing(string(res.Output))
}

// UninstallFrom uninstalls a package from the given device.
func UninstallFrom(serial string, packageName string) CommandReturn {
	args := append(deviceArgs(serial), "uninstall", packageName)
	return RunOnlyWithTimeout(120*time.Second, "adb", args...)
}

// protectedPackagePrefixes are never offered for uninstall by cleanup: they
// are development and test infrastructure that test runs rely on.
var protectedPackagePrefixes = []string{
	"androidx.test.",
	"com.android.cli.",
	"dev.devicelab.",
}

// IsProtectedPackage reports whether cleanup must never uninstall name. The
// configured package and every package below it (e.g. its .test variant) are
// always protected, along with well-known test tooling and extraKeep entries.
func IsProtectedPackage(name string, configuredPackage string, extraKeep []string) bool {
	if name == "" {
		return true
	}
	if configuredPackage != "" && (name == configuredPackage || strings.HasPrefix(name, configuredPackage+".")) {
		return true
	}
	for _, prefix := range protectedPackagePrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	for _, keep := range extraKeep {
		keep = strings.TrimSpace(keep)
		if keep == "" {
			continue
		}
		if name == keep || strings.HasPrefix(name, keep+".") {
			return true
		}
	}
	return false
}

func parseDfOutput(output string) (int64, int64, int64, error) {
	lines := splitLines(output)
	for i := len(lines) - 1; i >= 0; i-- {
		fields := strings.Fields(lines[i])
		if len(fields) < 4 {
			continue
		}
		total, tErr := strconv.ParseInt(fields[1], 10, 64)
		used, uErr := strconv.ParseInt(fields[2], 10, 64)
		avail, aErr := strconv.ParseInt(fields[3], 10, 64)
		if tErr == nil && uErr == nil && aErr == nil {
			return total, used, avail, nil
		}
	}
	return 0, 0, 0, errors.New("cannot parse df output")
}

func parseSingleDuKB(output string) (int64, error) {
	for _, line := range splitLines(output) {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		kb, err := strconv.ParseInt(fields[0], 10, 64)
		if err == nil {
			return kb, nil
		}
	}
	return 0, errors.New("cannot parse du output: " + strings.TrimSpace(output))
}

// parseDuListing parses `du -sk` output into a path -> KB map.
func parseDuListing(output string) map[string]int64 {
	sizes := map[string]int64{}
	for _, line := range splitLines(output) {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "du:") {
			continue
		}
		sep := strings.IndexAny(line, " \t")
		if sep <= 0 {
			continue
		}
		kb, err := strconv.ParseInt(line[:sep], 10, 64)
		if err != nil {
			continue
		}
		if p := strings.TrimSpace(line[sep+1:]); p != "" {
			sizes[p] = kb
		}
	}
	return sizes
}

// FormatKB renders a KB size as a human readable string.
func FormatKB(kb int64) string {
	switch {
	case kb < 0:
		return "unknown"
	case kb >= 1024*1024*1024:
		return fmt.Sprintf("%.1f TB", float64(kb)/(1024*1024*1024))
	case kb >= 1024*1024:
		return fmt.Sprintf("%.1f GB", float64(kb)/(1024*1024))
	case kb >= 1024:
		return fmt.Sprintf("%.1f MB", float64(kb)/1024)
	default:
		return fmt.Sprintf("%d KB", kb)
	}
}

func splitLines(output string) []string {
	return strings.Split(strings.ReplaceAll(output, "\r", ""), "\n")
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
