package prompt

import "github.com/AlecAivazis/survey/v2"

func Bool(msg string, def bool) (result bool, _ error) {
	err := survey.AskOne(&survey.Confirm{
		Message: msg,
		Default: def,
	}, &result)
	return result, err
}
