package console

import (
	"fmt"
	"sliver/client/assets"
	"sort"

	"github.com/AlecAivazis/survey"
)

func selectConfig() *assets.ClientConfig {

	configs := assets.GetConfigs()
	if len(configs) == 0 {
		return nil
	}

	answer := struct {
		Config string
	}{}
	qs := getPromptForConfigs(configs)
	err := survey.Ask(qs, &answer)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	return configs[answer.Config]
}

func getPromptForConfigs(configs map[string]*assets.ClientConfig) []*survey.Question {

	keys := []string{}
	for k := range configs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return []*survey.Question{
		{
			Name: "config",
			Prompt: &survey.Select{
				Message: "Select a server:",
				Options: keys,
				Default: keys[0],
			},
		},
	}
}
