package tasks

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/rsteube/carapace"
)

// SelectBeaconTask - Select a beacon task interactively.
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

// BeaconTaskIDCompleter returns a structured list of tasks completions, grouped by state.
func BeaconTaskIDCompleter(con *console.SliverClient) carapace.Action {
	callback := func(ctx carapace.Context) carapace.Action {
		beacon := con.ActiveTarget.GetBeacon()
		if beacon == nil {
			return carapace.ActionMessage("no active beacon")
		}

		beaconTasks, err := con.Rpc.GetBeaconTasks(context.Background(), &clientpb.Beacon{ID: beacon.ID})
		if err != nil {
			return carapace.ActionMessage("Failed to fetch tasks: %s", err.Error())
		}

		completed := make([]string, 0)
		pending := make([]string, 0)
		sent := make([]string, 0)
		canceled := make([]string, 0)

		for _, task := range beaconTasks.Tasks {
			var desc string

			switch task.State {
			case "pending":
				pending = append(pending, task.ID)
				pending = append(pending, task.Description)

			case "completed":
				completedAt := time.Unix(task.CompletedAt, 0).Format(time.RFC1123)
				if time.Unix(task.CompletedAt, 0).IsZero() {
					completedAt = ""
				}
				desc += fmt.Sprintf("(completed: %s)", completedAt)
				desc += task.Description

				completed = append(completed, task.ID)
				completed = append(completed, task.Description)

			case "sent":
				sentAt := time.Unix(task.SentAt, 0).Format(time.RFC1123)
				if time.Unix(task.SentAt, 0).IsZero() {
					sentAt = ""
				}
				desc += fmt.Sprintf("(sent: %s)", sentAt)
				desc += task.Description

				sent = append(sent, task.ID)
				sent = append(sent, task.Description)

			case "canceled":
				canceled = append(canceled, task.ID)
				canceled = append(canceled, task.Description)
			}
		}

		return carapace.Batch(
			carapace.ActionValuesDescribed(pending...).Tag("pending tasks"),
			carapace.ActionValuesDescribed(completed...).Tag("completed tasks"),
			carapace.ActionValuesDescribed(sent...).Tag("sent tasks"),
			carapace.ActionValuesDescribed(canceled...).Tag("canceled tasks"),
		).ToA()
	}

	return carapace.ActionCallback(callback)
}

// BeaconPendingTasksCompleter completes pending tasks.
func BeaconPendingTasksCompleter(con *console.SliverClient) carapace.Action {
	callback := func(ctx carapace.Context) carapace.Action {
		beacon := con.ActiveTarget.GetBeacon()
		if beacon == nil {
			return carapace.ActionMessage("no active beacon")
		}

		beaconTasks, err := con.Rpc.GetBeaconTasks(context.Background(), &clientpb.Beacon{ID: beacon.ID})
		if err != nil {
			return carapace.ActionMessage("Failed to fetch tasks: %s", err.Error())
		}

		pending := make([]string, 0)

		for _, task := range beaconTasks.Tasks {
			if task.State == "pending" {
				pending = append(pending, task.ID)
				pending = append(pending, task.Description)
			}
		}

		return carapace.ActionValuesDescribed(pending...).Tag("pending tasks")
	}

	return carapace.ActionCallback(callback)
}
