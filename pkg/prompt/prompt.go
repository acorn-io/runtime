package prompt

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/pterm/pterm"
)

var (
	NoPromptRemove bool
)

func Bool(msg string, def bool) (result bool, _ error) {
	err := survey.AskOne(&survey.Confirm{
		Message: msg,
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
