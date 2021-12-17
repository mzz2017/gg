package common

import "github.com/AlecAivazis/survey/v2"

func SetRequire(options *survey.AskOptions) error {
	options.Validators = append(options.Validators, survey.Required)
	return nil
}
