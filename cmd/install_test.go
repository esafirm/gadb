package cmd

import "testing"

func TestExtractErrorMessage(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name:   "adb failure",
			output: "Performing Streamed Install\nadb: failed to install app.apk: Failure [INSTALL_FAILED_ALREADY_EXISTS]\n",
			want:   "adb: failed to install app.apk: Failure [INSTALL_FAILED_ALREADY_EXISTS]",
		},
		{
			name:   "pm failure without adb prefix",
			output: "Failure [INSTALL_FAILED_VERSION_DOWNGRADE]",
			want:   "Failure [INSTALL_FAILED_VERSION_DOWNGRADE]",
		},
		{
			name:   "no failure detail",
			output: "Performing Streamed Install\n",
			want:   "",
		},
		{
			name:   "empty output",
			output: "",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractErrorMessage(tt.output); got != tt.want {
				t.Fatalf("extractErrorMessage(%q) = %q, want %q", tt.output, got, tt.want)
			}
		})
	}
}

func TestCanRecoverVersionDowngradeSkipsUnrelatedErrors(t *testing.T) {
	// Should not touch the APK or apkanalyzer when the install failed for a
	// reason other than a downgrade.
	if canRecoverVersionDowngrade("missing.apk", "Failure [INSTALL_FAILED_INSUFFICIENT_STORAGE]") {
		t.Fatal("expected no recovery for an unrelated install failure")
	}
}
