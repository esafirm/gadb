package avd

import (
	"strings"
	"testing"
)

func TestParseMemTotalMB(t *testing.T) {
	cases := map[string]int{
		"MemTotal:        4062432 kB\n": 3967,
		"MemTotal:        2048000 kB\n": 2000,
		"":                              0,
		"garbage":                       0,
	}
	for input, want := range cases {
		if got := parseMemTotalMB(input); got != want {
			t.Errorf("parseMemTotalMB(%q) = %d, want %d", input, got, want)
		}
	}
}

func TestClassifyRenderer(t *testing.T) {
	cases := map[string]string{
		`GLES: Google (Google Inc. (Google)), Android Emulator OpenGL ES Translator (ANGLE (Google, Vulkan 1.3.0 (SwiftShader Device (LLVM 10.0.0)), SwiftShader driver-5.0.0))`: "software",
		`GLES: Google (Google Inc. (Google)), Android Emulator OpenGL ES Translator (ANGLE (Apple, Apple M3, OpenGL 4.1))`:                                                       "hardware",
		`GLES: Mesa/X.org, llvmpipe`: "software",
		``:                           "unknown",
	}
	for input, want := range cases {
		if got := classifyRenderer(input); got != want {
			t.Errorf("classifyRenderer(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestCheckReportEvaluateHighImpact(t *testing.T) {
	report := &CheckReport{
		AVD:      AVD{Name: "Test"},
		Image:    &Image{TagID: "google_apis_playstore,page_size_16kb", PageSizeKB: 16, Complete: true},
		Runtime:  &Runtime{Renderer: "software", MemTotalMB: 3967},
		GPU:      "auto",
		RAMFloor: 4096,
	}
	report.evaluate()

	if report.Verdict != "high" {
		t.Fatalf("Verdict = %q, want high", report.Verdict)
	}
	notes := strings.Join(report.Notes, "\n")
	for _, want := range []string{"16 KB", "software rendering", "Play Store"} {
		if !strings.Contains(notes, want) {
			t.Errorf("notes missing %q:\n%s", want, notes)
		}
	}
	next := strings.Join(report.Next, "\n")
	if !strings.Contains(next, "gadb slim create Test_slim") {
		t.Errorf("next missing the create command:\n%s", next)
	}
	if strings.Contains(next, "gadb slim tune Test") {
		t.Errorf("next suggests tuning an image that cannot reach the target:\n%s", next)
	}
}

func TestCheckReportEvaluateLowImpact(t *testing.T) {
	report := &CheckReport{
		AVD:      AVD{Name: "Test"},
		Image:    &Image{TagID: "google_apis", PageSizeKB: 4, Complete: true},
		Runtime:  &Runtime{Renderer: "hardware", MemTotalMB: 2048},
		Manifest: &Manifest{},
		GPU:      "host",
	}
	report.evaluate()

	if report.Verdict != "low" {
		t.Fatalf("Verdict = %q, want low", report.Verdict)
	}
	notes := strings.Join(report.Notes, "\n")
	if strings.Contains(notes, "software rendering") {
		t.Errorf("low-impact report claims software rendering:\n%s", notes)
	}
}

func TestCheckReadsHostConfigAndImage(t *testing.T) {
	sdk := useSDK(t)
	abi := HostABI()
	writeImage(t, sdk, "android-35", "google_apis", abi, map[string]string{
		"AndroidVersion.ApiLevel": "35.0",
		"SystemImage.Abi":         abi,
		"SystemImage.TagId":       "google_apis",
	}, true)

	a := testAVDWithConfig(t, "hw.ramSize=2048\nhw.gpu.mode=host\nhw.audioOutput=yes\nhw.camera.back=virtualscene\nhw.camera.front=emulated\nimage.sysdir.1=system-images/android-35/google_apis/"+abi+"/\n")
	stubStopped(t)

	report, err := Check(a)
	if err != nil {
		t.Fatal(err)
	}
	if report.RAM != "2048" || report.GPU != "host" {
		t.Fatalf("report RAM/GPU = %q/%q", report.RAM, report.GPU)
	}
	if report.Audio != "on" || report.Camera != "on" {
		t.Fatalf("report audio/camera = %q/%q", report.Audio, report.Camera)
	}
	if report.Image == nil || report.Image.APIMajor != 35 {
		t.Fatalf("report image = %+v", report.Image)
	}
	if report.Verdict != "medium" {
		t.Fatalf("Verdict = %q, want medium for an untuned AVD", report.Verdict)
	}
	if !strings.Contains(strings.Join(report.Next, "\n"), "gadb slim tune Test") {
		t.Fatalf("next does not suggest tuning: %+v", report.Next)
	}
}

func TestCheckIncompleteImageIsHighImpact(t *testing.T) {
	sdk := useSDK(t)
	abi := HostABI()
	writeImage(t, sdk, "android-34", "google_apis", abi, map[string]string{
		"AndroidVersion.ApiLevel": "34.0",
		"SystemImage.Abi":         abi,
		"SystemImage.TagId":       "google_apis",
	}, false)

	a := testAVDWithConfig(t, "hw.ramSize=2048\nhw.gpu.mode=host\nimage.sysdir.1=system-images/android-34/google_apis/"+abi+"/\n")
	stubStopped(t)

	report, err := Check(a)
	if err != nil {
		t.Fatal(err)
	}
	if report.Verdict != "high" {
		t.Fatalf("Verdict = %q, want high", report.Verdict)
	}
	if !strings.Contains(strings.Join(report.Notes, "\n"), "incomplete") {
		t.Fatalf("notes do not mention the incomplete image: %+v", report.Notes)
	}
}
