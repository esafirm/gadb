package adb

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestEvaluateEmulatorStandardAvd(t *testing.T) {
	confirmed, signals := evaluateEmulator("emulator-5554", map[string]string{
		"ro.kernel.qemu":           "1",
		"ro.boot.qemu":             "1",
		"ro.hardware":              "ranchu",
		"ro.build.characteristics": "emulator",
	})

	if !confirmed {
		t.Fatalf("expected standard AVD to be confirmed, signals: %#v", signals)
	}
	if len(signals) < 4 {
		t.Fatalf("expected at least 4 signals, got %#v", signals)
	}
}

func TestEvaluateEmulatorRequiresPropertySignal(t *testing.T) {
	// An emulator-looking serial alone must never be enough.
	confirmed, _ := evaluateEmulator("emulator-5554", map[string]string{})
	if confirmed {
		t.Fatal("serial prefix alone must not confirm an emulator")
	}
}

func TestEvaluateEmulatorRejectsPhysicalDevice(t *testing.T) {
	confirmed, _ := evaluateEmulator("R58M123ABCD", map[string]string{
		"ro.kernel.qemu": "0",
		"ro.hardware":    "qcom",
	})
	if confirmed {
		t.Fatal("physical device must never be confirmed as an emulator")
	}
}

func TestEvaluateEmulatorFlagshipProps(t *testing.T) {
	// Some emulators (e.g. cloud) do not use the emulator- serial prefix.
	confirmed, signals := evaluateEmulator("127.0.0.1:6520", map[string]string{
		"ro.hardware": "goldfish",
	})
	if !confirmed {
		t.Fatalf("goldfish hardware must confirm an emulator, signals: %#v", signals)
	}
}

func TestIsProtectedPackage(t *testing.T) {
	configured := "com.bandlab.bandlab"

	cases := []struct {
		name     string
		pkg      string
		expected bool
	}{
		{"configured package", configured, true},
		{"configured test variant", configured + ".test", true},
		{"androidx test services", "androidx.test.services", true},
		{"androidx orchestrator", "androidx.test.orchestrator", true},
		{"devicelab driver", "dev.devicelab.driver.android", true},
		{"android cli tooling", "com.android.cli.interact.instrumentation", true},
		{"unrelated app", "com.example.demo", false},
		{"empty name", "", true},
	}

	for _, c := range cases {
		if got := IsProtectedPackage(c.pkg, configured, nil); got != c.expected {
			t.Errorf("%s: IsProtectedPackage(%q) = %v, want %v", c.name, c.pkg, got, c.expected)
		}
	}

	// Extra keep entries protect the package itself and its children only.
	if !IsProtectedPackage("com.my.keep", "", []string{"com.my.keep"}) {
		t.Error("extra keep entry must protect its package")
	}
	if !IsProtectedPackage("com.my.keep.sub", "", []string{"com.my.keep"}) {
		t.Error("extra keep entry must protect child packages")
	}
	if IsProtectedPackage("com.my.keepsake", "", []string{"com.my.keep"}) {
		t.Error("extra keep entry must not protect lookalike prefixes")
	}
}

func TestParseDfOutput(t *testing.T) {
	output := "Filesystem      1K-blocks    Used Available Use% Mounted on\n" +
		"/dev/block/dm-5   6082144 2821676   3260468  47% /mnt/pass_through/0/emulated\n"

	total, used, avail, err := parseDfOutput(output)
	if err != nil {
		t.Fatalf("parseDfOutput error: %v", err)
	}
	if total != 6082144 || used != 2821676 || avail != 3260468 {
		t.Fatalf("unexpected values: total=%d used=%d avail=%d", total, used, avail)
	}
}

func TestParseDfOutputNoData(t *testing.T) {
	if _, _, _, err := parseDfOutput("Filesystem 1K-blocks Used Available Use% Mounted on\n"); err == nil {
		t.Fatal("expected error for header-only df output")
	}
}

func TestParseDuListing(t *testing.T) {
	output := "7172\t/data/local/tmp/.studio\r\n" +
		"45236\t/data/local/tmp/lldb-server\n" +
		"du: cannot access '/data/local/tmp/nope': No such file or directory\n" +
		"740\t/data/local/tmp/some dir\n"

	sizes := parseDuListing(output)

	if sizes["/data/local/tmp/.studio"] != 7172 {
		t.Errorf("expected .studio size 7172, got %d", sizes["/data/local/tmp/.studio"])
	}
	if sizes["/data/local/tmp/lldb-server"] != 45236 {
		t.Errorf("expected lldb-server size 45236, got %d", sizes["/data/local/tmp/lldb-server"])
	}
	if sizes["/data/local/tmp/some dir"] != 740 {
		t.Errorf("expected path with space to be parsed, got %d", sizes["/data/local/tmp/some dir"])
	}
	if len(sizes) != 3 {
		t.Errorf("expected 3 entries, got %d: %#v", len(sizes), sizes)
	}
}

func TestParseSingleDuKB(t *testing.T) {
	kb, err := parseSingleDuKB("75188\t/data/local/tmp\n")
	if err != nil || kb != 75188 {
		t.Fatalf("expected 75188, got %d (err: %v)", kb, err)
	}

	if _, err := parseSingleDuKB("du: cannot access '/nope': No such file or directory\n"); err == nil {
		t.Fatal("expected error when no size is present")
	}
}

func TestParseSingleDuKBWithCarriageReturn(t *testing.T) {
	kb, err := parseSingleDuKB("1234\t/data/misc/perfetto-traces\r\n")
	if err != nil || kb != 1234 {
		t.Fatalf("expected 1234, got %d (err: %v)", kb, err)
	}
}

func TestIsTempDirKept(t *testing.T) {
	expected := map[string]bool{
		".studio":                   true,
		"avdslim_state.json":        true,
		"lldb-server":               true,
		"start_lldb_server.sh":      true,
		"devicelab-driver.log":      true,
		"deployer_device_jar_x.jar": false,
		"patch.seq":                 false,
	}

	for name, want := range expected {
		if got := isTempDirKept(name); got != want {
			t.Errorf("isTempDirKept(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestSystemDebrisTargetsAreSafe(t *testing.T) {
	// Guard rails for the target list itself: only leaf scratch areas under
	// /data that are safe to empty, never broad roots or kept locations.
	tooBroad := map[string]bool{
		"/data":        true,
		"/data/data":   true,
		"/data/system": true,
		"/data/misc":   true,
		"/data/local":  true,
		"/data/media":  true,
	}

	seen := map[string]bool{}
	for _, target := range SystemDebrisTargets {
		if !strings.HasPrefix(target, "/data/") {
			t.Errorf("debris target %q must be an absolute path under /data", target)
		}
		if tooBroad[target] {
			t.Errorf("debris target %q is too broad to empty", target)
		}
		if seen[target] {
			t.Errorf("duplicate debris target %q", target)
		}
		seen[target] = true

		for _, other := range SystemDebrisTargets {
			if other != target && strings.HasPrefix(other, target+"/") {
				t.Errorf("debris target %q contains %q", target, other)
			}
		}
	}

	names := SystemDebrisNames()
	if len(names) != len(SystemDebrisTargets) {
		t.Fatalf("got %d names for %d targets", len(names), len(SystemDebrisTargets))
	}
	for i, name := range names {
		if want := filepath.Base(SystemDebrisTargets[i]); name != want {
			t.Errorf("name[%d] = %q, want %q", i, name, want)
		}
	}
}

func TestFormatKB(t *testing.T) {
	cases := []struct {
		kb       int64
		expected string
	}{
		{-1, "unknown"},
		{0, "0 KB"},
		{512, "512 KB"},
		{1024, "1.0 MB"},
		{75188, "73.4 MB"},
		{1024 * 1024, "1.0 GB"},
		{6082144, "5.8 GB"},
	}

	for _, c := range cases {
		if got := FormatKB(c.kb); got != c.expected {
			t.Errorf("FormatKB(%d) = %q, want %q", c.kb, got, c.expected)
		}
	}
}

func TestShellQuote(t *testing.T) {
	quoted := shellQuote("/data/local/tmp/plain")
	if !reflect.DeepEqual(quoted, "'/data/local/tmp/plain'") {
		t.Fatalf("unexpected quote: %s", quoted)
	}
	// Single quotes inside paths must not break out of the quoting.
	quoted = shellQuote("it's")
	if quoted != `'it'\''s'` {
		t.Fatalf("unexpected escaping: %s", quoted)
	}
}
