package logtui

import (
	"testing"
)

func TestParseLogLine(t *testing.T) {
	line := "09-30 17:26:34.145   649   806 D WifiScoreCard: txRate: 1 txSpeed: 1"
	entry := ParseLogLine(line)
	if entry == nil {
		t.Fatalf("expected parsed entry, got nil")
	}

	if entry.Date != "09-30" {
		t.Errorf("expected date 09-30, got %s", entry.Date)
	}
	if entry.Timestamp != "17:26:34.145" {
		t.Errorf("expected timestamp 17:26:34.145, got %s", entry.Timestamp)
	}
	if entry.PID != "649" {
		t.Errorf("expected PID 649, got %s", entry.PID)
	}
	if entry.TID != "806" {
		t.Errorf("expected TID 806, got %s", entry.TID)
	}
	if entry.Level != "D" {
		t.Errorf("expected level D, got %s", entry.Level)
	}
	if entry.Tag != "WifiScoreCard" {
		t.Errorf("expected tag WifiScoreCard, got %s", entry.Tag)
	}
	if entry.Message != "txRate: 1 txSpeed: 1" {
		t.Errorf("expected message 'txRate: 1 txSpeed: 1', got '%s'", entry.Message)
	}
	if entry.IsCrashStart {
		t.Errorf("did not expect crash start on normal debug line")
	}
}

func TestParseLogLineCrashDetection(t *testing.T) {
	crashLine := "08-15 12:34:56.789 12345 12345 E AndroidRuntime: FATAL EXCEPTION: main"
	entry := ParseLogLine(crashLine)
	if entry == nil {
		t.Fatalf("expected parsed entry, got nil")
	}
	if !entry.IsCrashStart {
		t.Errorf("expected IsCrashStart to be true for FATAL EXCEPTION")
	}

	stackLine := "08-15 12:34:56.789 12345 12345 E AndroidRuntime: \tat com.example.MainActivity.onCreate(MainActivity.java:45)"
	stackEntry := ParseLogLine(stackLine)
	if stackEntry == nil {
		t.Fatalf("expected parsed entry, got nil")
	}
	if !stackEntry.IsCrashContinue {
		t.Errorf("expected IsCrashContinue to be true for stack trace line")
	}
	if !stackEntry.IsStackTrace {
		t.Errorf("expected IsStackTrace to be true for stack trace line")
	}
}

func TestTagColorDeterminism(t *testing.T) {
	c1 := TagColor("MyTag")
	c2 := TagColor("MyTag")
	if c1 != c2 {
		t.Errorf("TagColor should be deterministic for the same tag, got %v and %v", c1, c2)
	}

	c3 := TagColor("AnotherTag")
	t.Logf("Tag colors: MyTag=%v, AnotherTag=%v", c1, c3)
}

func TestWordWrap(t *testing.T) {
	msg := "The quick brown fox jumps over the lazy dog"
	chunks := wordWrap(msg, 15)
	if len(chunks) < 3 {
		t.Errorf("expected wrapped chunks, got: %v", chunks)
	}
	for _, c := range chunks {
		if len([]rune(c)) > 15 {
			t.Errorf("chunk exceeds max width: %q", c)
		}
	}
}
