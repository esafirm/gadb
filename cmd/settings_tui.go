package cmd

import (
	"bytes"
	"fmt"
	"io"
	"text/template"

	"github.com/chzyer/readline"
	"github.com/juju/ansiterm"
	"github.com/manifoldco/promptui/list"
	"github.com/manifoldco/promptui/screenbuf"
	pui "github.com/manifoldco/promptui"
)

// settingsTabKey is the Tab key. Pressing it on the highlighted row toggles
// the setting immediately without opening the action selector.
const settingsTabKey = readline.CharTab

const (
	settingsHideCursor = "\033[?25l"
	settingsShowCursor = "\033[?25h"
)

type settingsCompiledTemplates struct {
	label    *template.Template
	active   *template.Template
	inactive *template.Template
	selected *template.Template
	details  *template.Template
	help     *template.Template
}

func compileSettingsTemplates(src *pui.SelectTemplates) (*settingsCompiledTemplates, error) {
	if src == nil {
		src = &pui.SelectTemplates{}
	}
	funcMap := src.FuncMap
	if funcMap == nil {
		funcMap = pui.FuncMap
	}

	labelText := src.Label
	if labelText == "" {
		labelText = fmt.Sprintf("%s {{.}}: ", pui.IconInitial)
	}
	labelTpl, err := template.New("").Funcs(funcMap).Parse(labelText)
	if err != nil {
		return nil, err
	}

	activeText := src.Active
	if activeText == "" {
		activeText = fmt.Sprintf("%s {{ . | underline }}", pui.IconSelect)
	}
	activeTpl, err := template.New("").Funcs(funcMap).Parse(activeText)
	if err != nil {
		return nil, err
	}

	inactiveText := src.Inactive
	if inactiveText == "" {
		inactiveText = "  {{.}}"
	}
	inactiveTpl, err := template.New("").Funcs(funcMap).Parse(inactiveText)
	if err != nil {
		return nil, err
	}

	selectedText := src.Selected
	if selectedText == "" {
		selectedText = fmt.Sprintf(`{{ "%s" | green }} {{ . | faint }}`, pui.IconGood)
	}
	selectedTpl, err := template.New("").Funcs(funcMap).Parse(selectedText)
	if err != nil {
		return nil, err
	}

	var detailsTpl *template.Template
	if src.Details != "" {
		detailsTpl, err = template.New("").Funcs(funcMap).Parse(src.Details)
		if err != nil {
			return nil, err
		}
	}

	helpText := src.Help
	if helpText == "" {
		helpText = fmt.Sprintf(`{{ "Use the arrow keys to navigate:" | faint }} {{ .NextKey | faint }} ` +
			`{{ .PrevKey | faint }} {{ .PageDownKey | faint }} {{ .PageUpKey | faint }} ` +
			`{{ if .Search }} {{ "and" | faint }} {{ .SearchKey | faint }} {{ "toggles search" | faint }}{{ end }}`)
	}
	helpTpl, err := template.New("").Funcs(funcMap).Parse(helpText)
	if err != nil {
		return nil, err
	}

	return &settingsCompiledTemplates{
		label:    labelTpl,
		active:   activeTpl,
		inactive: inactiveTpl,
		selected: selectedTpl,
		details:  detailsTpl,
		help:     helpTpl,
	}, nil
}

func renderSettingsTpl(tpl *template.Template, data interface{}) []byte {
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return []byte(fmt.Sprintf("%v", data))
	}
	return buf.Bytes()
}

func renderSettingsDetails(tpl *template.Template, item interface{}) [][]byte {
	if tpl == nil {
		return nil
	}
	var buf bytes.Buffer
	w := ansiterm.NewTabWriter(&buf, 0, 0, 8, ' ', 0)
	if err := tpl.Execute(w, item); err != nil {
		fmt.Fprintf(w, "%v", item)
	}
	w.Flush()
	return bytes.Split(buf.Bytes(), []byte("\n"))
}

// runSettingsPicker shows the filterable settings list. Enter picks the
// highlighted row (original index into items). Tab toggles it directly:
// Tab is translated to Enter via FuncFilterInputRune so readline returns,
// and tabPressed distinguishes the two. On Tab no "selected" line is left
// behind so the caller can re-open the picker for the next toggle.
func runSettingsPicker(label interface{}, items []settingsTUIItem, size int, cursorPos int, templates *pui.SelectTemplates, searcher list.Searcher) (originalIndex int, tabPressed bool, err error) {
	if size == 0 {
		size = 5
	}
	l, err := list.New(items, size)
	if err != nil {
		return 0, false, err
	}
	l.Searcher = searcher
	l.SetCursor(cursorPos)

	compiled, err := compileSettingsTemplates(templates)
	if err != nil {
		return 0, false, err
	}

	keys := &pui.SelectKeys{
		Prev:     pui.Key{Code: pui.KeyPrev, Display: pui.KeyPrevDisplay},
		Next:     pui.Key{Code: pui.KeyNext, Display: pui.KeyNextDisplay},
		PageUp:   pui.Key{Code: pui.KeyBackward, Display: pui.KeyBackwardDisplay},
		PageDown: pui.Key{Code: pui.KeyForward, Display: pui.KeyForwardDisplay},
		Search:   pui.Key{Code: '/', Display: "/"},
	}

	c := &readline.Config{}
	c.FuncFilterInputRune = func(r rune) (rune, bool) {
		if r == settingsTabKey {
			tabPressed = true
			return pui.KeyEnter, true
		}
		return r, true
	}
	if err := c.Init(); err != nil {
		return 0, false, err
	}
	c.Stdin = readline.NewCancelableStdin(c.Stdin)
	c.HistoryLimit = -1
	c.UniqueEditLine = true

	rl, err := readline.NewEx(c)
	if err != nil {
		return 0, false, err
	}

	rl.Write([]byte(settingsHideCursor))
	sb := screenbuf.New(rl)

	cur := pui.NewCursor("", nil, false)

	canSearch := searcher != nil
	searchMode := true

	helpKeys := struct {
		NextKey     string
		PrevKey     string
		PageDownKey string
		PageUpKey   string
		Search      bool
		SearchKey   string
	}{
		NextKey:     keys.Next.Display,
		PrevKey:     keys.Prev.Display,
		PageDownKey: keys.PageDown.Display,
		PageUpKey:   keys.PageUp.Display,
		SearchKey:   keys.Search.Display,
		Search:      canSearch,
	}

	c.SetListener(func(line []rune, pos int, key rune) ([]rune, int, bool) {
		switch {
		case key == pui.KeyEnter:
			// Returnchevron handled by readline; just fall through to redraw.
		case key == keys.Next.Code || (key == 'j' && !searchMode):
			l.Next()
		case key == keys.Prev.Code || (key == 'k' && !searchMode):
			l.Prev()
		case key == keys.Search.Code:
			if !canSearch {
				break
			}
			if searchMode {
				searchMode = false
				cur.Replace("")
				l.CancelSearch()
			} else {
				searchMode = true
			}
		case key == pui.KeyBackspace || key == pui.KeyCtrlH:
			if !canSearch || !searchMode {
				break
			}
			cur.Backspace()
			if len(cur.Get()) > 0 {
				l.Search(cur.Get())
			} else {
				l.CancelSearch()
			}
		case key == keys.PageUp.Code || (key == 'h' && !searchMode):
			l.PageUp()
		case key == keys.PageDown.Code || (key == 'l' && !searchMode):
			l.PageDown()
		default:
			if canSearch && searchMode {
				cur.Update(string(line))
				l.Search(cur.Get())
			}
		}

		if searchMode {
			sb.WriteString(pui.SearchPrompt + cur.Format())
		} else {
			sb.Write(renderSettingsTpl(compiled.help, helpKeys))
		}

		sb.Write(renderSettingsTpl(compiled.label, label))

		visible, idx := l.Items()
		last := len(visible) - 1
		for i, item := range visible {
			page := " "
			switch i {
			case 0:
				if l.CanPageUp() {
					page = "↑"
				} else {
					page = " "
				}
			case last:
				if l.CanPageDown() {
					page = "↓"
				}
			}
			output := []byte(page + " ")
			if i == idx {
				output = append(output, renderSettingsTpl(compiled.active, item)...)
			} else {
				output = append(output, renderSettingsTpl(compiled.inactive, item)...)
			}
			sb.Write(output)
		}

		if idx == list.NotFound {
			sb.WriteString("")
			sb.WriteString("No results")
		} else {
			for _, d := range renderSettingsDetails(compiled.details, visible[idx]) {
				sb.Write(d)
			}
		}

		sb.Flush()
		return nil, 0, true
	})

	wasTab := false
	for {
		_, err = rl.Readline()
		if err != nil {
			switch {
			case err == readline.ErrInterrupt, err.Error() == "Interrupt":
				err = pui.ErrInterrupt
			case err == io.EOF:
				err = pui.ErrEOF
			}
			break
		}
		_, idx := l.Items()
		if idx != list.NotFound {
			wasTab = tabPressed
			break
		}
		// No results: stay in the picker, a Tab here means nothing to toggle.
		tabPressed = false
	}

	if err != nil {
		sb.Reset()
		sb.WriteString("")
		sb.Flush()
		rl.Write([]byte(settingsShowCursor))
		rl.Close()
		return 0, false, err
	}

	visible, idx := l.Items()
	originalIndex = l.Index()

	if wasTab {
		// Leave no "selected" line behind; the caller re-opens the picker
		// with the toggled state.
		_ = visible
		_ = idx
		sb.Reset()
		if err := sb.Clear(); err != nil {
			sb.Reset()
			sb.WriteString("")
			sb.Flush()
		} else {
			sb.Flush()
		}
		rl.Write([]byte(settingsShowCursor))
		rl.Close()
		return originalIndex, true, nil
	}

	item := visible[idx]
	sb.Reset()
	sb.Write(renderSettingsTpl(compiled.selected, item))
	sb.Flush()
	rl.Write([]byte(settingsShowCursor))
	rl.Close()

	return originalIndex, false, nil
}
