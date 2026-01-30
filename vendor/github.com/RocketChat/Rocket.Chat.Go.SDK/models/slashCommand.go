package models

type SlashCommand struct {
	Command         string `json:"command"`
	Params          string `json:"params,omitempty"`
	Description     string `json:"description,omitempty"`
	ClientOnly      bool   `json:"clientOnly"`
	ProvidesPreview bool   `json:"providesPreview"`
}
