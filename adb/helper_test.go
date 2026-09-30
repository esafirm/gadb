package adb

import (
	"reflect"
	"testing"
)

func TestSplitPackageListCarriageReturn(t *testing.T) {
	// Simulate ADB output with CRLF line endings
	raw := "package:com.google.android.youtube\r\npackage:com.google.android.apps.messaging\r\npackage:com.example.app\r\n\r\n"
	result := splitPackageList(raw, "package:")

	expected := []string{
		"com.google.android.youtube",
		"com.google.android.apps.messaging",
		"com.example.app",
	}

	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %#v, got %#v", expected, result)
	}

	// Verify no \r remains
	for _, p := range result {
		for _, ch := range p {
			if ch == '\r' || ch == '\n' {
				t.Fatalf("package name %q contains newline character", p)
			}
		}
	}
}

func TestSplitAvdListCarriageReturn(t *testing.T) {
	raw := "Pixel_7_Pro_API_34\r\nPixel_6_API_33\r\nINFO | Some emulator info\r\n"
	result := splitAvdList(raw)

	expected := []string{
		"Pixel_7_Pro_API_34",
		"Pixel_6_API_33",
	}

	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %#v, got %#v", expected, result)
	}
}
