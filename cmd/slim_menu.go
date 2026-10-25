package cmd

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	pui "github.com/manifoldco/promptui"
)

const (
	slimActionCheck       = "check"
	slimActionCreate      = "create"
	slimActionStart       = "start"
	slimActionOn          = "on"
	slimActionOff         = "off"
	slimActionStatus      = "status"
	slimActionSnapshot    = "snapshot"
	slimActionUnbake      = "unbake"
	slimActionTune        = "tune"
	slimActionRestoreTune = "restore-tune"
	slimActionQuit        = "quit"
)

type slimMenuItem struct {
	Group   string
	Label   string
	Caption string
	Action  string
}

func slimMenuItems() []slimMenuItem {
	return []slimMenuItem{
		{Group: "INSPECT", Label: "🔍 Check AVDs", Caption: "Audit the system image, config.ini and live guest without changing anything.", Action: slimActionCheck},
		{Group: "INSPECT", Label: "📊 Status", Caption: "Show what is applied, whether the guest drifted and whether the golden snapshot is valid.", Action: slimActionStatus},
		{Group: "CREATE & LAUNCH", Label: "🏗️ Create a slim AVD", Caption: "Build a new AVD from the newest complete 4 KB Google APIs image.", Action: slimActionCreate},
		{Group: "CREATE & LAUNCH", Label: "🚀 Start a slim AVD", Caption: "Tune, boot and prune an AVD, restoring the golden snapshot when it is valid.", Action: slimActionStart},
		{Group: "TUNE & RESTORE", Label: "🧹 Slim a running emulator", Caption: "Disable the curated package groups and apply background limits on a running emulator.", Action: slimActionOn},
		{Group: "TUNE & RESTORE", Label: "♻️ Restore a slimmed emulator", Caption: "Re-enable packages, restore settings and put config.ini back.", Action: slimActionOff},
		{Group: "TUNE & RESTORE", Label: "⚙️ Tune host config", Caption: "Set guest RAM, host GPU, audio and camera emulation in config.ini.", Action: slimActionTune},
		{Group: "TUNE & RESTORE", Label: "↩️ Restore host config", Caption: "Put back the config.ini values recorded by the last tune.", Action: slimActionRestoreTune},
		{Group: "SNAPSHOTS", Label: "📸 Bake a golden snapshot", Caption: "Cold boot, apply the same pruning and save gadb_clean for fast boots.", Action: slimActionSnapshot},
		{Group: "SNAPSHOTS", Label: "🗑️ Remove the golden snapshot", Caption: "Delete gadb_clean and clear its record.", Action: slimActionUnbake},
		{Group: "EXIT", Label: "👋 Quit", Caption: "Leave the menu.", Action: slimActionQuit},
	}
}

func runSlimMenu() error {
	actions := map[string]func() error{
		slimActionCheck:       func() error { return slimCheckCmd.RunE(slimCheckCmd, nil) },
		slimActionCreate:      runCreateFromMenu,
		slimActionStart:       func() error { return slimStartCmd.RunE(slimStartCmd, nil) },
		slimActionOn:          func() error { return slimOnCmd.RunE(slimOnCmd, nil) },
		slimActionOff:         func() error { return slimOffCmd.RunE(slimOffCmd, nil) },
		slimActionStatus:      func() error { return slimStatusCmd.RunE(slimStatusCmd, nil) },
		slimActionSnapshot:    func() error { return slimSnapshotCmd.RunE(slimSnapshotCmd, nil) },
		slimActionUnbake:      func() error { return slimUnbakeCmd.RunE(slimUnbakeCmd, nil) },
		slimActionTune:        func() error { return slimTuneCmd.RunE(slimTuneCmd, nil) },
		slimActionRestoreTune: runRestoreTuneFromMenu,
	}

	choice, err := runSlimTUI()
	if err != nil || choice == "" || choice == slimActionQuit {
		return err
	}
	if action := actions[choice]; action != nil {
		return action()
	}
	return nil
}

type slimMenuModel struct {
	items    []slimMenuItem
	selected int
	width    int
	choice   string
}

func newSlimMenuModel() *slimMenuModel {
	return &slimMenuModel{items: slimMenuItems()}
}

func (m *slimMenuModel) Init() tea.Cmd { return nil }

func (m *slimMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			m.selected = (m.selected - 1 + len(m.items)) % len(m.items)
		case "down", "j":
			m.selected = (m.selected + 1) % len(m.items)
		case "enter":
			m.choice = m.items[m.selected].Action
			return m, tea.Quit
		case "q", "esc", "ctrl+c":
			m.choice = slimActionQuit
			return m, tea.Quit
		}
	}
	return m, nil
}

var (
	slimMenuTitleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	slimMenuSelectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	slimMenuItemStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	slimMenuCaptionStyle  = lipgloss.NewStyle().Faint(true)
	slimMenuHelpStyle     = lipgloss.NewStyle().Faint(true)
)

func (m *slimMenuModel) View() string {
	var view strings.Builder
	view.WriteString(slimMenuTitleStyle.Render("gadb slim"))
	view.WriteString("\n\n")

	for i, item := range m.items {
		if i > 0 && item.Group != m.items[i-1].Group {
			view.WriteString("\n")
		}
		label := "  " + item.Label
		if i == m.selected {
			view.WriteString(slimMenuSelectedStyle.Render("▸ " + item.Label))
		} else {
			view.WriteString(slimMenuItemStyle.Render(label))
		}
		view.WriteString("\n")
	}

	view.WriteString("\n")
	caption := m.items[m.selected].Caption
	if m.width > 4 {
		view.WriteString(slimMenuCaptionStyle.Width(m.width - 4).Render(caption))
	} else {
		view.WriteString(slimMenuCaptionStyle.Render(caption))
	}
	view.WriteString("\n\n")
	view.WriteString(slimMenuHelpStyle.Render("↑/↓ move · enter run · q quit"))
	return view.String()
}

func runSlimTUI() (string, error) {
	model := newSlimMenuModel()
	program := tea.NewProgram(model)
	if _, err := program.Run(); err != nil {
		return "", err
	}
	return model.choice, nil
}

func runCreateFromMenu() error {
	prompt := pui.Prompt{Label: "New AVD name"}
	name, err := prompt.Run()
	if err != nil {
		return nil
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	return slimCreateCmd.RunE(slimCreateCmd, []string{name})
}

func runRestoreTuneFromMenu() error {
	slimRestore = true
	defer func() { slimRestore = false }()
	return slimTuneCmd.RunE(slimTuneCmd, nil)
}
