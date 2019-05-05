package main

import (
	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/tests/util"
)

var (
	confirmAns     = false
	inputAns       = ""
	multiselectAns = []string{}
	selectAns      = ""
	passwordAns    = ""
)

var goodTable = []TestUtil.TestTableEntry{
	{
		"confirm", &survey.Confirm{
			Message: "Is it raining?",
			Help:    "Go outside, if your head becomes wet the answer is probably 'yes'",
		}, &confirmAns, nil,
	},
	{
		"input", &survey.Input{
			Message: "What is your phone number:",
			Help:    "Phone number should include the area code, parentheses optional",
		}, &inputAns, nil,
	},
	{
		"select", &survey.MultiSelect{
			Message: "What days are you available:",
			Help:    "We are closed weekends and avaibility is limited on Wednesday",
			Options: []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday"},
			Default: []string{"Monday", "Tuesday", "Thursday", "Friday"},
		}, &multiselectAns, nil,
	},
	{
		"select", &survey.Select{
			Message: "Choose a color:",
			Help:    "Blue is the best color, but it is your choice",
			Options: []string{"red", "blue", "green"},
			Default: "blue",
		}, &selectAns, nil,
	},
	{
		"password", &survey.Password{
			Message: "Enter a secret:",
			Help:    "Don't really enter a secret, this is just for testing",
		}, &passwordAns, nil,
	},
}

func main() {
	TestUtil.RunTable(goodTable)
}
