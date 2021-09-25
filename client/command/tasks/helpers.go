package tasks

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/protobuf/clientpb"
)

// SelectBeaconTask - Select a beacon task interactively
func SelectBeaconTask(tasks []*clientpb.BeaconTask) (*clientpb.BeaconTask, error) {

	// Render selection table
	buf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(buf, 0, 2, 2, ' ', 0)
	for _, task := range tasks {
		shortID := strings.Split(task.ID, "-")[0]
		fmt.Fprintf(table, "%s\t%s\t%s\t\n", shortID, task.Description, prettyState(task.State))
	}
	table.Flush()
	options := strings.Split(buf.String(), "\n")
	options = options[:len(options)-1]
	if len(options) == 0 {
		return nil, errors.New("no task to select from")
	}

	selected := ""
	prompt := &survey.Select{
		Message: "Select a beacon task:",
		Options: options,
	}
	err := survey.AskOne(prompt, &selected)
	if err != nil {
		return nil, err
	}
	for index, value := range options {
		if value == selected {
			return tasks[index], nil
		}
	}
	return nil, errors.New("task not found")
}
