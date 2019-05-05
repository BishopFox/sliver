package main

import (
	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/tests/util"
)

var answer = ""

var goodTable = []TestUtil.TestTableEntry{
	{
		"standard", &survey.Select{
			Message: "Choose a color:",
			Options: []string{"red", "blue", "green"},
		}, &answer, nil,
	},
	{
		"short", &survey.Select{
			Message: "Choose a color:",
			Options: []string{"red", "blue"},
		}, &answer, nil,
	},
	{
		"default", &survey.Select{
			Message: "Choose a color (should default blue):",
			Options: []string{"red", "blue", "green"},
			Default: "blue",
		}, &answer, nil,
	},
	{
		"one", &survey.Select{
			Message: "Choose one:",
			Options: []string{"hello"},
		}, &answer, nil,
	},
	{
		"no help, type ?", &survey.Select{
			Message: "Choose a color:",
			Options: []string{"red", "blue"},
		}, &answer, nil,
	},
	{
		"passes through bottom", &survey.Select{
			Message: "Choose one:",
			Options: []string{"red", "blue"},
		}, &answer, nil,
	},
	{
		"passes through top", &survey.Select{
			Message: "Choose one:",
			Options: []string{"red", "blue"},
		}, &answer, nil,
	},
	{
		"can navigate with j/k", &survey.Select{
			Message: "Choose one:",
			Options: []string{"red", "blue", "green"},
		}, &answer, nil,
	},
}

var badTable = []TestUtil.TestTableEntry{
	{
		"no options", &survey.Select{
			Message: "Choose one:",
		}, &answer, nil,
	},
}

func main() {
	TestUtil.RunTable(goodTable)
	TestUtil.RunErrorTable(badTable)
}
