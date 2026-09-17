// Copyright © 2019 Esa Firman esafirm21@gmail.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"strings"

	adb "github.com/esafirm/gadb/adb"
	color "github.com/fatih/color"
	pui "github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

type deviceSetting struct {
	Name        string
	Namespace   string
	Key         string
	Description string
	OnValue     string
	OffValue    string
}

var knownSettings = []deviceSetting{
	{
		Name:        "Don't keep activities",
		Namespace:   "global",
		Key:         "always_finish_activities",
		Description: "Destroy activities as soon as they leave the screen",
		OnValue:     "1",
		OffValue:    "0",
	},
	{
		Name:        "Stay awake while charging",
		Namespace:   "global",
		Key:         "stay_on_while_plugged_in",
		Description: "Keep screen on while plugged in (7 = all sources)",
		OnValue:     "7",
		OffValue:    "0",
	},
	{
		Name:        "Show taps",
		Namespace:   "system",
		Key:         "show_touches",
		Description: "Show visual feedback for taps",
		OnValue:     "1",
		OffValue:    "0",
	},
	{
		Name:        "Pointer location",
		Namespace:   "system",
		Key:         "pointer_location",
		Description: "Show pointer coordinates overlay",
		OnValue:     "1",
		OffValue:    "0",
	},
	{
		Name:        "Window animation scale",
		Namespace:   "global",
		Key:         "window_animation_scale",
		Description: "Window animation scale (toggle 1x / off)",
		OnValue:     "1",
		OffValue:    "0",
	},
	{
		Name:        "Transition animation scale",
		Namespace:   "global",
		Key:         "transition_animation_scale",
		Description: "Transition animation scale (toggle 1x / off)",
		OnValue:     "1",
		OffValue:    "0",
	},
	{
		Name:        "Animator duration scale",
		Namespace:   "global",
		Key:         "animator_duration_scale",
		Description: "Animator duration scale (toggle 1x / off)",
		OnValue:     "1",
		OffValue:    "0",
	},
	{
		Name:        "Auto-rotate screen",
		Namespace:   "system",
		Key:         "accelerometer_rotation",
		Description: "Enable auto-rotation",
		OnValue:     "1",
		OffValue:    "0",
	},
	{
		Name:        "Heads-up notifications",
		Namespace:   "global",
		Key:         "heads_up_notifications_enabled",
		Description: "Show pop-up heads-up notifications",
		OnValue:     "1",
		OffValue:    "0",
	},
	{
		Name:        "ADB enabled",
		Namespace:   "global",
		Key:         "adb_enabled",
		Description: "USB debugging state",
		OnValue:     "1",
		OffValue:    "0",
	},
}

// Color styles (padding is applied before coloring so columns stay aligned).
var (
	settingsNameStyle  = color.New(color.FgCyan, color.Bold).SprintFunc()
	settingsKeyStyle   = color.New(color.FgMagenta).SprintFunc()
	settingsDimStyle   = color.New(color.Faint).SprintFunc()
	settingsOnStyle    = color.New(color.FgGreen, color.Bold).SprintFunc()
	settingsOffStyle   = color.New(color.FgRed, color.Bold).SprintFunc()
	settingsWarnStyle  = color.New(color.FgYellow, color.Bold).SprintFunc()
	settingsErrStyle   = color.New(color.FgRed).SprintFunc()
	settingsOkStyle    = color.New(color.FgGreen).SprintFunc()
	settingsTitleStyle = color.New(color.FgHiWhite, color.Bold).SprintFunc()
)

var settingsList bool

var settingsCmd = &cobra.Command{
	Use:   "settings [key-or-name] [on|off|toggle]",
	Short: "Toggle common device settings with a filterable TUI",
	Long: `Toggle common device settings (e.g. Don't keep activities).

Without arguments it opens a filterable interactive list (type to filter,
Tab toggles instantly, Enter opens Toggle / Turn ON / Turn OFF).
With arguments it toggles directly without the TUI:

  gadb settings
  gadb settings --list
  gadb settings always_finish_activities
  gadb settings "don't keep" off
  gadb settings show_touches on`,
	Run: func(cmd *cobra.Command, args []string) {
		if settingsList {
			printSettingsList()
			return
		}
		if len(args) == 0 {
			runSettingsTUI()
			return
		}
		desired := "toggle"
		if len(args) > 1 {
			desired = args[1]
		}
		runSettingsDirect(args[0], desired)
	},
}

func settingsCurrentValue(s deviceSetting) (string, error) {
	res := adb.SettingsGet(s.Namespace, s.Key)
	if res.Error != nil {
		return "", res.Error
	}
	return strings.TrimSpace(string(res.Output)), nil
}

func settingsIsOn(value string) bool {
	v := strings.TrimSpace(value)
	if v == "" || v == "null" {
		return false
	}
	return v != "0" && v != "0.0"
}

func settingsStateLabel(value string, err error) string {
	if err != nil {
		return "?"
	}
	if strings.TrimSpace(value) == "null" || strings.TrimSpace(value) == "" {
		return "unset"
	}
	if settingsIsOn(value) {
		return "ON"
	}
	return "OFF"
}

// settingsBadge returns a colorized state badge using narrow characters only:
// green ON, red OFF, yellow for unset/error. Wide glyphs would break
// promptui's line-based redraw on narrow terminals.
func settingsBadge(value string, err error) string {
	if err != nil {
		return settingsWarnStyle("[ERR]  ")
	}
	v := strings.TrimSpace(value)
	if v == "null" || v == "" {
		return settingsWarnStyle("[UNSET]")
	}
	if settingsIsOn(v) {
		return settingsOnStyle("[ON]   ")
	}
	return settingsOffStyle("[OFF]  ")
}

func settingsToggleValue(s deviceSetting, current string) string {
	if settingsIsOn(current) {
		return s.OffValue
	}
	return s.OnValue
}

func settingsDisplayLine(s deviceSetting, current string, err error) string {
	// Keep rows short: promptui assumes 1 row = 1 terminal line, so a row
	// wider than the terminal wraps and corrupts redraws. Namespace and live
	// value are shown in the details pane instead.
	paddedName := fmt.Sprintf("%-26s", s.Name)
	return fmt.Sprintf("%s %s  %s",
		settingsNameStyle(paddedName),
		settingsBadge(current, err),
		settingsKeyStyle(s.Key),
	)
}

func printSettingsList() {
	fmt.Println(settingsTitleStyle("Device settings:"))
	for _, s := range knownSettings {
		current, err := settingsCurrentValue(s)
		fmt.Println("  " + settingsDisplayLine(s, current, err))
	}
	fmt.Println(settingsDimStyle("Tip: `gadb settings` for the interactive list, `gadb settings <key> on|off|toggle` for direct control."))
}

func matchSettingsIndices(query string) []int {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}
	// Prefer exact key match
	for i, s := range knownSettings {
		if strings.ToLower(s.Key) == q {
			return []int{i}
		}
	}
	matched := []int{}
	for i, s := range knownSettings {
		combined := strings.ToLower(s.Name + " " + s.Key + " " + s.Namespace + " " + s.Description)
		if strings.Contains(combined, q) {
			matched = append(matched, i)
		}
	}
	return matched
}

// settingsApplyAndVerify writes the new value and reads it back, reporting
// the verified end state. It compares on/off state (not raw strings) because
// e.g. animation scales read back as "1.0" after putting "1".
// Used by the direct CLI path where printed feedback is wanted.
func settingsApplyAndVerify(s deviceSetting, newValue string, expectOn bool) {
	fmt.Printf("%s  %s.%s → %s\n",
		settingsNameStyle(s.Name),
		settingsDimStyle(s.Namespace),
		settingsKeyStyle(s.Key),
		settingsDimStyle(newValue),
	)
	res := adb.SettingsPut(s.Namespace, s.Key, newValue)
	if res.Error != nil {
		fmt.Println(settingsErrStyle("✘ Failed:"), res.Error.Error())
		return
	}
	verified, err := settingsCurrentValue(s)
	if err != nil {
		fmt.Println(settingsWarnStyle("⚠ Applied, but could not read back the value."), err.Error())
		return
	}
	verified = strings.TrimSpace(verified)
	if settingsIsOn(verified) == expectOn {
		fmt.Printf("%s %s is now %s (%s)\n",
			settingsOkStyle("✔"),
			settingsNameStyle(s.Name),
			settingsBadge(verified, nil),
			settingsDimStyle(verified),
		)
	} else {
		fmt.Printf("%s %s reads back as %s — device may have ignored the change.\n",
			settingsWarnStyle("⚠"),
			settingsNameStyle(s.Name),
			settingsDimStyle(verified),
		)
	}
}

// settingsApplySilent writes the new value without any success output: the
// TUI list itself shows the updated state on the next refresh. Failures and
// read-back mismatches are still reported so silent toggles never hide errors.
func settingsApplySilent(s deviceSetting, newValue string, expectOn bool) {
	res := adb.SettingsPut(s.Namespace, s.Key, newValue)
	if res.Error != nil {
		fmt.Println(settingsErrStyle("✘ Failed:"), res.Error.Error())
		return
	}
	verified, err := settingsCurrentValue(s)
	if err != nil {
		fmt.Println(settingsWarnStyle("⚠ Applied, but could not read back the value."), err.Error())
		return
	}
	verified = strings.TrimSpace(verified)
	if settingsIsOn(verified) != expectOn {
		fmt.Printf("%s %s reads back as %s — device may have ignored the change.\n",
			settingsWarnStyle("⚠"),
			settingsNameStyle(s.Name),
			settingsDimStyle(verified),
		)
	}
}

// settingsToggleSilent flips the current value without any success output.
func settingsToggleSilent(s deviceSetting) {
	current, err := settingsCurrentValue(s)
	if err != nil {
		fmt.Println(settingsErrStyle("✘ Could not read current value:"), err.Error())
		return
	}
	current = strings.TrimSpace(current)
	settingsApplySilent(s, settingsToggleValue(s, current), !settingsIsOn(current))
}

func runSettingsDirect(query string, desired string) {
	matched := matchSettingsIndices(query)
	if len(matched) == 0 {
		fmt.Printf("No setting matches %q. Available keys:\n", query)
		for _, s := range knownSettings {
			fmt.Printf("  %s  %s\n", settingsKeyStyle(fmt.Sprintf("%-32s", s.Key)), s.Name)
		}
		return
	}
	if len(matched) > 1 {
		fmt.Printf("Multiple settings match %q:\n", query)
		for _, i := range matched {
			fmt.Printf("  %s  %s\n", settingsKeyStyle(fmt.Sprintf("%-32s", knownSettings[i].Key)), knownSettings[i].Name)
		}
		fmt.Println("Be more specific (use the full key).")
		return
	}
	s := knownSettings[matched[0]]
	current, _ := settingsCurrentValue(s)

	var newValue string
	var expectOn bool
	switch strings.ToLower(strings.TrimSpace(desired)) {
	case "", "toggle":
		newValue = settingsToggleValue(s, current)
		expectOn = !settingsIsOn(current)
	case "on", "1", "true", "enable", "enabled":
		newValue = s.OnValue
		expectOn = true
	case "off", "0", "false", "disable", "disabled":
		newValue = s.OffValue
		expectOn = false
	default:
		fmt.Printf("Unknown value %q. Use on|off|toggle.\n", desired)
		return
	}

	settingsApplyAndVerify(s, newValue, expectOn)
}

// settingsTUIItem is the row model for the filterable list. Templates render
// .Label (pre-colorized) and .Details shows key/description/live value.
type settingsTUIItem struct {
	Label    string
	Setting  deviceSetting
	Current  string
	IsExit   bool
	HasError bool
}

// settingsSelectTemplates holds the promptui templates for the settings
// browser. Kept in one place so tests can execute the exact same templates
// (a template referencing an unexported field fails at runtime and promptui
// silently falls back to dumping the whole item, which breaks redraws).
func settingsSelectTemplates() *pui.SelectTemplates {
	return &pui.SelectTemplates{
		Label:    "{{ . }}:",
		Active:   fmt.Sprintf("%s {{ .Label | underline }}", pui.IconSelect),
		Inactive: "  {{ .Label }}",
		Selected: fmt.Sprintf(`{{ %q | green }} {{ .Label | faint }}`, pui.IconGood),
		// No Details pane: the row itself (Name + [ON]/[OFF] badge + key)
		// is the whole UI, so Tab toggling only changes the toggle text.
		Details: "",
		Help:    `{{ "Tab toggles instantly" | faint }} {{ "• ↑/↓ navigate • type to filter" | faint }}`,
	}
}

func buildSettingsTUIItems() ([]settingsTUIItem, bool) {
	items := make([]settingsTUIItem, 0, len(knownSettings)+1)
	errCount := 0
	for _, s := range knownSettings {
		current, err := settingsCurrentValue(s)
		if err != nil {
			errCount++
		}
		current = strings.TrimSpace(current)
		items = append(items, settingsTUIItem{
			Label:    settingsDisplayLine(s, current, err),
			Setting:  s,
			Current:  current,
			HasError: err != nil,
		})
	}
	if errCount == len(knownSettings) {
		return nil, false
	}
	items = append(items, settingsTUIItem{
		Label:   settingsDimStyle("Exit"),
		Current: "-",
		IsExit:  true,
		Setting: deviceSetting{Name: "Exit", Description: "Leave the settings browser"},
	})
	return items, true
}

func settingsTUISearcher(items []settingsTUIItem) func(input string, index int) bool {
	return func(input string, index int) bool {
		if items[index].IsExit {
			return strings.Contains(strings.ToLower("exit quit leave"), strings.ToLower(input))
		}
		s := items[index].Setting
		combined := strings.ToLower(s.Name + " " + s.Key + " " + s.Namespace + " " + s.Description)
		return strings.Contains(combined, strings.ToLower(input))
	}
}

func runSettingsTUI() {
	cursorPos := 0
	for {
		items, ok := buildSettingsTUIItems()
		if !ok {
			fmt.Println(settingsErrStyle("✘ Could not read device settings. Is a device connected? (try `adb devices`)"))
			return
		}
		if cursorPos < 0 {
			cursorPos = 0
		}
		if cursorPos >= len(items) {
			cursorPos = len(items) - 1
		}

		idx, viaTab, err := runSettingsPicker(
			"Pick a setting — Tab toggles, Enter selects, type to filter",
			items,
			12,
			cursorPos,
			settingsSelectTemplates(),
			settingsTUISearcher(items),
		)
		if err != nil {
			fmt.Println("\n" + settingsDimStyle("Cancelled"))
			return
		}
		cursorPos = idx
		if items[idx].IsExit {
			return
		}

		if viaTab {
			// Instant toggle without the action selector; the refreshed
			// list shows the new state, no success output needed.
			settingsToggleSilent(items[idx].Setting)
			continue
		}

		if runSettingsAction(items[idx]) {
			return
		}
	}
}

// runSettingsAction shows the second step: what to do with the picked setting.
// Returns true when the whole browser should exit.
func runSettingsAction(item settingsTUIItem) (exit bool) {
	s := item.Setting
	// Re-read so the action menu never works on a stale value.
	current, err := settingsCurrentValue(s)
	if err != nil {
		fmt.Println(settingsErrStyle("✘ Could not read current value:"), err.Error())
		return false
	}
	current = strings.TrimSpace(current)
	currentlyOn := settingsIsOn(current)

	toggleDir := settingsOnStyle("ON")
	if currentlyOn {
		toggleDir = settingsOffStyle("OFF")
	}
	actions := []string{
		fmt.Sprintf("Toggle (currently %s -> turn %s)",
			settingsBadge(current, nil), toggleDir),
		fmt.Sprintf("%s Turn %s", settingsOnStyle("[ON]"), settingsOnStyle("ON")),
		fmt.Sprintf("%s Turn %s", settingsOffStyle("[OFF]"), settingsOffStyle("OFF")),
		settingsDimStyle("< Back to list"),
	}

	prompt := pui.Select{
		Label: fmt.Sprintf("%s is %s (%s)",
			settingsNameStyle(s.Name),
			settingsBadge(current, nil),
			settingsDimStyle(s.Namespace+"."+s.Key+" = "+current),
		),
		Items: actions,
		Size:  len(actions),
	}
	choice, _, err := prompt.Run()
	if err != nil {
		fmt.Println(settingsDimStyle("Back to list."))
		return false
	}

	var newValue string
	var expectOn bool
	switch choice {
	case 0:
		newValue = settingsToggleValue(s, current)
		expectOn = !currentlyOn
	case 1:
		newValue = s.OnValue
		expectOn = true
	case 2:
		newValue = s.OffValue
		expectOn = false
	default:
		return false
	}

	// Silent: the refreshed list shows the new state, no success output.
	settingsApplySilent(s, newValue, expectOn)
	return false
}

func init() {
	rootCmd.AddCommand(settingsCmd)
	settingsCmd.Flags().BoolVarP(&settingsList, "list", "l", false, "List current values without opening the TUI")
}
