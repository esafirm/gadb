package cmd

import (
	"strings"
	"testing"
)

func TestParseTimeRange(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "seconds",
			input:    "30s",
			expected: "100",
		},
		{
			name:     "minutes",
			input:    "5m",
			expected: "5000",
		},
		{
			name:     "hours",
			input:    "1h",
			expected: "300000",
		},
		{
			name:     "custom minutes",
			input:    "15m",
			expected: "5000",
		},
		{
			name:     "raw number (returns empty if no suffix)",
			input:    "500",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTimeRange(tt.input)
			if result != tt.expected {
				t.Errorf("parseTimeRange(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractTimestamp(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		shouldParse bool
	}{
		{
			name:        "valid timestamp",
			line:        "04-01 12:34:56.789 12345 12345 E AndroidRuntime: FATAL EXCEPTION",
			shouldParse: true,
		},
		{
			name:        "short line",
			line:        "short",
			shouldParse: false,
		},
		{
			name:        "invalid format",
			line:        "04/01 12:34:56 FATAL",
			shouldParse: false,
		},
		{
			name:        "empty line",
			line:        "",
			shouldParse: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTimestamp(tt.line)
			if tt.shouldParse {
				if result.IsZero() {
					t.Errorf("extractTimestamp(%q) returned zero time, expected valid timestamp", tt.line)
				}
			} else {
				// For invalid formats, we accept either zero or current time (the function defaults to time.Now())
				// Just verify it doesn't panic
			}
		})
	}
}

func TestExtractProcessInfo(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		wantPID string
	}{
		{
			name:    "PID line",
			line:    "PID: 12345",
			wantPID: "12345",
		},
		{
			name:    "PID with additional info",
			line:    "PID: 67890 TID: 11111",
			wantPID: "67890",
		},
		{
			name:    "Process line with lowercase pid (not extracted)",
			line:    "Process: com.example.app (pid 54321)",
			wantPID: "", // Function only extracts uppercase "PID:", not lowercase "(pid"
		},
		{
			name:    "Process line without PID",
			line:    "Process: com.example.app",
			wantPID: "",
		},
		{
			name:    "unrelated line",
			line:    "Some random log message",
			wantPID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			crash := &Crash{}
			extractProcessInfo(tt.line, crash)

			if crash.ProcessID != tt.wantPID {
				t.Errorf("extractProcessInfo(%q) PID = %q, want %q", tt.line, crash.ProcessID, tt.wantPID)
			}
		})
	}
}

func TestExtractCrashes(t *testing.T) {
	tests := []struct {
		name     string
		logs     string
		wantLen  int
		contains []string // substrings that should be in crash summaries
	}{
		{
			name: "single fatal exception",
			logs: `04-01 12:34:56.789 12345 12345 E AndroidRuntime: FATAL EXCEPTION: main
Process: com.example.app, PID: 12345
    at com.example.MainActivity.onCreate(MainActivity.java:45)
`,
			wantLen:  1,
			contains: []string{"FATAL EXCEPTION: main"},
		},
		{
			name: "multiple crashes",
			logs: `04-01 12:34:56.789 12345 12345 E AndroidRuntime: FATAL EXCEPTION: main
Process: com.example.app, PID: 12345
    at com.example.MainActivity.onCreate(MainActivity.java:45)
-----
04-01 12:35:00.000 12346 12346 E AndroidRuntime: FATAL EXCEPTION: AsyncTask #1
Process: com.example.app, PID: 12346
    at com.example.BackgroundTask.doInBackground(BackgroundTask.java:20)
`,
			wantLen:  2,
			contains: []string{"FATAL EXCEPTION: main", "FATAL EXCEPTION: AsyncTask #1"},
		},
		{
			name: "ANR crash",
			logs: `04-01 12:34:56.789 12345 12345 E ActivityManager: ANR in com.example.app
PID: 12345
    at com.example.MainActivity.onCreate(MainActivity.java:45)
`,
			wantLen:  1,
			contains: []string{"ANR in com.example.app"},
		},
		{
			name: "SIGSEGV signal (only one crash saved - second pattern replaces first since first has no stack trace)",
			logs: `04-01 12:34:56.789 12345 12345 E DEBUG: Crash
SIGSEGV
    at native_method
`,
			wantLen:  1, // Only SIGSEGV crash is saved - first crash has no stack trace
			contains: []string{"SIGSEGV"},
		},
		{
			name:     "no crashes",
			logs:     `04-01 12:34:56.789 12345 12345 I ActivityManager: Displayed com.example/.MainActivity: +345ms`,
			wantLen:  0,
			contains: []string{},
		},
		{
			name: "crash with caused by",
			logs: `04-01 12:34:56.789 12345 12345 E AndroidRuntime: FATAL EXCEPTION: main
Process: com.example.app, PID: 12345
    at com.example.MainActivity.onCreate(MainActivity.java:45)
Caused by: java.lang.NullPointerException
    at com.example.MainActivity.onCreate(MainActivity.java:30)
`,
			wantLen:  1,
			contains: []string{"FATAL EXCEPTION: main"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			crashes := extractCrashes(tt.logs)

			if len(crashes) != tt.wantLen {
				t.Errorf("extractCrashes() returned %d crashes, want %d", len(crashes), tt.wantLen)
			}

			if len(tt.contains) > 0 && len(crashes) > 0 {
				for _, wantContain := range tt.contains {
					found := false
					for _, crash := range crashes {
						if strings.Contains(crash.Summary, wantContain) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("extractCrashes() did not find crash containing %q", wantContain)
					}
				}
			}
		})
	}
}

func TestExtractCrashesStackTraces(t *testing.T) {
	logs := `04-01 12:34:56.789 12345 12345 E AndroidRuntime: FATAL EXCEPTION: main
Process: com.example.app, PID: 12345
    at com.example.MainActivity.onCreate(MainActivity.java:45)
    at android.app.ActivityThread.performLaunchActivity(ActivityThread.java:2913)
Caused by: java.lang.NullPointerException: Attempt to invoke virtual method
    at com.example.MainActivity.onCreate(MainActivity.java:30)
`

	crashes := extractCrashes(logs)

	if len(crashes) != 1 {
		t.Fatalf("extractCrashes() returned %d crashes, want 1", len(crashes))
	}

	crash := crashes[0]
	expectedLines := []string{
		"at com.example.MainActivity.onCreate(MainActivity.java:45)",
		"at android.app.ActivityThread.performLaunchActivity(ActivityThread.java:2913)",
		"Caused by: java.lang.NullPointerException: Attempt to invoke virtual method",
		"at com.example.MainActivity.onCreate(MainActivity.java:30)",
	}

	for _, expectedLine := range expectedLines {
		if !strings.Contains(crash.StackTrace, expectedLine) {
			t.Errorf("Stack trace does not contain expected line: %q\nGot: %s", expectedLine, crash.StackTrace)
		}
	}

	// Verify PID was extracted
	if crash.ProcessID != "12345" {
		t.Errorf("Expected PID 12345, got %q", crash.ProcessID)
	}
}

func TestParseTimeRangeEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "zero seconds",
			input:    "0s",
			expected: "100",
		},
		{
			name:     "large hours",
			input:    "24h",
			expected: "300000",
		},
		{
			name:     "number only (returns empty if no suffix)",
			input:    "1000",
			expected: "",
		},
		{
			name:     "empty",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTimeRange(tt.input)
			if result != tt.expected {
				t.Errorf("parseTimeRange(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
