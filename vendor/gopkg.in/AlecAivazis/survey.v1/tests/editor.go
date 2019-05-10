package main

import (
	"fmt"
	"strings"

	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/tests/util"
)

var answer = ""

var goodTable = []TestUtil.TestTableEntry{
	{
		"should open in editor", &survey.Editor{
			Message: "should open",
		}, &answer, nil,
	},
	{
		"has help", &survey.Editor{
			Message: "press ? to see message",
			Help:    "Does this work?",
		}, &answer, nil,
	},
	{
		"should not include the default value in the prompt", &survey.Editor{
			Message:     "the default value 'Hello World' should not include in the prompt",
			HideDefault: true,
			Default:     "Hello World",
		}, &answer, nil,
	},
	{
		"should write the default value to the temporary file before the launch of the editor", &survey.Editor{
			Message:       "the default value 'Hello World' is written to the temporary file before the launch of the editor",
			AppendDefault: true,
			Default:       "Hello World",
		}, &answer, nil,
	},
	{
		Name: "should print the validation error, and recall the submitted invalid value instead of the default",
		Prompt: &survey.Editor{
			Message:       "the default value 'Hello World' is written to the temporary file before the launch of the editor",
			AppendDefault: true,
			Default:       `this is the default value. change it to something containing "invalid" (in vi type "ccinvalid<Esc>ZZ")`,
		},
		Value: &answer,
		Validate: func(v interface{}) error {
			s := v.(string)
			if strings.Contains(s, "invalid") {
				return fmt.Errorf(`this is the error message. change the input to something not containing "invalid"`)
			}
			return nil
		},
	},
}

func main() {
	TestUtil.RunTable(goodTable)
}
