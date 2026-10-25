package avd

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func writeImage(t *testing.T, sdk, platform, tag, abi string, props map[string]string, complete bool) string {
	t.Helper()

	dir := filepath.Join(sdk, "system-images", platform, tag, abi)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	lines := make([]string, 0, len(props))
	for key, value := range props {
		lines = append(lines, key+"="+value)
	}
	sort.Strings(lines)
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(dir, "source.properties"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	if complete {
		for _, name := range []string{"system.img", "ramdisk.img"} {
			if err := os.WriteFile(filepath.Join(dir, name), []byte("image"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}

	return dir
}

func useSDK(t *testing.T) string {
	t.Helper()
	sdk := t.TempDir()
	t.Setenv("ANDROID_HOME", sdk)
	t.Setenv("ANDROID_SDK_ROOT", "")
	return sdk
}

func TestFindImagesFiltersToCompleteGoogleAPIs4KB(t *testing.T) {
	sdk := useSDK(t)
	hostABI := HostABI()
	otherABI := "x86_64"
	if hostABI == "x86_64" {
		otherABI = "arm64-v8a"
	}

	writeImage(t, sdk, "android-35", "google_apis", hostABI, map[string]string{
		"AndroidVersion.ApiLevel": "35.0",
		"SystemImage.Abi":         hostABI,
		"SystemImage.TagId":       "google_apis",
		"SystemImage.TagDisplay":  "Google APIs",
	}, true)
	writeImage(t, sdk, "android-36", "google_apis_playstore", hostABI, map[string]string{
		"AndroidVersion.ApiLevel": "36.0",
		"SystemImage.Abi":         hostABI,
		"SystemImage.TagId":       "google_apis_playstore",
	}, true)
	writeImage(t, sdk, "android-37.2-beta3", "google_apis_playstore_ps16k", hostABI, map[string]string{
		"AndroidVersion.ApiLevel": "37.1",
		"SystemImage.Abi":         hostABI,
		"SystemImage.TagId":       "google_apis_playstore,page_size_16kb",
		"Pkg.Desc":                "System Image Page Size 16k arm64-v8a with Google Play.",
	}, true)
	writeImage(t, sdk, "android-34", "google_apis", hostABI, map[string]string{
		"AndroidVersion.ApiLevel": "34.0",
		"SystemImage.Abi":         hostABI,
		"SystemImage.TagId":       "google_apis",
	}, false)
	writeImage(t, sdk, "android-33", "google_apis", otherABI, map[string]string{
		"AndroidVersion.ApiLevel": "33.0",
		"SystemImage.Abi":         otherABI,
		"SystemImage.TagId":       "google_apis",
	}, true)

	images, err := FindImages(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 1 {
		t.Fatalf("FindImages() returned %d images, want 1: %+v", len(images), images)
	}
	if images[0].APIMajor != 35 || images[0].PageSizeKB != 4 {
		t.Fatalf("FindImages() = %+v, want API 35 and 4 KB pages", images[0])
	}

	if images, _ := FindImages(34); len(images) != 0 {
		t.Fatalf("FindImages(34) = %+v, want no complete image", images)
	}
	if images, _ := FindImages(35); len(images) != 1 {
		t.Fatalf("FindImages(35) = %+v, want one image", images)
	}
}

func TestInspectImagePageSizeAndFloor(t *testing.T) {
	sdk := useSDK(t)
	dir := writeImage(t, sdk, "android-37.2-beta3", "google_apis_playstore_ps16k", HostABI(), map[string]string{
		"AndroidVersion.ApiLevel": "37.1",
		"SystemImage.Abi":         HostABI(),
		"SystemImage.TagId":       "google_apis_playstore,page_size_16kb",
		"SystemImage.TagDisplay":  "Google APIs PlayStore, Page Size 16KB",
		"Pkg.Desc":                "System Image Page Size 16k arm64-v8a with Google Play.",
	}, true)

	image, err := InspectImage(dir)
	if err != nil {
		t.Fatal(err)
	}
	if image.PageSizeKB != 16 {
		t.Fatalf("PageSizeKB = %d, want 16", image.PageSizeKB)
	}
	if image.APIMajor != 37 {
		t.Fatalf("APIMajor = %d, want 37", image.APIMajor)
	}
	if !image.Complete {
		t.Fatal("Complete = false, want true")
	}
	if !strings.Contains(image.PkgID, "system-images;android-37.2-beta3;google_apis_playstore_ps16k;") {
		t.Fatalf("PkgID = %q", image.PkgID)
	}
	if floor := image.EffectiveRAMFloor(); floor != 4096 {
		t.Fatalf("EffectiveRAMFloor() = %d, want 4096", floor)
	}
	if got, want := image.ShortLabel(), "37/playstore+16KB"; got != want {
		t.Fatalf("ShortLabel() = %q, want %q", got, want)
	}
}

func TestInspectImageIncomplete(t *testing.T) {
	sdk := useSDK(t)
	dir := writeImage(t, sdk, "android-34", "google_apis", HostABI(), map[string]string{
		"AndroidVersion.ApiLevel": "34.0",
		"SystemImage.Abi":         HostABI(),
		"SystemImage.TagId":       "google_apis",
	}, false)

	image, err := InspectImage(dir)
	if err != nil {
		t.Fatal(err)
	}
	if image.Complete {
		t.Fatal("Complete = true, want false for a stub image")
	}
	if floor := image.EffectiveRAMFloor(); floor != 0 {
		t.Fatalf("EffectiveRAMFloor() = %d, want 0", floor)
	}
}

func TestImageForAVD(t *testing.T) {
	sdk := useSDK(t)
	abi := HostABI()
	writeImage(t, sdk, "android-35", "google_apis", abi, map[string]string{
		"AndroidVersion.ApiLevel": "35.0",
		"SystemImage.Abi":         abi,
		"SystemImage.TagId":       "google_apis",
	}, true)

	path := filepath.Join(t.TempDir(), "config.ini")
	content := "image.sysdir.1=system-images/android-35/google_apis/" + abi + "/\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	config, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}

	image, err := ImageForAVD(config)
	if err != nil {
		t.Fatal(err)
	}
	if image.APIMajor != 35 || image.ABI != abi {
		t.Fatalf("ImageForAVD() = %+v", image)
	}
	if got, want := image.ShortLabel(), "35/google_apis"; got != want {
		t.Fatalf("ShortLabel() = %q, want %q", got, want)
	}
}
