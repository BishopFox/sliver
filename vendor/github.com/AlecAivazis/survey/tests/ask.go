package main

import (
	"fmt"

	"gopkg.in/AlecAivazis/survey.v1"
)

// the questions to ask
var simpleQs = []*survey.Question{
	{
		Name: "name",
		Prompt: &survey.Input{
			Message: "What is your name?",
			Default: "Johnny Appleseed",
		},
	},
	{
		Name: "color",
		Prompt: &survey.Select{
			Message: "Choose a color:",
			Options: []string{"red", "blue", "green", "yellow"},
			Default: "yellow",
		},
		Validate: survey.Required,
	},
}

func main() {

	fmt.Println("Asking many.")
	// a place to store the answers
	ans := struct {
		Name  string
		Color string
	}{}
	err := survey.Ask(simpleQs, &ans)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Println("Asking one.")
	answer := ""
	err = survey.AskOne(simpleQs[0].Prompt, &answer, nil)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Printf("Answered with %v.\n", answer)

	fmt.Println("Asking one with validation.")
	vAns := ""
	err = survey.AskOne(&survey.Input{Message: "What is your name?"}, &vAns, survey.Required)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Printf("Answered with %v.\n", vAns)
}
