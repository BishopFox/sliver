package event

import (
	"encoding/json"
)

type MSC1767Audio struct {
	Duration int   `json:"duration"`
	Waveform []int `json:"waveform"`
}

type serializableMSC1767Audio MSC1767Audio

func (ma *MSC1767Audio) MarshalJSON() ([]byte, error) {
	if ma.Waveform == nil {
		ma.Waveform = []int{}
	}
	return json.Marshal((*serializableMSC1767Audio)(ma))
}

type MSC3245Voice struct{}
