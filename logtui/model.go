package logtui

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/esafirm/gadb/adb"
)

type Mode int

const (
	ModeNormal Mode = iota
	ModeFilter
	ModeExclude
	ModeLevelPicker
	ModeCrashDetail
	ModeHelp
	ModePackageInput
)

const maxLogBuffer = 20000

// Model is the main Bubble Tea model for the logcat TUI
type Model struct {
	ctx           context.Context
	cancelCtx     context.CancelFunc
	deviceSerial  string
	deviceModel   string
	androidRelease string
	sdkVersion    string
	packageName   string
	allPackages   bool
	pids          map[string]bool

	// Log stream & buffer
	logEntryChan chan *LogEntry
	errChan      chan error
	pidChan      chan []string
	logEntries   []LogEntry
	filteredIndices []int

	// Viewport & Scrolling
	scroll                 int // lines scrolled up from tail (0 = following tail)
	newLinesWhileScrolled  int
	paused                 bool
	width                  int
	height                 int

	// Modes & Modals
	mode             Mode
	filterInput      textinput.Model
	excludeInput     textinput.Model
	packageInput     textinput.Model
	installedPackages []string
	filteredPackages  []string
	packageCursor     int
	levelFilter      string // "ALL", "D", "I", "W", "E"
	levelPickerCursor int

	// Crashes
	crashes      []CrashEvent
	currentCrash *CrashEvent
	crashScroll  int

	// Toasts
	toastMsg string
	toastErr bool

	// Stream cleanup
	streamCancel func()
}

// Config parameters for creating a new Model
type Config struct {
	DeviceSerial string
	PackageName  string
	Level        string
	AllPackages  bool
	FilterQuery  string
	ExcludeQuery string
}

// NewModel creates an initialized logcat TUI model
func NewModel(cfg Config) *Model {
	ctx, cancel := context.WithCancel(context.Background())

	fi := textinput.New()
	fi.Placeholder = "filter query..."
	fi.CharLimit = 100
	if cfg.FilterQuery != "" {
		fi.SetValue(cfg.FilterQuery)
	}

	xi := textinput.New()
	xi.Placeholder = "exclude query..."
	xi.CharLimit = 100
	if cfg.ExcludeQuery != "" {
		xi.SetValue(cfg.ExcludeQuery)
	}

	pi := textinput.New()
	pi.Placeholder = "package name (e.g. com.example.app)"
	pi.CharLimit = 100
	if cfg.PackageName != "" {
		pi.SetValue(cfg.PackageName)
	}

	lvl := strings.ToUpper(cfg.Level)
	if lvl == "" {
		lvl = "ALL"
	}

	m := &Model{
		ctx:             ctx,
		cancelCtx:       cancel,
		deviceSerial:    cfg.DeviceSerial,
		packageName:     cfg.PackageName,
		allPackages:     cfg.AllPackages || cfg.PackageName == "",
		pids:            make(map[string]bool),
		logEntryChan:    make(chan *LogEntry, 2000),
		errChan:         make(chan error, 10),
		pidChan:         make(chan []string, 10),
		logEntries:      make([]LogEntry, 0, 1000),
		filteredIndices: make([]int, 0, 1000),
		filterInput:     fi,
		excludeInput:    xi,
		packageInput:    pi,
		levelFilter:     lvl,
		mode:            ModeNormal,
		width:           100,
		height:          30,
	}

	return m
}

// Init begins streaming logcat and pid watching
func (m *Model) Init() tea.Cmd {
	// Query device properties
	if m.deviceSerial != "" {
		m.androidRelease = adb.GetDeviceProperty(m.deviceSerial, "ro.build.version.release")
		m.sdkVersion = adb.GetDeviceProperty(m.deviceSerial, "ro.build.version.sdk")
		m.deviceModel = adb.GetDeviceProperty(m.deviceSerial, "ro.product.model")
	}

	// Start logcat stream
	m.streamCancel = StartLogcatStream(m.ctx, m.deviceSerial, m.logEntryChan, m.errChan)

	// Start PID watcher
	if m.packageName != "" {
		go WatchPids(m.ctx, m.deviceSerial, m.packageName, 2*time.Second, m.pidChan)
	}

	return tea.Batch(
		waitForLogEntry(m.logEntryChan),
		waitForPidUpdate(m.pidChan),
		loadInstalledPackagesCmd(m.deviceSerial),
	)
}

type logEntryMsg *LogEntry
type pidsMsg []string
type installedPackagesMsg []string
type errMsg error
type clearToastMsg struct{}

func loadInstalledPackagesCmd(serial string) tea.Cmd {
	return func() tea.Msg {
		pkgs, _ := adb.GetInstalledPackages(serial)
		return installedPackagesMsg(pkgs)
	}
}

func waitForLogEntry(ch <-chan *LogEntry) tea.Cmd {
	return func() tea.Msg {
		entry, ok := <-ch
		if !ok {
			return nil
		}
		return logEntryMsg(entry)
	}
}

func waitForPidUpdate(ch <-chan []string) tea.Cmd {
	return func() tea.Msg {
		pids, ok := <-ch
		if !ok {
			return nil
		}
		return pidsMsg(pids)
	}
}

func clearToastCmd() tea.Cmd {
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg {
		return clearToastMsg{}
	})
}

// Update handles Bubble Tea messages
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case logEntryMsg:
		entry := (*LogEntry)(msg)
		if entry != nil {
			if !m.paused {
				// Buffer pruning to prevent unbounded memory growth
				if len(m.logEntries) >= maxLogBuffer {
					cut := 4000
					m.logEntries = m.logEntries[cut:]
					m.recomputeFiltered()
				}

				newIdx := len(m.logEntries)
				m.logEntries = append(m.logEntries, *entry)

				// Crash aggregation
				if entry.IsCrashStart {
					m.currentCrash = &CrashEvent{
						Timestamp: entry.Timestamp,
						Summary:   entry.Message,
						Lines:     []LogEntry{*entry},
					}
					m.crashes = append(m.crashes, *m.currentCrash)
				} else if entry.IsCrashContinue && m.currentCrash != nil {
					m.currentCrash.Lines = append(m.currentCrash.Lines, *entry)
					// Update last crash in list
					if len(m.crashes) > 0 {
						m.crashes[len(m.crashes)-1] = *m.currentCrash
					}
				} else if !entry.IsCrashContinue {
					m.currentCrash = nil
				}

				// Check filter
				filter := m.currentFilter()
				if filter.Allows(entry) {
					m.filteredIndices = append(m.filteredIndices, newIdx)
					if m.scroll > 0 {
						m.newLinesWhileScrolled++
					}
				}
			}
		}
		cmds = append(cmds, waitForLogEntry(m.logEntryChan))

	case pidsMsg:
		pids := []string(msg)
		newMap := make(map[string]bool)
		for _, p := range pids {
			newMap[p] = true
		}
		m.pids = newMap
		if !m.allPackages && m.packageName != "" {
			m.recomputeFiltered()
		}
		cmds = append(cmds, waitForPidUpdate(m.pidChan))

	case installedPackagesMsg:
		m.installedPackages = []string(msg)
		m.filterPackageSuggestions()

	case clearToastMsg:
		m.toastMsg = ""
		m.toastErr = false

	case errMsg:
		m.showToast(msg.Error(), true)

	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.scrollUp(3)
		case tea.MouseButtonWheelDown:
			m.scrollDown(3)
		}

	case tea.KeyMsg:
		switch m.mode {
		case ModeFilter:
			return m.handleFilterKey(msg)
		case ModeExclude:
			return m.handleExcludeKey(msg)
		case ModeLevelPicker:
			return m.handleLevelPickerKey(msg)
		case ModeCrashDetail:
			return m.handleCrashDetailKey(msg)
		case ModeHelp:
			return m.handleHelpKey(msg)
		case ModePackageInput:
			return m.handlePackageInputKey(msg)
		default:
			return m.handleNormalKey(msg)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.Cleanup()
		return m, tea.Quit

	case " ":
		m.paused = !m.paused
		if m.paused {
			m.showToast("Log streaming paused", false)
		} else {
			m.showToast("Log streaming resumed", false)
		}

	case "up", "k":
		m.scrollUp(1)

	case "down", "j":
		m.scrollDown(1)

	case "pgup":
		m.scrollUp(20)

	case "pgdown":
		m.scrollDown(20)

	case "end", "G":
		m.scroll = 0
		m.newLinesWhileScrolled = 0

	case "f":
		m.mode = ModeFilter
		m.filterInput.Focus()
		return m, textinput.Blink

	case "x":
		m.mode = ModeExclude
		m.excludeInput.Focus()
		return m, textinput.Blink

	case "L":
		m.mode = ModeLevelPicker
		// Set cursor to current active level
		for i, opt := range LevelFilterOptions {
			if opt.Code == m.levelFilter {
				m.levelPickerCursor = i
				break
			}
		}

	case "a":
		m.allPackages = !m.allPackages
		m.recomputeFiltered()
		if m.allPackages {
			m.showToast("Showing all packages", false)
		} else {
			m.showToast("Filtered to package: "+m.packageName, false)
		}

	case "p":
		m.mode = ModePackageInput
		m.packageInput.SetValue(m.packageName)
		m.packageInput.Focus()
		m.filterPackageSuggestions()
		return m, textinput.Blink

	case "c":
		m.logEntries = nil
		m.filteredIndices = nil
		m.scroll = 0
		m.newLinesWhileScrolled = 0
		_ = adb.ClearLogcat(m.deviceSerial)
		m.showToast("Cleared log buffer", false)

	case "w":
		visible := make([]LogEntry, len(m.filteredIndices))
		for i, idx := range m.filteredIndices {
			visible[i] = m.logEntries[idx]
		}
		filename, err := ExportLogsToFile(visible, "gadb-log")
		if err != nil {
			m.showToast("Export failed: "+err.Error(), true)
		} else {
			m.showToast("Exported to "+filename, false)
		}

	case "y":
		if len(m.crashes) > 0 {
			m.mode = ModeCrashDetail
			m.crashScroll = 0
		} else {
			m.showToast("No crashes detected", false)
		}

	case "?", "h":
		m.mode = ModeHelp
	}

	return m, nil
}

func (m *Model) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.filterInput.SetValue("")
		m.filterInput.Blur()
		m.mode = ModeNormal
		m.recomputeFiltered()
		return m, nil
	case tea.KeyEnter:
		m.filterInput.Blur()
		m.mode = ModeNormal
		m.recomputeFiltered()
		return m, nil
	default:
		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		m.recomputeFiltered()
		return m, cmd
	}
}

func (m *Model) handleExcludeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.excludeInput.SetValue("")
		m.excludeInput.Blur()
		m.mode = ModeNormal
		m.recomputeFiltered()
		return m, nil
	case tea.KeyEnter:
		m.excludeInput.Blur()
		m.mode = ModeNormal
		m.recomputeFiltered()
		return m, nil
	default:
		var cmd tea.Cmd
		m.excludeInput, cmd = m.excludeInput.Update(msg)
		m.recomputeFiltered()
		return m, cmd
	}
}

func (m *Model) handleLevelPickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.mode = ModeNormal
	case "up", "k":
		m.levelPickerCursor--
		if m.levelPickerCursor < 0 {
			m.levelPickerCursor = len(LevelFilterOptions) - 1
		}
	case "down", "j":
		m.levelPickerCursor++
		if m.levelPickerCursor >= len(LevelFilterOptions) {
			m.levelPickerCursor = 0
		}
	case "enter":
		m.levelFilter = LevelFilterOptions[m.levelPickerCursor].Code
		m.mode = ModeNormal
		m.recomputeFiltered()
		m.showToast("Level filter: "+m.levelFilter, false)
	}
	return m, nil
}

func (m *Model) handleCrashDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(m.crashes) == 0 {
		m.mode = ModeNormal
		return m, nil
	}
	crash := m.crashes[len(m.crashes)-1]

	switch msg.String() {
	case "esc", "q":
		m.mode = ModeNormal
	case "up", "k":
		if m.crashScroll > 0 {
			m.crashScroll--
		}
	case "down", "j":
		m.crashScroll++
	case "c":
		err := CopyToClipboard(crash.FullStackTrace())
		if err != nil {
			m.showToast("Failed to copy: "+err.Error(), true)
		} else {
			m.showToast("Copied stack trace to clipboard", false)
		}
	case "a":
		prompt := FormatAIAgentPrompt(crash, m.packageName)
		err := CopyToClipboard(prompt)
		if err != nil {
			m.showToast("Failed to copy: "+err.Error(), true)
		} else {
			m.showToast("Copied AI agent prompt to clipboard", false)
		}
	case "w":
		filename, err := ExportCrashToFile(crash)
		if err != nil {
			m.showToast("Export failed: "+err.Error(), true)
		} else {
			m.showToast("Exported crash to "+filename, false)
		}
	case "s":
		_ = SearchCrashOnline(crash.Summary)
		m.showToast("Opened browser search", false)
	}
	return m, nil
}

func (m *Model) handleHelpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "?", "h", "enter":
		m.mode = ModeNormal
	}
	return m, nil
}

func (m *Model) handlePackageInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.packageInput.Blur()
		m.mode = ModeNormal
		return m, nil

	case tea.KeyUp:
		if m.packageCursor > 0 {
			m.packageCursor--
		}
		return m, nil

	case tea.KeyDown:
		if m.packageCursor < len(m.filteredPackages)-1 {
			m.packageCursor++
		}
		return m, nil

	case tea.KeyTab:
		if m.packageCursor >= 0 && m.packageCursor < len(m.filteredPackages) {
			m.packageInput.SetValue(m.filteredPackages[m.packageCursor])
		}
		return m, nil

	case tea.KeyEnter:
		selected := strings.TrimSpace(m.packageInput.Value())
		// If user navigated suggestions with arrows and didn't change text, prefer highlighted
		if m.packageCursor >= 0 && m.packageCursor < len(m.filteredPackages) && selected != "" && strings.Contains(m.filteredPackages[m.packageCursor], selected) {
			selected = m.filteredPackages[m.packageCursor]
		}

		m.packageInput.Blur()
		m.mode = ModeNormal
		m.packageName = selected
		m.allPackages = (selected == "")

		// Refresh PIDs immediately
		m.pids = make(map[string]bool)
		if m.packageName != "" {
			if pids, err := adb.GetPidsForPackage(m.deviceSerial, m.packageName); err == nil {
				for _, p := range pids {
					m.pids[p] = true
				}
			}
			m.showToast("Target package set to: "+m.packageName, false)
		} else {
			m.showToast("Target package cleared (showing all)", false)
		}

		m.recomputeFiltered()
		return m, nil

	default:
		var cmd tea.Cmd
		m.packageInput, cmd = m.packageInput.Update(msg)
		m.filterPackageSuggestions()
		return m, cmd
	}
}

func (m *Model) filterPackageSuggestions() {
	query := strings.ToLower(strings.TrimSpace(m.packageInput.Value()))
	var filtered []string
	for _, p := range m.installedPackages {
		if query == "" || strings.Contains(strings.ToLower(p), query) {
			filtered = append(filtered, p)
			if len(filtered) >= 15 {
				break
			}
		}
	}
	m.filteredPackages = filtered
	if m.packageCursor >= len(m.filteredPackages) {
		m.packageCursor = 0
	}
}

func (m *Model) scrollUp(delta int) {
	m.scroll += delta
	maxScroll := len(m.filteredIndices)
	if m.scroll > maxScroll {
		m.scroll = maxScroll
	}
}

func (m *Model) scrollDown(delta int) {
	m.scroll -= delta
	if m.scroll <= 0 {
		m.scroll = 0
		m.newLinesWhileScrolled = 0
	}
}

func (m *Model) currentFilter() *LogFilter {
	return &LogFilter{
		IncludeQuery: m.filterInput.Value(),
		ExcludeQuery: m.excludeInput.Value(),
		Level:        m.levelFilter,
		PackageName:  m.packageName,
		AllPackages:  m.allPackages,
		PIDs:         m.pids,
	}
}

func (m *Model) recomputeFiltered() {
	filter := m.currentFilter()
	filtered := make([]int, 0, len(m.logEntries))
	for i := range m.logEntries {
		if filter.Allows(&m.logEntries[i]) {
			filtered = append(filtered, i)
		}
	}
	m.filteredIndices = filtered
	if m.scroll > len(m.filteredIndices) {
		m.scroll = len(m.filteredIndices)
	}
}

func (m *Model) showToast(msg string, isErr bool) {
	m.toastMsg = msg
	m.toastErr = isErr
}

// Cleanup stops background processes and cleans up terminal
func (m *Model) Cleanup() {
	if m.streamCancel != nil {
		m.streamCancel()
	}
	if m.cancelCtx != nil {
		m.cancelCtx()
	}
}

// View renders the TUI screen
func (m *Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return "Initializing..."
	}

	headerStr := RenderHeader(m, m.width)
	headerLines := strings.Split(headerStr, "\n")
	headerH := len(headerLines)

	footerStr := RenderFooter(m, m.width)
	footerLines := strings.Split(footerStr, "\n")
	footerH := len(footerLines)

	viewportH := m.height - headerH - footerH
	if viewportH < 1 {
		viewportH = 1
	}

	totalVisible := len(m.filteredIndices)
	var viewportContent string

	if totalVisible == 0 {
		emptyMsg := " Waiting for log lines..."
		if m.paused {
			emptyMsg = " Logcat PAUSED — Press Space to resume"
		}
		viewportContent = lipgloss.NewStyle().
			Height(viewportH).
			Width(m.width).
			Foreground(lipgloss.Color("#6C7086")).
			Render(emptyMsg)
	} else {
		// Calculate slice of filteredIndices to render
		end := totalVisible - m.scroll
		if end < 0 {
			end = 0
		}
		start := end - viewportH
		if start < 0 {
			start = 0
		}

		lines := RenderViewportLines(m, start, end-start, m.width)
		// If wrapped lines exceed viewportH, take the last viewportH lines
		if len(lines) > viewportH {
			lines = lines[len(lines)-viewportH:]
		}
		// Pad to fill viewportH
		for len(lines) < viewportH {
			lines = append(lines, "")
		}
		viewportContent = strings.Join(lines, "\n")
	}

	screen := headerStr + "\n" + viewportContent + "\n" + footerStr

	// Render overlay modals if active
	switch m.mode {
	case ModePackageInput:
		modal := RenderPackageModal(m, m.width, m.height)
		return modal
	case ModeLevelPicker:
		modal := RenderLevelPickerModal(m, m.width, m.height)
		return modal
	case ModeCrashDetail:
		modal := RenderCrashPopup(m, m.width, m.height)
		return modal
	case ModeHelp:
		modal := RenderHelpModal(m.width, m.height)
		return modal
	}

	return screen
}
