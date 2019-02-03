package console

import (
	"encoding/json"
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

func getNewConfig() *assets.ClientConfig {
	for {
		text := ""
		prompt := &survey.Multiline{
			Message: "Paste new config",
		}
		survey.AskOne(prompt, &text, nil)
		conf := assets.ClientConfig{}
		err := json.Unmarshal([]byte(text), &conf)
		if err != nil {
			return &conf
		}
	}
}
