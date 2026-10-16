package utils

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/chzyer/readline"
	pui "github.com/manifoldco/promptui"
)

// Show selection promp and return the selected index and item
func ShowSelection(message string, selections []string) (int, string) {
	prompt := pui.Select{
		Label: message,
		Items: selections,
	}

	selectedIndex, selectedItem, err := prompt.Run()
	if err != nil {
		panic(err)
	}
	return selectedIndex, selectedItem
}

// interpretYesNo maps a typed answer to a yes/no decision. Empty input means
// "no" (the default advertised in the prompt). The second return value reports
// whether the input was a recognized answer.
func interpretYesNo(input string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "", "n", "no":
		return false, true
	case "y", "yes":
		return true, true
	default:
		return false, false
	}
}

// Show a yes or no confirmation prompt, return true if yes, false if no.
// Accepts y/yes/n/no (case-insensitive); pressing Enter answers the [y/N]
// default, which is no. Works with piped input too, so it is scriptable.
func ShowYesOrNoConfirmation(message string) bool {
	if readline.IsTerminal(int(os.Stdin.Fd())) {
		return confirmOnTerminal(message)
	}
	return confirmFromReader(message)
}

// confirmOnTerminal prompts through a readline session. readline puts the
// terminal into raw mode itself, so the prompt keeps working even when the
// terminal was left in a mode where Enter sends a bare carriage return that
// the kernel never translates to a newline, and it treats both '\r' and '\n'
// as Enter. It also gives backspace and arrow-key editing for free, and
// restores the previous terminal state when done.
func confirmOnTerminal(message string) bool {
	rl, err := readline.NewEx(&readline.Config{
		Prompt:       message + " [y/N] ",
		HistoryLimit: -1,
	})
	if err != nil {
		return confirmFromReader(message)
	}
	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err != nil {
			// Interrupt (Ctrl+C) or EOF (Ctrl+D); readline already moved
			// the cursor to the next line.
			return false
		}

		answer, valid := interpretYesNo(line)
		if valid {
			return answer
		}
		fmt.Println(">> please answer y or n")
	}
}

// confirmFromReader reads the answer straight from stdin, so the prompt stays
// scriptable through pipes.
func confirmFromReader(message string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s [y/N] ", message)

		line, err := readInputLine(reader)
		if err != nil && line == "" {
			fmt.Println()
			return false
		}

		answer, valid := interpretYesNo(line)
		if valid {
			return answer
		}
		fmt.Println(">> please answer y or n")
	}
}

// readInputLine reads a single line, accepting '\n', '\r\n' or a bare '\r' as
// the terminator. It returns whatever was read when it hits EOF.
func readInputLine(reader *bufio.Reader) (string, error) {
	var line []byte

	for {
		b, err := reader.ReadByte()
		if err != nil {
			if len(line) > 0 {
				return string(line), nil
			}
			return "", err
		}

		switch b {
		case '\n':
			return string(line), nil
		case '\r':
			// Swallow the '\n' of a '\r\n' pair, so it does not answer the
			// next prompt by itself.
			if next, err := reader.Peek(1); err == nil && next[0] == '\n' {
				_, _ = reader.ReadByte()
			}
			return string(line), nil
		case '\b', 0x7f: // backspace, in case the input is not line-edited
			if len(line) > 0 {
				line = line[:len(line)-1]
			}
		default:
			line = append(line, b)
		}
	}
}

// Show a prompt that ask for input, return the input string
func ShowPrompt(message string) string {
	prompt := pui.Prompt{
		Label: message,
	}

	result, _ := prompt.Run()
	return result
}
