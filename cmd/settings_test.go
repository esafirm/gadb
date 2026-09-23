package cmd

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
	"text/template"

	pui "github.com/manifoldco/promptui"
)

var ansiPattern = regexp.MustCompile("\x1b\\[[0-9;]*m")

func stripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

func executeTemplate(t *testing.T, text string, data interface{}) string {
	t.Helper()
	tpl, err := template.New("").Funcs(pui.FuncMap).Parse(text)
	if err != nil {
		t.Fatalf("template parse failed: %v", err)
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		t.Fatalf("template execute failed for %v: %v", data, err)
	}
	return buf.String()
}

// TestSettingsTemplates ensures the TUI templates render against the real
// item types. text/template cannot read unexported struct fields: execution
// fails and promptui silently falls back to dumping the whole item with %v,
// producing ultra-long lines that corrupt terminal redraws (stacked Search:
// lines). This test fails loudly instead.
func TestSettingsTemplates(t *testing.T) {
	tpls := settingsSelectTemplates()

	items := []settingsTUIItem{}
	for _, s := range knownSettings {
		items = append(items, settingsTUIItem{
			Label:   settingsDisplayLine(s, "1", nil),
			Setting: s,
			Current: "1",
		})
	}
	items = append(items, settingsTUIItem{
		Label:   "Exit",
		Current: "-",
		IsExit:  true,
		Setting: deviceSetting{Name: "Exit", Description: "Leave the settings browser"},
	})

	for _, item := range items {
		name := item.Setting.Key
		if item.IsExit {
			name = "exit"
		}
		for _, tplName := range []string{"Active", "Inactive", "Selected"} {
			var text string
			switch tplName {
			case "Active":
				text = tpls.Active
			case "Inactive":
				text = tpls.Inactive
			case "Selected":
				text = tpls.Selected
			}
			out := executeTemplate(t, text, item)
			if strings.Contains(out, "\n") {
				t.Errorf("%s template for %q rendered multiple lines: %q", tplName, name, out)
			}
			if !strings.Contains(stripANSI(out), stripANSI(item.Label)) {
				t.Errorf("%s template for %q lost the label: %q", tplName, name, out)
			}
		}

		details := ""
		if tpls.Details != "" {
			details = executeTemplate(t, tpls.Details, item)
			lines := strings.Split(details, "\n")
			if len(lines) != 3 {
				t.Errorf("Details template for %q rendered %d lines, want 3: %q", name, len(lines), details)
			}
			if !item.IsExit && !strings.Contains(details, item.Setting.Key) {
				t.Errorf("Details template for %q does not contain the key: %q", name, details)
			}
		} else if compiled, err := compileSettingsTemplates(tpls); err != nil {
			t.Errorf("compileSettingsTemplates failed: %v", err)
		} else if compiled.details != nil {
			t.Errorf("Details template should be empty so only the toggle text changes, got %q", tpls.Details)
		}
		_ = details
	}
}

// TestSettingsDisplayLineWidth guards promptui's assumption that one row is
// one terminal line: rows wider than the terminal wrap and corrupt redraws.
func TestSettingsDisplayLineWidth(t *testing.T) {
	const maxWidth = 70
	for _, s := range knownSettings {
		line := stripANSI(settingsDisplayLine(s, "1.0", nil))
		if len([]rune(line)) > maxWidth {
			t.Errorf("display line for %q is %d cells wide, max %d: %q",
				s.Key, len([]rune(line)), maxWidth, line)
		}
		if strings.Contains(line, "\n") {
			t.Errorf("display line for %q contains a newline", s.Key)
		}
	}
}

// TestSettingsTabHint ensures the TUI advertises the Tab-to-toggle shortcut
// in both the picker label path (Help template) and the compiled templates.
func TestSettingsTabHint(t *testing.T) {
	tpls := settingsSelectTemplates()
	if !strings.Contains(strings.ToLower(tpls.Help), "tab") {
		t.Errorf("Help template should mention Tab, got %q", tpls.Help)
	}
	compiled, err := compileSettingsTemplates(tpls)
	if err != nil {
		t.Fatalf("compileSettingsTemplates failed: %v", err)
	}
	out := stripANSI(string(renderSettingsTpl(compiled.help, struct {
		NextKey     string
		PrevKey     string
		PageDownKey string
		PageUpKey   string
		Search      bool
		SearchKey   string
	}{})))
	if !strings.Contains(strings.ToLower(out), "tab") {
		t.Errorf("rendered help should mention Tab, got %q", out)
	}
}

// TestAirplaneModeSetting verifies that Airplane mode is registered in knownSettings
// and matches queries properly.
func TestAirplaneModeSetting(t *testing.T) {
	indices := matchSettingsIndices("airplane")
	if len(indices) != 1 {
		t.Fatalf("expected 1 match for 'airplane', got %d", len(indices))
	}
	s := knownSettings[indices[0]]
	if s.Key != "airplane_mode_on" {
		t.Errorf("expected Key to be 'airplane_mode_on', got %q", s.Key)
	}
	if s.Name != "Airplane mode" {
		t.Errorf("expected Name to be 'Airplane mode', got %q", s.Name)
	}
	if s.Namespace != "global" {
		t.Errorf("expected Namespace to be 'global', got %q", s.Namespace)
	}
	if s.OnValue != "1" {
		t.Errorf("expected OnValue to be '1', got %q", s.OnValue)
	}
	if s.OffValue != "0" {
		t.Errorf("expected OffValue to be '0', got %q", s.OffValue)
	}

	keyIndices := matchSettingsIndices("airplane_mode_on")
	if len(keyIndices) != 1 || keyIndices[0] != indices[0] {
		t.Errorf("expected exact key match for 'airplane_mode_on' to match the same setting")
	}

	if val := settingsToggleValue(s, "0"); val != "1" {
		t.Errorf("toggling off airplane mode should return '1', got %q", val)
	}
	if val := settingsToggleValue(s, "1"); val != "0" {
		t.Errorf("toggling on airplane mode should return '0', got %q", val)
	}
}

// TestSettingsAlias ensures that 'setting' resolves to settingsCmd as an alias.
func TestSettingsAlias(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"setting"})
	if err != nil {
		t.Fatalf("unexpected error finding 'setting': %v", err)
	}
	if cmd != settingsCmd {
		t.Errorf("expected 'setting' to resolve to settingsCmd, got %v", cmd.Name())
	}
}
