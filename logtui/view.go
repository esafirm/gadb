package logtui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	tsWidth     = 12
	tagWidth    = 23
	levelWidth  = 3
	prefixWidth = tsWidth + 1 + tagWidth + 1 + levelWidth + 1 // 41
)

// Styles
var (
	headerStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#0E0E16")).
			Foreground(lipgloss.Color("#B4E6FF"))

	headerDimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#505064"))

	headerMutedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#646982"))

	headerSep = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#2D2D3E")).
			Render("  │  ")

	footerBlockStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), true, false, false, false).
				BorderForeground(lipgloss.Color("#565F89"))

	statusDimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C7086"))

	crashBadgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF7878")).
			Bold(true)

	toastInfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAB387")).
			Bold(true)

	toastErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F38BA8")).
			Bold(true)

	// Crash row styles
	crashBg      = lipgloss.Color("#400000")
	crashFg      = lipgloss.Color("#FFC8C8")
	crashTagFg   = lipgloss.Color("#FFB4B4")
	crashTagBg   = lipgloss.Color("#500000")
	crashBadgeBg = lipgloss.Color("#780000")
)

// renderLevelBadge formats a 3-character level badge with appropriate colors
func renderLevelBadge(level string, isCrash bool) string {
	lvl := fmt.Sprintf(" %-1s ", level)
	if isCrash {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(crashBadgeBg).
			Bold(true).
			Render(lvl)
	}

	switch level {
	case "E", "F", "A":
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#FF4D4D")).
			Bold(true).
			Render(lvl)
	case "W":
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#FFC83B")).
			Bold(true).
			Render(lvl)
	case "I":
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#51CF66")).
			Bold(true).
			Render(lvl)
	case "D":
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#339AF0")).
			Bold(true).
			Render(lvl)
	default:
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Render(lvl)
	}
}

// formatTag pads/truncates a tag to exactly tagWidth characters, right-aligned
func formatTag(tag string) string {
	runes := []rune(tag)
	if len(runes) > tagWidth {
		return string(runes[:tagWidth])
	}
	return fmt.Sprintf("%*s", tagWidth, tag)
}

// formatTimestamp pads/truncates timestamp to exactly tsWidth
func formatTimestamp(ts string) string {
	runes := []rune(ts)
	if len(runes) > tsWidth {
		return string(runes[:tsWidth])
	}
	return fmt.Sprintf("%-*s", tsWidth, ts)
}

// wordWrap splits a message into chunks that fit within maxWidth
func wordWrap(msg string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{msg}
	}
	if len([]rune(msg)) <= maxWidth {
		return []string{msg}
	}

	var lines []string
	rem := msg

	for len(rem) > 0 {
		runes := []rune(rem)
		if len(runes) <= maxWidth {
			lines = append(lines, rem)
			break
		}

		// Look for last space before maxWidth
		chunk := string(runes[:maxWidth])
		lastSpace := strings.LastIndex(chunk, " ")
		if lastSpace > 0 {
			lines = append(lines, strings.TrimRight(rem[:lastSpace], " "))
			rem = strings.TrimLeft(rem[lastSpace:], " ")
		} else {
			// Hard break
			lines = append(lines, string(runes[:maxWidth]))
			rem = string(runes[maxWidth:])
		}
	}

	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

// renderKeyCap renders a pastel key button with label
func renderKeyCap(key, label string, bgHex string) string {
	cap := lipgloss.NewStyle().
		Background(lipgloss.Color(bgHex)).
		Foreground(lipgloss.Color("#000000")).
		Bold(true).
		Padding(0, 1).
		Render(key)
	lbl := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#CDD6F4")).
		Render(" " + label + "  ")
	return cap + lbl
}

type keyCapDef struct {
	key      string
	label    string
	bgHex    string
	priority int // 1 = core/essential, 2 = important, 3 = secondary, 4 = extra
}

var allKeyCaps = []keyCapDef{
	{key: "Space", label: "pause", bgHex: "#89B4FA", priority: 1},
	{key: "f", label: "filter", bgHex: "#A6E3A1", priority: 1},
	{key: "x", label: "exclude", bgHex: "#FAB387", priority: 3},
	{key: "L", label: "level", bgHex: "#CBA6F7", priority: 2},
	{key: "a", label: "all", bgHex: "#89DCEB", priority: 3},
	{key: "p", label: "pkg", bgHex: "#B4BEFE", priority: 1},
	{key: "c", label: "clear", bgHex: "#F5C2E7", priority: 4},
	{key: "m", label: "mouse", bgHex: "#E5C07B", priority: 4},
	{key: "y", label: "crash", bgHex: "#F38BA8", priority: 2},
	{key: "w", label: "export", bgHex: "#94E2D5", priority: 4},
	{key: "?", label: "help", bgHex: "#F9E2AF", priority: 1},
	{key: "q", label: "quit", bgHex: "#BAC2DE", priority: 1},
}

// RenderHeader renders the two-line header bar adapted to window width
func RenderHeader(m *Model, width int) string {
	if width <= 0 {
		width = 80
	}

	// Row 1: Device · OS/API · Package · Crashes
	var r1 []string

	// Device icon and model
	devStr := ""
	if m.deviceModel != "" {
		mName := m.deviceModel
		if width < 80 && len(mName) > 18 {
			mName = mName[:15] + "…"
		}
		devStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#64DCFF")).Render("📱 ") +
			lipgloss.NewStyle().Foreground(lipgloss.Color("#B4E6FF")).Bold(true).Render(mName)
	} else if m.deviceSerial != "" {
		sName := m.deviceSerial
		if width < 80 && len(sName) > 18 {
			sName = sName[:15] + "…"
		}
		devStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#B4E6FF")).Bold(true).Render(sName)
	} else {
		devStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD43B")).Render("⚠ No device")
	}
	r1 = append(r1, devStr)

	// OS / API (adapt to terminal width)
	if m.androidRelease != "" || m.sdkVersion != "" {
		if width >= 105 {
			info := ""
			if m.androidRelease != "" && m.sdkVersion != "" {
				info = fmt.Sprintf("Android %s  API %s", m.androidRelease, m.sdkVersion)
			} else if m.androidRelease != "" {
				info = fmt.Sprintf("Android %s", m.androidRelease)
			} else {
				info = fmt.Sprintf("API %s", m.sdkVersion)
			}
			r1 = append(r1, headerMutedStyle.Render(info))
		} else if width >= 80 && m.sdkVersion != "" {
			r1 = append(r1, headerMutedStyle.Render("API "+m.sdkVersion))
		}
	}

	// Package
	if m.allPackages {
		r1 = append(r1, headerDimStyle.Render("all packages"))
	} else if m.packageName != "" {
		pkgDisplay := m.packageName
		maxPkgLen := width / 3
		if maxPkgLen < 15 {
			maxPkgLen = 15
		}
		if len(pkgDisplay) > maxPkgLen {
			pkgDisplay = pkgDisplay[:maxPkgLen-1] + "…"
		}
		pkgStr := lipgloss.NewStyle().Foreground(lipgloss.Color("#64C8E6")).Bold(true).Render(pkgDisplay)
		r1 = append(r1, pkgStr)
	}

	// Crash count
	if len(m.crashes) > 0 {
		cCount := fmt.Sprintf("%d crash", len(m.crashes))
		if len(m.crashes) > 1 {
			cCount += "es"
		}
		r1 = append(r1, crashBadgeStyle.Render(cCount))
	}

	row1Text := "  " + strings.Join(r1, headerSep)
	if lipgloss.Width(row1Text) > width {
		runes := []rune(row1Text)
		if len(runes) > width {
			row1Text = string(runes[:width-1]) + "…"
		}
	}
	pad1 := width - lipgloss.Width(row1Text)
	if pad1 > 0 {
		row1Text += strings.Repeat(" ", pad1)
	}
	row1 := lipgloss.NewStyle().
		Background(lipgloss.Color("#0E0E16")).
		MaxWidth(width).
		Render(row1Text)

	// Row 2: Live status, scroll indicator, active filters, toast
	var r2 []string

	if m.paused {
		r2 = append(r2, lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD43B")).Bold(true).Render("⏸ PAUSED"))
	} else {
		count := len(m.logEntries)
		cStr := fmt.Sprintf("%d lines", count)
		if count >= 1000 {
			cStr = fmt.Sprintf("%.1fk lines", float64(count)/1000.0)
		}
		statStr := lipgloss.NewStyle().Foreground(lipgloss.Color("#64E664")).Render("● ") +
			lipgloss.NewStyle().Foreground(lipgloss.Color("#8CE68C")).Render(cStr)
		r2 = append(r2, statStr)
	}

	if m.scroll > 0 {
		r2 = append(r2, lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD43B")).Render(fmt.Sprintf("↑ scrolled +%d", m.scroll)))
	}

	if m.levelFilter != "ALL" && m.levelFilter != "" {
		r2 = append(r2, lipgloss.NewStyle().Foreground(lipgloss.Color("#F4A261")).Bold(true).Render("level: "+m.levelFilter))
	}

	if m.mode == ModeFilter {
		r2 = append(r2, lipgloss.NewStyle().Foreground(lipgloss.Color("#20C997")).Bold(true).Render("filter: "+m.filterInput.Value()+"█"))
	} else if m.filterInput.Value() != "" {
		r2 = append(r2, lipgloss.NewStyle().Foreground(lipgloss.Color("#20C997")).Render("filter: "+m.filterInput.Value()))
	}

	if m.mode == ModeExclude {
		r2 = append(r2, lipgloss.NewStyle().Foreground(lipgloss.Color("#FF922B")).Bold(true).Render("exclude: "+m.excludeInput.Value()+"█"))
	} else if m.excludeInput.Value() != "" {
		r2 = append(r2, lipgloss.NewStyle().Foreground(lipgloss.Color("#FF922B")).Render("exclude: "+m.excludeInput.Value()))
	}

	if m.toastMsg != "" {
		toast := toastInfoStyle.Render(m.toastMsg)
		if m.toastErr {
			toast = toastErrorStyle.Render(m.toastMsg)
		}
		r2 = append(r2, toast)
	}

	row2Text := "  " + strings.Join(r2, headerSep)
	if lipgloss.Width(row2Text) > width {
		runes := []rune(row2Text)
		if len(runes) > width {
			row2Text = string(runes[:width-1]) + "…"
		}
	}
	pad2 := width - lipgloss.Width(row2Text)
	if pad2 > 0 {
		row2Text += strings.Repeat(" ", pad2)
	}
	row2 := lipgloss.NewStyle().
		Background(lipgloss.Color("#0E0E16")).
		MaxWidth(width).
		Render(row2Text)

	return row1 + "\n" + row2
}

// RenderFooter renders the bottom bar with status and pastel keycaps adapted to width
func RenderFooter(m *Model, width int) string {
	if width <= 0 {
		width = 80
	}

	// Line 1: Status summary
	line1 := ""
	if m.scroll > 0 {
		if m.newLinesWhileScrolled > 0 {
			line1 = fmt.Sprintf("%d logs behind — Press End or G to jump to bottom", m.newLinesWhileScrolled)
		} else {
			line1 = fmt.Sprintf("Scrolled ↑%d — Press End or G to follow tail", m.scroll)
		}
	} else if m.paused {
		line1 = "Logcat PAUSED — Press Space to resume"
	} else {
		line1 = fmt.Sprintf("%d logs captured · %d visible", len(m.logEntries), len(m.filteredIndices))
	}

	maxStatusW := width - 4
	if maxStatusW > 0 && lipgloss.Width(line1) > maxStatusW {
		runes := []rune(line1)
		if len(runes) > maxStatusW {
			line1 = string(runes[:maxStatusW-1]) + "…"
		}
	}
	statusLine := "  " + statusDimStyle.Render(line1)

	// Line 2: Responsive Keycaps
	availW := width - 4
	if availW < 20 {
		availW = 20
	}

	var capsLine string
	// Find the highest priority tier (from 4 down to 1) that fits within availW
	for maxPrio := 4; maxPrio >= 1; maxPrio-- {
		var sb strings.Builder
		sb.WriteString("  ")
		for _, kc := range allKeyCaps {
			if kc.priority <= maxPrio {
				sb.WriteString(renderKeyCap(kc.key, kc.label, kc.bgHex))
			}
		}
		cand := sb.String()
		if lipgloss.Width(cand) <= availW {
			capsLine = cand
			break
		}
	}

	// Fallback if even prio 1 doesn't fit in narrow terminal
	if capsLine == "" {
		var sb strings.Builder
		sb.WriteString("  ")
		sb.WriteString(renderKeyCap("?", "help", "#F9E2AF"))
		sb.WriteString(renderKeyCap("q", "quit", "#BAC2DE"))
		capsLine = sb.String()
	}

	content := statusLine + "\n" + capsLine
	return footerBlockStyle.Width(width).Render(content)
}

// RenderViewportLines renders the rustycat-formatted lines for the visible viewport
func RenderViewportLines(m *Model, startIdx, count, width int) []string {
	var renderedLines []string
	msgWidth := width - prefixWidth
	if msgWidth < 20 {
		msgWidth = 20
	}

	lastTag := ""

	endIdx := startIdx + count
	if endIdx > len(m.filteredIndices) {
		endIdx = len(m.filteredIndices)
	}

	for i := startIdx; i < endIdx; i++ {
		entryIdx := m.filteredIndices[i]
		entry := m.logEntries[entryIdx]

		isCrash := entry.IsCrashStart || entry.IsCrashContinue
		showTag := entry.Tag != lastTag

		tagDisplay := ""
		if showTag {
			tagDisplay = formatTag(entry.Tag)
		} else {
			tagDisplay = strings.Repeat(" ", tagWidth)
		}

		tsDisplay := formatTimestamp(entry.Timestamp)
		lvlBadge := renderLevelBadge(entry.Level, isCrash)

		// Styles
		tsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#505064"))
		tagStyle := lipgloss.NewStyle().Foreground(TagColor(entry.Tag))
		msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#D0D0E0"))

		if isCrash {
			tsStyle = lipgloss.NewStyle().Foreground(crashFg).Background(crashBg)
			tagStyle = lipgloss.NewStyle().Foreground(crashTagFg).Background(crashTagBg).Bold(true)
			msgStyle = lipgloss.NewStyle().Foreground(crashFg).Background(crashBg).Bold(true)
		}

		chunks := wordWrap(entry.Message, msgWidth)
		for ci, chunk := range chunks {
			if ci == 0 {
				line := tsStyle.Render(tsDisplay) + " " +
					tagStyle.Render(tagDisplay) + " " +
					lvlBadge + " " +
					msgStyle.Render(chunk)
				renderedLines = append(renderedLines, line)
			} else {
				// Continuation line indent with prefixWidth spaces
				indent := strings.Repeat(" ", prefixWidth)
				if isCrash {
					indent = lipgloss.NewStyle().Background(crashBg).Render(indent)
				}
				renderedLines = append(renderedLines, indent+msgStyle.Render(chunk))
			}
		}

		lastTag = entry.Tag
	}

	return renderedLines
}

// RenderLevelPickerModal renders the floating level picker overlay
func RenderLevelPickerModal(m *Model, width, height int) string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F4A261")).Render(" Filter Log Levels (↑↓ to select, Enter to apply)\n\n"))

	for i, opt := range LevelFilterOptions {
		cursor := "  "
		if i == m.levelPickerCursor {
			cursor = "❯ "
		}

		active := ""
		if opt.Code == m.levelFilter {
			active = " (active)"
		}

		item := fmt.Sprintf("%s%s%s", cursor, opt.Title, active)
		if i == m.levelPickerCursor {
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#000000")).Background(lipgloss.Color("#F4A261")).Bold(true).Render(item) + "\n")
		} else if opt.Code == m.levelFilter {
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#51CF66")).Render(item) + "\n")
		} else {
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#E0E0E0")).Render(item) + "\n")
		}
	}

	modalBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#F4A261")).
		Padding(1, 2).
		Width(48).
		Render(b.String())

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, modalBox)
}

// RenderCrashPopup renders the floating crash detail popup
func RenderCrashPopup(m *Model, width, height int) string {
	if len(m.crashes) == 0 {
		return ""
	}

	crash := m.crashes[len(m.crashes)-1]

	popupW := int(float64(width) * 0.85)
	if popupW < 50 {
		popupW = 50
	}
	if popupW > width-4 {
		popupW = width - 4
	}

	popupH := int(float64(height) * 0.75)
	if popupH < 12 {
		popupH = 12
	}
	if popupH > height-4 {
		popupH = height - 4
	}

	var content strings.Builder
	title := fmt.Sprintf(" Crash Detail — %s @ %s ", crash.Summary, crash.Timestamp)
	if len(title) > popupW-6 {
		title = title[:popupW-9] + "… "
	}
	content.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF7878")).Render(title) + "\n\n")

	// Stack trace lines
	allLines := crash.Lines
	visibleH := popupH - 6
	if visibleH < 3 {
		visibleH = 3
	}

	start := m.crashScroll
	if start > len(allLines)-visibleH {
		start = len(allLines) - visibleH
	}
	if start < 0 {
		start = 0
	}

	end := start + visibleH
	if end > len(allLines) {
		end = len(allLines)
	}

	maxLineLen := popupW - 8
	if maxLineLen < 20 {
		maxLineLen = 20
	}

	for i := start; i < end; i++ {
		l := allLines[i]
		raw := strings.TrimRight(l.Raw, "\r\n")
		if len(raw) > maxLineLen {
			raw = raw[:maxLineLen-1] + "…"
		}

		lineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#CCCCCC"))
		if strings.Contains(strings.ToLower(raw), "fatal exception") || strings.Contains(strings.ToLower(raw), "caused by:") {
			lineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF7878")).Bold(true)
		} else if strings.HasPrefix(strings.TrimSpace(l.Message), "at ") {
			lineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD43B"))
		}
		content.WriteString(lineStyle.Render(raw) + "\n")
	}

	// Footer inside popup
	var footer strings.Builder
	footer.WriteString("\n")
	if popupW >= 75 {
		footer.WriteString(renderKeyCap("c", "copy stack", "#F3BCDB"))
		footer.WriteString(renderKeyCap("a", "agent prompt", "#A6E3A1"))
		footer.WriteString(renderKeyCap("w", "export", "#89B4FA"))
		footer.WriteString(renderKeyCap("s", "search web", "#F9E2AF"))
		footer.WriteString(renderKeyCap("Esc", "close", "#BAC2DE"))
	} else {
		footer.WriteString(renderKeyCap("c", "copy", "#F3BCDB"))
		footer.WriteString(renderKeyCap("a", "ai", "#A6E3A1"))
		footer.WriteString(renderKeyCap("Esc", "close", "#BAC2DE"))
	}
	content.WriteString(footer.String())

	modalBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#F38BA8")).
		Padding(1, 2).
		Width(popupW).
		Height(popupH).
		Render(content.String())

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, modalBox)
}

// RenderHelpModal renders the cheat sheet overlay
func RenderHelpModal(width, height int) string {
	boxW := 70
	if boxW > width-4 {
		boxW = width - 4
	}
	if boxW < 36 {
		boxW = 36
	}

	shortcuts := [][]string{
		{"Space", "Pause / resume logcat streaming"},
		{"↑ / k", "Scroll up one line"},
		{"↓ / j", "Scroll down one line"},
		{"PgUp / PgDn", "Scroll 20 lines"},
		{"End / G", "Jump to bottom (resume auto-tail)"},
		{"mouse wheel", "Scroll (only when mouse capture is ON, see m)"},
		{"m", "Toggle mouse capture (OFF = drag to select & copy)"},
		{"f", "Include filter (type to search tag & message)"},
		{"x", "Exclude filter (type to hide noise)"},
		{"L", "Log level picker (All, D, I, W, E)"},
		{"a", "Toggle all packages vs target package"},
		{"p", "Set or pick target package name"},
		{"c", "Clear log buffer (and device logcat)"},
		{"w", "Export captured logs to file"},
		{"y", "Open Crash & ANR detail popup"},
		{"?", "Toggle this help menu"},
		{"q / Ctrl+C", "Quit gadb log"},
	}

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F9E2AF")).Render(" gadb log — Keyboard Shortcuts\n\n"))

	keyW := 14
	if boxW < 48 {
		keyW = 10
	}
	for _, s := range shortcuts {
		k := lipgloss.NewStyle().Foreground(lipgloss.Color("#A6E3A1")).Bold(true).Width(keyW).Render(s[0])
		desc := lipgloss.NewStyle().Foreground(lipgloss.Color("#E0E0E0")).Render(s[1])
		b.WriteString(k + desc + "\n")
	}

	b.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("Copy: drag to select + Cmd/Ctrl+C (mouse OFF) · w exports · y,c copies crash"))
	b.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("Press Esc or ? to close this dialog"))

	modalBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#F9E2AF")).
		Padding(1, 2).
		Width(boxW).
		Render(b.String())

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, modalBox)
}

// RenderPackageModal renders the interactive package input and suggestion dialog
func RenderPackageModal(m *Model, width, height int) string {
	boxW := 60
	if boxW > width-4 {
		boxW = width - 4
	}
	if boxW < 36 {
		boxW = 36
	}
	innerWidth := boxW - 6
	if innerWidth < 20 {
		innerWidth = 20
	}

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#64DCFF")).Render(" Target Package Filter") + "\n\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#AAAAAA")).Render("Type package name or select from device:") + "\n")
	b.WriteString(m.packageInput.View() + "\n\n")

	if len(m.filteredPackages) > 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086")).Render("Installed packages:") + "\n")
		maxShow := 8
		if len(m.filteredPackages) < maxShow {
			maxShow = len(m.filteredPackages)
		}
		for i := 0; i < maxShow; i++ {
			p := strings.TrimSpace(strings.ReplaceAll(m.filteredPackages[i], "\r", ""))
			cursor := "  "
			if i == m.packageCursor {
				cursor = "❯ "
			}

			// Truncate p if too long to prevent wrapping inside box
			maxItemLen := innerWidth - 4
			displayPkg := p
			if len(displayPkg) > maxItemLen && maxItemLen > 5 {
				displayPkg = displayPkg[:maxItemLen-1] + "…"
			}

			lineText := cursor + displayPkg
			if i == m.packageCursor {
				b.WriteString(lipgloss.NewStyle().
					Foreground(lipgloss.Color("#000000")).
					Background(lipgloss.Color("#64DCFF")).
					Bold(true).
					Width(innerWidth).
					Render(lineText) + "\n")
			} else {
				b.WriteString(lipgloss.NewStyle().
					Foreground(lipgloss.Color("#E0E0E0")).
					Render(lineText) + "\n")
			}
		}
	} else {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086")).Render("(Press Enter with empty text to clear package filter)") + "\n")
	}

	hint := "[Enter] Apply  [Tab] Select  [↑↓] Browse  [Esc] Cancel"
	if innerWidth < 50 {
		hint = "[Enter] Apply  [Tab] Pick  [Esc] Cancel"
	}
	b.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(hint))

	modalBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#64DCFF")).
		Padding(1, 2).
		Width(boxW).
		Render(b.String())

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, modalBox)
}
