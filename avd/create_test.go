package avd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestValidAVDName(t *testing.T) {
	for _, name := range []string{"Pixel_5", "pixel-5", "a.b_c-d", "A1"} {
		if !ValidAVDName(name) {
			t.Errorf("ValidAVDName(%q) = false, want true", name)
		}
	}
	for _, name := range []string{"", "bad name", "bad/name", "bad\nname", "é"} {
		if ValidAVDName(name) {
			t.Errorf("ValidAVDName(%q) = true, want false", name)
		}
	}
}

func TestFindAVDManagerUsesSDK(t *testing.T) {
	sdk := t.TempDir()
	t.Setenv("ANDROID_HOME", sdk)
	t.Setenv("ANDROID_SDK_ROOT", "")
	t.Setenv("PATH", t.TempDir())

	name := "avdmanager"
	if runtime.GOOS == "windows" {
		name = "avdmanager.bat"
	}
	path := filepath.Join(sdk, "cmdline-tools", "latest", "bin", name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	if got := FindAVDManager(); got != path {
		t.Fatalf("FindAVDManager() = %q, want %q", got, path)
	}
}

func TestCreateRejectsExistingAVD(t *testing.T) {
	testAVD(t) // creates Test.avd and sets ANDROID_AVD_HOME

	_, err := Create(CreateOptions{Name: "Test"})
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("Create() error = %v, want an already-exists error", err)
	}
}

func TestCreateWithoutImageExplainsWhatIsInstalled(t *testing.T) {
	sdk := useSDK(t)
	abi := HostABI()
	writeImage(t, sdk, "android-34", "google_apis", abi, map[string]string{
		"AndroidVersion.ApiLevel": "34.0",
		"SystemImage.Abi":         abi,
		"SystemImage.TagId":       "google_apis",
	}, false)
	t.Setenv("ANDROID_AVD_HOME", t.TempDir())

	_, err := Create(CreateOptions{Name: "Pixel_Slim"})
	if err == nil {
		t.Fatal("Create() succeeded without a complete image")
	}
	message := err.Error()
	for _, want := range []string{
		"no complete 4 KB google_apis image",
		"incomplete (system.img missing)",
		`sdkmanager "system-images;android-35;google_apis;` + abi + `"`,
	} {
		if !strings.Contains(message, want) {
			t.Errorf("error missing %q:\n%s", want, message)
		}
	}
}

func TestCreateRefusesMissingAVDManager(t *testing.T) {
	sdk := useSDK(t)
	abi := HostABI()
	writeImage(t, sdk, "android-35", "google_apis", abi, map[string]string{
		"AndroidVersion.ApiLevel": "35.0",
		"SystemImage.Abi":         abi,
		"SystemImage.TagId":       "google_apis",
	}, true)
	t.Setenv("ANDROID_AVD_HOME", t.TempDir())
	t.Setenv("PATH", t.TempDir())

	_, err := Create(CreateOptions{Name: "Pixel_Slim"})
	if err == nil || !strings.Contains(err.Error(), "avdmanager not found") {
		t.Fatalf("Create() error = %v, want a missing avdmanager error", err)
	}
}

func TestCreateRunsAVDManagerAndTunes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake avdmanager is a shell script")
	}

	sdk := useSDK(t)
	abi := HostABI()
	writeImage(t, sdk, "android-35", "google_apis", abi, map[string]string{
		"AndroidVersion.ApiLevel": "35.0",
		"SystemImage.Abi":         abi,
		"SystemImage.TagId":       "google_apis",
	}, true)
	t.Setenv("ANDROID_AVD_HOME", t.TempDir())

	argsFile := installFakeAVDManager(t, sdk)
	stubStopped(t)

	result, err := Create(CreateOptions{
		Name:   "Pixel_Slim",
		API:    35,
		Device: "pixel_6",
		RAM:    2048,
		GPU:    "host",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.AVD.Name != "Pixel_Slim" {
		t.Fatalf("AVD = %q", result.AVD.Name)
	}
	if result.Image.APIMajor != 35 {
		t.Fatalf("Image = %+v", result.Image)
	}
	if result.Device != "pixel_6" {
		t.Fatalf("Device = %q", result.Device)
	}
	if result.Tune == nil {
		t.Fatal("Tune = nil")
	}

	args, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	wantArgs := "Pixel_Slim\nsystem-images;android-35;google_apis;" + abi + "\npixel_6\n"
	if string(args) != wantArgs {
		t.Fatalf("avdmanager args = %q, want %q", args, wantArgs)
	}

	config, err := LoadConfig(result.AVD.ConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{
		"hw.ramSize":     "2048",
		"hw.gpu.mode":    "host",
		"hw.audioOutput": "no",
		"hw.camera.back": "none",
	} {
		if got, _ := config.Get(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
	if _, err := os.Stat(result.AVD.ManifestPath()); err != nil {
		t.Fatalf("manifest was not written: %v", err)
	}
}

func TestCreateWarnsOnOldAPI(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake avdmanager is a shell script")
	}

	sdk := useSDK(t)
	abi := HostABI()
	writeImage(t, sdk, "android-26", "google_apis", abi, map[string]string{
		"AndroidVersion.ApiLevel": "26.0",
		"SystemImage.Abi":         abi,
		"SystemImage.TagId":       "google_apis",
	}, true)
	t.Setenv("ANDROID_AVD_HOME", t.TempDir())

	installFakeAVDManager(t, sdk)
	stubStopped(t)

	result, err := Create(CreateOptions{Name: "Pixel_Old", API: 26})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Warnings) == 0 || !strings.Contains(result.Warnings[0], "API 26 is old") {
		t.Fatalf("Warnings = %+v, want an old-API warning", result.Warnings)
	}
}

// installFakeAVDManager writes a shell script that mimics `avdmanager create
// avd`: it records its arguments and creates a minimal AVD config.
func installFakeAVDManager(t *testing.T, sdk string) string {
	t.Helper()

	argsFile := filepath.Join(t.TempDir(), "avdmanager-args")
	t.Setenv("FAKE_AVDMANAGER_ARGS", argsFile)
	// The script needs basic tools but must not find a real avdmanager.
	t.Setenv("PATH", "/usr/bin:/bin")

	script := `#!/bin/sh
while [ $# -gt 0 ]; do
  case "$1" in
    -n) name="$2"; shift 2 ;;
    -k) pkg="$2"; shift 2 ;;
    -d) device="$2"; shift 2 ;;
    *) shift ;;
  esac
done

mkdir -p "$ANDROID_AVD_HOME/$name.avd"
sysdir=$(printf '%s' "$pkg" | tr ';' '/')
cat > "$ANDROID_AVD_HOME/$name.avd/config.ini" <<EOF
hw.ramSize=4096
hw.gpu.mode=auto
hw.gpu.enabled=yes
hw.audioInput=yes
hw.audioOutput=yes
hw.camera.back=virtualscene
hw.camera.front=emulated
hw.keyboard=yes
image.sysdir.1=$sysdir/
EOF
printf '%s\n' "$name" "$pkg" "$device" > "$FAKE_AVDMANAGER_ARGS"
`

	path := filepath.Join(sdk, "cmdline-tools", "latest", "bin", "avdmanager")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return argsFile
}
