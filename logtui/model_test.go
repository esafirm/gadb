package logtui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestModelLifecycleAndKeyEvents(t *testing.T) {
	cfg := Config{
		DeviceSerial: "test-device-1",
		PackageName:  "com.test.app",
		Level:        "ALL",
		AllPackages:  false,
	}

	m := NewModel(cfg)
	m.width = 120
	m.height = 30

	// 1. Initial state checks
	if m.mode != ModeNormal {
		t.Errorf("expected ModeNormal, got %v", m.mode)
	}
	if m.paused {
		t.Errorf("expected not paused initially")
	}

	// 2. Feed log lines via logEntryMsg
	entry1 := ParseLogLine("09-30 17:26:34.145   649   806 D TestTag: normal message from com.test.app")
	updatedM, _ := m.Update(logEntryMsg(entry1))
	m = updatedM.(*Model)

	if len(m.logEntries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(m.logEntries))
	}
	if len(m.filteredIndices) != 1 {
		t.Fatalf("expected 1 filtered index, got %d", len(m.filteredIndices))
	}

	// 3. Test Pause toggle (Space)
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m = updatedM.(*Model)
	if !m.paused {
		t.Errorf("expected paused to be true after pressing space")
	}

	// Line sent while paused should not be added
	entryIgnored := ParseLogLine("09-30 17:26:35.000   649   806 D TestTag: ignored line while paused")
	updatedM, _ = m.Update(logEntryMsg(entryIgnored))
	m = updatedM.(*Model)
	if len(m.logEntries) != 1 {
		t.Errorf("expected logEntries to stay 1 while paused, got %d", len(m.logEntries))
	}

	// Unpause
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m = updatedM.(*Model)
	if m.paused {
		t.Errorf("expected paused to be false after pressing space again")
	}

	// 4. Feed crash event lines
	crashStart := ParseLogLine("09-30 17:26:36.000   649   806 E AndroidRuntime: FATAL EXCEPTION: main com.test.app")
	updatedM, _ = m.Update(logEntryMsg(crashStart))
	m = updatedM.(*Model)

	crashStack := ParseLogLine("09-30 17:26:36.001   649   806 E AndroidRuntime: \tat com.test.app.MainActivity.onCreate(MainActivity.java:10)")
	updatedM, _ = m.Update(logEntryMsg(crashStack))
	m = updatedM.(*Model)

	if len(m.crashes) != 1 {
		t.Fatalf("expected 1 crash event, got %d", len(m.crashes))
	}
	if len(m.crashes[0].Lines) != 2 {
		t.Errorf("expected crash to have 2 lines, got %d", len(m.crashes[0].Lines))
	}

	// 5. Test opening Crash Detail popup ('y')
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = updatedM.(*Model)
	if m.mode != ModeCrashDetail {
		t.Errorf("expected ModeCrashDetail after pressing y, got %v", m.mode)
	}

	// Close popup with Esc
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updatedM.(*Model)
	if m.mode != ModeNormal {
		t.Errorf("expected ModeNormal after pressing Esc in crash detail, got %v", m.mode)
	}

	// 6. Test Level Picker ('L')
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	m = updatedM.(*Model)
	if m.mode != ModeLevelPicker {
		t.Errorf("expected ModeLevelPicker after pressing L, got %v", m.mode)
	}

	// Navigate and select 'E' (Error+)
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updatedM.(*Model)
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updatedM.(*Model)
	if m.mode != ModeNormal {
		t.Errorf("expected ModeNormal after selecting level, got %v", m.mode)
	}

	// 7. Test Help modal ('?')
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = updatedM.(*Model)
	if m.mode != ModeHelp {
		t.Errorf("expected ModeHelp after pressing ?, got %v", m.mode)
	}
	view := m.View()
	if !strings.Contains(view, "Keyboard Shortcuts") {
		t.Errorf("expected help view to contain 'Keyboard Shortcuts', got: %s", view)
	}

	// Close help
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updatedM.(*Model)
	if m.mode != ModeNormal {
		t.Errorf("expected ModeNormal after closing help, got %v", m.mode)
	}

	// 8. Test Package input modal ('p')
	m.installedPackages = []string{"com.example.one", "com.example.two"}
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m = updatedM.(*Model)
	if m.mode != ModePackageInput {
		t.Errorf("expected ModePackageInput after pressing p, got %v", m.mode)
	}
	pkgModalView := m.View()
	if !strings.Contains(pkgModalView, "Target Package Filter") {
		t.Errorf("expected package modal to contain 'Target Package Filter', got: %s", pkgModalView)
	}

	// Type new package name and press Enter
	m.packageInput.SetValue("com.new.app")
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updatedM.(*Model)
	if m.mode != ModeNormal {
		t.Errorf("expected ModeNormal after entering package, got %v", m.mode)
	}
	if m.packageName != "com.new.app" {
		t.Errorf("expected packageName 'com.new.app', got '%s'", m.packageName)
	}

	// 9. Test View() rendering in normal mode (toggle all packages to see lines)
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updatedM.(*Model)

	normalView := m.View()
	if !strings.Contains(normalView, "TestTag") && !strings.Contains(normalView, "AndroidRuntime") {
		t.Errorf("expected view to contain log tags, got:\n%s", normalView)
	}
	if !strings.Contains(normalView, "test-device-1") {
		t.Errorf("expected view to contain device serial, got:\n%s", normalView)
	}

	// 9. Clean up
	m.Cleanup()
}
