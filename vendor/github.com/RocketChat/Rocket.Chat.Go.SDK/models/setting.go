package models

type Setting struct {
	ID           string  `json:"_id"`
	Blocked      bool    `json:"blocked"`
	Group        string  `json:"group"`
	Hidden       bool    `json:"hidden"`
	Public       bool    `json:"public"`
	Type         string  `json:"type"`
	PackageValue string  `json:"packageValue"`
	Sorter       int     `json:"sorter"`
	Value        string  `json:"value"`
	ValueBool    bool    `json:"valueBool"`
	ValueInt     float64 `json:"valueInt"`
	ValueSource  string  `json:"valueSource"`
	ValueAsset   Asset   `json:"asset"`
}

type Asset struct {
	DefaultUrl string `json:"defaultUrl"`
}
