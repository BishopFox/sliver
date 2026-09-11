package forms

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
	"unicode"

	"charm.land/huh/v2"
	"github.com/bishopfox/sliver/client/theme"
	"github.com/bishopfox/sliver/protobuf/clientpb"
)

type crackJobChoice struct {
	id    string
	label string
}

// CrackJobSelectForm prompts for a durable crack job and returns its full ID.
func CrackJobSelectForm(jobs []*clientpb.CrackJob) (string, error) {
	choices := crackJobChoices(jobs)
	if len(choices) == 0 {
		return "", errors.New("crack job options are required")
	}

	options := make([]huh.Option[string], 0, len(choices))
	for _, choice := range choices {
		options = append(options, huh.NewOption(choice.label, choice.id))
	}

	selectedID := choices[0].id
	field := huh.NewSelect[string]().
		Title("Select a crack job:").
		Options(options...).
		Height(listHeight(len(options)) + 1).
		Value(&selectedID)
	form := huh.NewForm(huh.NewGroup(field)).
		WithTheme(theme.HuhTheme()).
		WithWidth(getTerminalWidth())
	if err := runForm(form); err != nil {
		return "", err
	}
	if strings.TrimSpace(selectedID) == "" {
		return "", errors.New("no crack job selected")
	}
	return selectedID, nil
}

func crackJobChoices(jobs []*clientpb.CrackJob) []crackJobChoice {
	ordered := make([]*clientpb.CrackJob, 0, len(jobs))
	for _, job := range jobs {
		if job != nil && strings.TrimSpace(job.GetID()) != "" {
			ordered = append(ordered, job)
		}
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		return crackJobChoiceTime(ordered[i]).After(crackJobChoiceTime(ordered[j]))
	})
	if len(ordered) == 0 {
		return nil
	}

	var output bytes.Buffer
	table := tabwriter.NewWriter(&output, 0, 2, 2, ' ', 0)
	for _, job := range ordered {
		_, _ = fmt.Fprintf(table, "JOB\t%s\t%s\t%s\tkeyspace %s\ttasks %d\tresults %d\n",
			shortCrackJobID(job.GetID()),
			safeCrackJobText(job.GetStatus().String()),
			formatCrackJobChoiceTime(job.GetCreatedAt()),
			valueOrDashForCrackJob(job.GetKeyspace()),
			len(job.GetTasks()),
			job.GetResultCount(),
		)
	}
	_ = table.Flush()

	labels := strings.Split(strings.TrimSuffix(output.String(), "\n"), "\n")
	choices := make([]crackJobChoice, 0, len(ordered))
	for index, job := range ordered {
		choices = append(choices, crackJobChoice{id: strings.TrimSpace(job.GetID()), label: labels[index]})
	}
	return choices
}

func crackJobChoiceTime(job *clientpb.CrackJob) time.Time {
	if job == nil {
		return time.Time{}
	}
	if job.GetUpdatedAt() > 0 {
		return time.Unix(job.GetUpdatedAt(), 0)
	}
	if parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(job.GetCreatedAt())); err == nil {
		return parsed
	}
	return time.Time{}
}

func shortCrackJobID(id string) string {
	id = safeCrackJobText(id)
	if short, _, ok := strings.Cut(id, "-"); ok {
		return short
	}
	return id
}

func formatCrackJobChoiceTime(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "0" {
		return "-"
	}
	if unix, err := strconv.ParseInt(value, 10, 64); err == nil {
		return time.Unix(unix, 0).UTC().Format(time.RFC3339)
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC().Format(time.RFC3339)
	}
	return safeCrackJobText(value)
}

func valueOrDashForCrackJob(value string) string {
	value = safeCrackJobText(value)
	if value == "" {
		return "-"
	}
	return value
}

func safeCrackJobText(value string) string {
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, value)
	value = strings.TrimSpace(value)
	const maximumRunes = 96
	runes := []rune(value)
	if len(runes) > maximumRunes {
		value = string(runes[:maximumRunes-1]) + "…"
	}
	return value
}
