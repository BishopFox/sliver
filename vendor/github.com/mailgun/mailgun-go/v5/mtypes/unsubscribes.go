package mtypes

type Unsubscribe struct {
	CreatedAt RFC2822Time `json:"created_at,omitempty"`
	Tags      []string    `json:"tags,omitempty"`
	ID        string      `json:"id,omitempty"`
	Address   string      `json:"address"`
}

type ListUnsubscribesResponse struct {
	Paging Paging        `json:"paging"`
	Items  []Unsubscribe `json:"items"`
}
