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

		details := executeTemplate(t, tpls.Details, item)
		lines := strings.Split(details, "\n")
		if len(lines) != 3 {
			t.Errorf("Details template for %q rendered %d lines, want 3: %q", name, len(lines), details)
		}
		if !item.IsExit && !strings.Contains(details, item.Setting.Key) {
			t.Errorf("Details template for %q does not contain the key: %q", name, details)
		}
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
