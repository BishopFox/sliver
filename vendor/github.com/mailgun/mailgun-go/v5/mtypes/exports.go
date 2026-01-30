package mtypes

type ExportList struct {
	Items []Export `json:"items"`
}

type Export struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	URL    string `json:"url"`
}
