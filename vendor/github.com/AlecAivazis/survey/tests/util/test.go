package TestUtil

import (
	"fmt"
	"reflect"

	"gopkg.in/AlecAivazis/survey.v1"
)

type TestTableEntry struct {
	Name     string
	Prompt   survey.Prompt
	Value    interface{}
	Validate func(interface{}) error
}

func formatAnswer(ans interface{}) {
	// show the answer to the user
	fmt.Printf("Answered %v.\n", reflect.ValueOf(ans).Elem())
	fmt.Println("---------------------")
}

func RunTable(table []TestTableEntry) {
	// go over every entry in the table
	for _, entry := range table {
		// tell the user what we are going to ask them
		fmt.Println(entry.Name)
		// perform the ask
		err := survey.AskOne(entry.Prompt, entry.Value, entry.Validate)
		if err != nil {
			fmt.Printf("AskOne on %v's prompt failed: %v.", entry.Name, err.Error())
			break
		}
		// show the answer to the user
		formatAnswer(entry.Value)
	}
}

func RunErrorTable(table []TestTableEntry) {
	// go over every entry in the table
	for _, entry := range table {
		// tell the user what we are going to ask them
		fmt.Println(entry.Name)
		// perform the ask
		err := survey.AskOne(entry.Prompt, entry.Value, entry.Validate)
		if err == nil {
			fmt.Printf("AskOne on %v's prompt didn't fail.", entry.Name)
			break
		}
	}
}
