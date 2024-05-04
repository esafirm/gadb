package utils

import (
	"errors"
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

// Show a yes or no confirmation prompt, return true if yes, false if no
func ShowYesOrNoConfirmation(message string) bool {
	validation := func(input string) error {
		if input == "y" || input == "Y" || input == "n" || input == "N" {
			return nil
		}
		return errors.New("answer not valid")
	}

	prompt := pui.Prompt{
		Label:    message + " [Y,n]",
		Validate: validation,
	}

	result, _ := prompt.Run()

	if result == "Y" || result == "y" {
		return true
	}
	return false
}

// Show a prompt that ask for input, return the input string
func ShowPrompt(message string) string {
	prompt := pui.Prompt{
		Label: message,
	}

	result, _ := prompt.Run()
	return result
}
