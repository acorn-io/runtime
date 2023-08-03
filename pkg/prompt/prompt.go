package prompt

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/pterm/pterm"
)

var (
	NoPromptRemove bool
)

func Password(msg string) ([]byte, error) {
	var res string
	err := survey.AskOne(&survey.Password{
		Message: msg,
	}, &res)
	return []byte(res), err
}

func Bool(msg string, def bool) (result bool, _ error) {
	err := survey.AskOne(&survey.Confirm{
		Message: msg,
		Default: def,
	}, &result)
	return result, err
}

func Choice(msg string, choices []string, def string) (result string, _ error) {
	err := survey.AskOne(&survey.Select{
		Message: msg,
		Options: choices,
		Default: def,
	}, &result)
	return result, err
}

func Remove(obj string) error {
	if NoPromptRemove {
		return nil
	}
	msg := "Do you want to remove the above " + obj
	if ok, err := Bool(msg, false); err != nil {
		return err
	} else if !ok {
		pterm.Warning.Println("Aborting remove")
		return fmt.Errorf("aborting remove")
	}
	return nil
}
