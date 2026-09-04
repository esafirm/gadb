package cmd

import (
	"strings"
	"testing"
)

func TestExtractCrashesThreadtimeFormat(t *testing.T) {
	logs := "08-15 12:34:56.789 12345 12345 E AndroidRuntime: FATAL EXCEPTION: main\n" +
		"08-15 12:34:56.789 12345 12345 E AndroidRuntime: Process: com.example.app, PID: 12345\n" +
		"08-15 12:34:56.789 12345 12345 E AndroidRuntime: java.lang.NullPointerException: Attempt to invoke virtual method\n" +
		"08-15 12:34:56.789 12345 12345 E AndroidRuntime: \tat com.example.MainActivity.onCreate(MainActivity.java:45)\n" +
		"08-15 12:34:56.789 12345 12345 E AndroidRuntime: \tat android.app.Activity.performCreate(Activity.java:8000)\n" +
		"08-15 12:34:57.000  1000  2000 I ActivityManager: Displayed com.example/.MainActivity\n"
	crashes := extractCrashes(logs)
	if len(crashes) != 1 {
		t.Fatalf("extractCrashes() returned %d crashes, want 1", len(crashes))
	}
	if !strings.Contains(crashes[0].StackTrace, "MainActivity.onCreate") {
		t.Errorf("stack missing frames: %q", crashes[0].StackTrace)
	}
	if crashes[0].ProcessID != "12345" {
		t.Errorf("pid=%q want 12345", crashes[0].ProcessID)
	}
}

func TestExtractCrashesThreadtimeMultiple(t *testing.T) {
	logs := "08-15 12:34:56.789 12345 12345 E AndroidRuntime: FATAL EXCEPTION: main\n" +
		"08-15 12:34:56.789 12345 12345 E AndroidRuntime: Process: com.example.app, PID: 12345\n" +
		"08-15 12:34:56.789 12345 12345 E AndroidRuntime: java.lang.NullPointerException: boom\n" +
		"08-15 12:34:56.789 12345 12345 E AndroidRuntime: \tat com.example.MainActivity.onCreate(MainActivity.java:45)\n" +
		"08-15 12:34:57.000  1000  2000 I ActivityManager: Displayed com.example/.MainActivity\n" +
		"08-15 12:35:00.000 12346 12346 E AndroidRuntime: FATAL EXCEPTION: AsyncTask #1\n" +
		"08-15 12:35:00.000 12346 12346 E AndroidRuntime: Process: com.example.app, PID: 12346\n" +
		"08-15 12:35:00.000 12346 12346 E AndroidRuntime: java.lang.RuntimeException: boom\n" +
		"08-15 12:35:00.000 12346 12346 E AndroidRuntime: \tat com.example.BackgroundTask.doInBackground(BackgroundTask.java:20)\n" +
		"08-15 12:35:01.000  1000  2000 I ActivityManager: Something else\n"
	crashes := extractCrashes(logs)
	if len(crashes) != 2 {
		t.Fatalf("extractCrashes() returned %d crashes, want 2", len(crashes))
	}
}

func TestExtractCrashesNativeThreadtime(t *testing.T) {
	logs := "08-15 12:34:56.789  1000   1000 A DEBUG: *** *** *** *** *** *** *** *** *** *** *** *** *** *** *** ***\n" +
		"08-15 12:34:56.789  1000   1000 A DEBUG: Build fingerprint: 'google/sdk_gphone'\n" +
		"08-15 12:34:56.789  1000   1000 A DEBUG: signal 11 (SIGSEGV), code 1 (SEGV_MAPERR)\n" +
		"08-15 12:34:56.789  1000   1000 A DEBUG: backtrace:\n" +
		"08-15 12:34:56.789  1000   1000 A DEBUG:       #00 pc 0000000000012345  /data/app/libnative.so\n" +
		"08-15 12:34:57.000  1000  2000 I ActivityManager: after crash\n"
	crashes := extractCrashes(logs)
	if len(crashes) == 0 {
		t.Fatalf("extractCrashes() returned 0 crashes, want >=1")
	}
}
