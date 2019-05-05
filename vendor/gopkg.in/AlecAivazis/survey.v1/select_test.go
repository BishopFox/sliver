package survey

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/Netflix/go-expect"
	"github.com/stretchr/testify/assert"
	"gopkg.in/AlecAivazis/survey.v1/core"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

func init() {
	// disable color output for all prompts to simplify testing
	core.DisableColor = true
}

func TestSelectRender(t *testing.T) {

	prompt := Select{
		Message: "Pick your word:",
		Options: []string{"foo", "bar", "baz", "buz"},
		Default: "baz",
	}

	helpfulPrompt := prompt
	helpfulPrompt.Help = "This is helpful"

	tests := []struct {
		title    string
		prompt   Select
		data     SelectTemplateData
		expected string
	}{
		{
			"Test Select question output",
			prompt,
			SelectTemplateData{SelectedIndex: 2, PageEntries: prompt.Options},
			strings.Join(
				[]string{
					fmt.Sprintf("%s Pick your word:  [Use arrows to move, type to filter]", core.QuestionIcon),
					"  foo",
					"  bar",
					fmt.Sprintf("%s baz", core.SelectFocusIcon),
					"  buz\n",
				},
				"\n",
			),
		},
		{
			"Test Select answer output",
			prompt,
			SelectTemplateData{Answer: "buz", ShowAnswer: true, PageEntries: prompt.Options},
			fmt.Sprintf("%s Pick your word: buz\n", core.QuestionIcon),
		},
		{
			"Test Select question output with help hidden",
			helpfulPrompt,
			SelectTemplateData{SelectedIndex: 2, PageEntries: prompt.Options},
			strings.Join(
				[]string{
					fmt.Sprintf("%s Pick your word:  [Use arrows to move, type to filter, %s for more help]", core.QuestionIcon, string(core.HelpInputRune)),
					"  foo",
					"  bar",
					fmt.Sprintf("%s baz", core.SelectFocusIcon),
					"  buz\n",
				},
				"\n",
			),
		},
		{
			"Test Select question output with help shown",
			helpfulPrompt,
			SelectTemplateData{SelectedIndex: 2, ShowHelp: true, PageEntries: prompt.Options},
			strings.Join(
				[]string{
					fmt.Sprintf("%s This is helpful", core.HelpIcon),
					fmt.Sprintf("%s Pick your word:  [Use arrows to move, type to filter]", core.QuestionIcon),
					"  foo",
					"  bar",
					fmt.Sprintf("%s baz", core.SelectFocusIcon),
					"  buz\n",
				},
				"\n",
			),
		},
	}

	for _, test := range tests {
		r, w, err := os.Pipe()
		assert.Nil(t, err, test.title)

		test.prompt.WithStdio(terminal.Stdio{Out: w})
		test.data.Select = test.prompt
		err = test.prompt.Render(
			SelectQuestionTemplate,
			test.data,
		)
		assert.Nil(t, err, test.title)

		w.Close()
		var buf bytes.Buffer
		io.Copy(&buf, r)

		assert.Contains(t, buf.String(), test.expected, test.title)
	}
}

func TestSelectPrompt(t *testing.T) {
	tests := []PromptTest{
		{
			"Test Select prompt interaction",
			&Select{
				Message: "Choose a color:",
				Options: []string{"red", "blue", "green"},
			},
			func(c *expect.Console) {
				c.ExpectString("Choose a color:")
				// Select blue.
				c.SendLine(string(terminal.KeyArrowDown))
				c.ExpectEOF()
			},
			"blue",
		},
		{
			"Test Select prompt interaction with default",
			&Select{
				Message: "Choose a color:",
				Options: []string{"red", "blue", "green"},
				Default: "green",
			},
			func(c *expect.Console) {
				c.ExpectString("Choose a color:")
				// Select green.
				c.SendLine("")
				c.ExpectEOF()
			},
			"green",
		},
		{
			"Test Select prompt interaction overriding default",
			&Select{
				Message: "Choose a color:",
				Options: []string{"red", "blue", "green"},
				Default: "blue",
			},
			func(c *expect.Console) {
				c.ExpectString("Choose a color:")
				// Select red.
				c.SendLine(string(terminal.KeyArrowUp))
				c.ExpectEOF()
			},
			"red",
		},
		{
			"Test Select prompt interaction and prompt for help",
			&Select{
				Message: "Choose a color:",
				Options: []string{"red", "blue", "green"},
				Help:    "My favourite color is red",
			},
			func(c *expect.Console) {
				c.ExpectString("Choose a color:")
				c.SendLine("?")
				c.ExpectString("My favourite color is red")
				// Select red.
				c.SendLine("")
				c.ExpectEOF()
			},
			"red",
		},
		{
			"Test Select prompt interaction with page size",
			&Select{
				Message:  "Choose a color:",
				Options:  []string{"red", "blue", "green"},
				PageSize: 1,
			},
			func(c *expect.Console) {
				c.ExpectString("Choose a color:")
				// Select green.
				c.SendLine(string(terminal.KeyArrowUp))
				c.ExpectEOF()
			},
			"green",
		},
		{
			"Test Select prompt interaction with vim mode",
			&Select{
				Message: "Choose a color:",
				Options: []string{"red", "blue", "green"},
				VimMode: true,
			},
			func(c *expect.Console) {
				c.ExpectString("Choose a color:")
				// Select blue.
				c.SendLine("j")
				c.ExpectEOF()
			},
			"blue",
		},
		{
			"Test Select prompt interaction with filter",
			&Select{
				Message: "Choose a color:",
				Options: []string{"red", "blue", "green"},
			},
			func(c *expect.Console) {
				c.ExpectString("Choose a color:")
				// Filter down to red and green.
				c.Send("re")
				// Select green.
				c.SendLine(string(terminal.KeyArrowDown))
				c.ExpectEOF()
			},
			"green",
		},
		{
			"Test Select prompt interaction with filter is case-insensitive",
			&Select{
				Message: "Choose a color:",
				Options: []string{"red", "blue", "green"},
			},
			func(c *expect.Console) {
				c.ExpectString("Choose a color:")
				// Filter down to red and green.
				c.Send("RE")
				// Select green.
				c.SendLine(string(terminal.KeyArrowDown))
				c.ExpectEOF()
			},
			"green",
		},
		{
			"Can select the first result in a filtered list if there is a default",
			&Select{
				Message: "Choose a color:",
				Options: []string{"red", "blue", "green"},
				Default: "blue",
			},
			func(c *expect.Console) {
				c.ExpectString("Choose a color:")
				// Make sure only red is showing
				c.SendLine("red")
				c.ExpectEOF()
			},
			"red",
		},
		{
			"Test Select prompt interaction with custom filter",
			&Select{
				Message: "Choose a color:",
				Options: []string{"red", "blue", "green"},
				FilterFn: func(filter string, options []string) (filtered []string) {
					result := DefaultFilterFn(filter, options)
					for _, v := range result {
						if len(v) >= 5 {
							filtered = append(filtered, v)
						}
					}
					return
				},
			},
			func(c *expect.Console) {
				c.ExpectString("Choose a color:")
				// Filter down to only green since custom filter only keeps options that are longer than 5 runes
				c.SendLine("re")
				c.ExpectEOF()
			},
			"green",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RunPromptTest(t, test)
		})
	}
}
