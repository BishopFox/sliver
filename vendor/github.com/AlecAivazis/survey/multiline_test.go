package survey

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"

	expect "github.com/Netflix/go-expect"
	"github.com/stretchr/testify/assert"
	"gopkg.in/AlecAivazis/survey.v1/core"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

func init() {
	// disable color output for all prompts to simplify testing
	core.DisableColor = true
}

func TestMultilineRender(t *testing.T) {

	tests := []struct {
		title    string
		prompt   Multiline
		data     MultilineTemplateData
		expected string
	}{
		{
			"Test Multiline question output without default",
			Multiline{Message: "What is your favorite month:"},
			MultilineTemplateData{},
			fmt.Sprintf("%s What is your favorite month: [Enter 2 empty lines to finish]", core.QuestionIcon),
		},
		{
			"Test Multiline question output with default",
			Multiline{Message: "What is your favorite month:", Default: "April"},
			MultilineTemplateData{},
			fmt.Sprintf("%s What is your favorite month: (April) [Enter 2 empty lines to finish]", core.QuestionIcon),
		},
		{
			"Test Multiline answer output",
			Multiline{Message: "What is your favorite month:"},
			MultilineTemplateData{Answer: "October", ShowAnswer: true},
			fmt.Sprintf("%s What is your favorite month: \nOctober", core.QuestionIcon),
		},
		{
			"Test Multiline question output without default but with help hidden",
			Multiline{Message: "What is your favorite month:", Help: "This is helpful"},
			MultilineTemplateData{},
			fmt.Sprintf("%s What is your favorite month: [Enter 2 empty lines to finish]", string(core.HelpInputRune)),
		},
		{
			"Test Multiline question output with default and with help hidden",
			Multiline{Message: "What is your favorite month:", Default: "April", Help: "This is helpful"},
			MultilineTemplateData{},
			fmt.Sprintf("%s What is your favorite month: (April) [Enter 2 empty lines to finish]", string(core.HelpInputRune)),
		},
		{
			"Test Multiline question output without default but with help shown",
			Multiline{Message: "What is your favorite month:", Help: "This is helpful"},
			MultilineTemplateData{ShowHelp: true},
			fmt.Sprintf("%s This is helpful\n%s What is your favorite month: [Enter 2 empty lines to finish]", core.HelpIcon, core.QuestionIcon),
		},
		{
			"Test Multiline question output with default and with help shown",
			Multiline{Message: "What is your favorite month:", Default: "April", Help: "This is helpful"},
			MultilineTemplateData{ShowHelp: true},
			fmt.Sprintf("%s This is helpful\n%s What is your favorite month: (April) [Enter 2 empty lines to finish]", core.HelpIcon, core.QuestionIcon),
		},
	}

	for _, test := range tests {
		r, w, err := os.Pipe()
		assert.Nil(t, err, test.title)

		test.prompt.WithStdio(terminal.Stdio{Out: w})
		test.data.Multiline = test.prompt
		err = test.prompt.Render(
			MultilineQuestionTemplate,
			test.data,
		)
		assert.Nil(t, err, test.title)

		w.Close()
		var buf bytes.Buffer
		io.Copy(&buf, r)

		assert.Contains(t, buf.String(), test.expected, test.title)
	}
}

func TestMultilinePrompt(t *testing.T) {
	tests := []PromptTest{
		{
			"Test Multiline prompt interaction",
			&Multiline{
				Message: "What is your name?",
			},
			func(c *expect.Console) {
				c.ExpectString("What is your name?")
				c.SendLine("Larry Bird\nI guess...\nnot sure\n\n")
				c.ExpectEOF()
			},
			"Larry Bird\nI guess...\nnot sure",
		},
		{
			"Test Multiline prompt interaction with default",
			&Multiline{
				Message: "What is your name?",
				Default: "Johnny Appleseed",
			},
			func(c *expect.Console) {
				c.ExpectString("What is your name?")
				c.SendLine("\n\n")
				c.ExpectEOF()
			},
			"Johnny Appleseed",
		},
		{
			"Test Multiline prompt interaction overriding default",
			&Multiline{
				Message: "What is your name?",
				Default: "Johnny Appleseed",
			},
			func(c *expect.Console) {
				c.ExpectString("What is your name?")
				c.SendLine("Larry Bird\n\n")
				c.ExpectEOF()
			},
			"Larry Bird",
		},
		{
			"Test Multiline does not implement help interaction",
			&Multiline{
				Message: "What is your name?",
				Help:    "It might be Satoshi Nakamoto",
			},
			func(c *expect.Console) {
				c.ExpectString("What is your name?")
				c.SendLine("?")
				c.SendLine("Satoshi Nakamoto\n\n")
				c.ExpectEOF()
			},
			"?\nSatoshi Nakamoto",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RunPromptTest(t, test)
		})
	}
}
