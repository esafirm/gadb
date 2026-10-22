package adb

import (
	"strings"
	"testing"
)

func TestPackageNameWithoutAnalyzerInPath(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	_, err := PackageName("app.apk")
	if err == nil || !strings.Contains(err.Error(), "not in the path") {
		t.Fatalf("expected missing analyzer error, got %v", err)
	}
}
