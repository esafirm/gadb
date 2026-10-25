package cmd

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSlimMenuItemsAreGroupedAndDescribed(t *testing.T) {
	items := slimMenuItems()
	if len(items) == 0 {
		t.Fatal("menu has no items")
	}

	seenActions := make(map[string]bool, len(items))
	groupChanges := make(map[string]bool)
	previousGroup := ""
	for _, item := range items {
		if item.Group == "" || item.Caption == "" || item.Action == "" {
			t.Errorf("incomplete menu item: %+v", item)
		}
		if seenActions[item.Action] {
			t.Errorf("duplicate menu action %q", item.Action)
		}
		seenActions[item.Action] = true
		if item.Group != previousGroup {
			if groupChanges[item.Group] {
				t.Errorf("group %q is split into multiple sections", item.Group)
			}
			groupChanges[item.Group] = true
			previousGroup = item.Group
		}
	}

	for _, group := range []string{"INSPECT", "CREATE & LAUNCH", "TUNE & RESTORE", "SNAPSHOTS", "EXIT"} {
		if !groupChanges[group] {
			t.Errorf("missing group %q", group)
		}
	}
}

func TestSlimMenuUsesSpacingAndShowsFocusedCaption(t *testing.T) {
	model := newSlimMenuModel()
	view := model.View()
	before := strings.Index(view, "🚀 Start a slim AVD")
	after := strings.Index(view, "🧹 Slim a running emulator")
	if strings.Contains(view, "INSPECT") || before < 0 || after < 0 ||
		!strings.Contains(view[before:after], "\n\n") {
		t.Fatalf("menu should separate groups with whitespace, not group labels:\n%s", view)
	}

	model.Update(tea.KeyMsg{Type: tea.KeyDown})
	if model.items[model.selected].Action != slimActionStatus {
		t.Fatalf("focused action = %q, want status", model.items[model.selected].Action)
	}
	if view := model.View(); !strings.Contains(view, model.items[model.selected].Caption) {
		t.Fatalf("view does not show focused caption %q", model.items[model.selected].Caption)
	}
}

func TestSlimMenuEnterRunsFocusedAction(t *testing.T) {
	model := newSlimMenuModel()
	model.Update(tea.KeyMsg{Type: tea.KeyDown})
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter did not quit the menu")
	}
	if model.choice != slimActionStatus {
		t.Fatalf("choice = %q, want status", model.choice)
	}
}
