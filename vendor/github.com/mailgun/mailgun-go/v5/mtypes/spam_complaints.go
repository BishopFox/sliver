package mtypes

// Complaint structures track how many times one of your emails have been marked as spam.
// the recipient thought your messages were not solicited.
type Complaint struct {
	Count     int         `json:"count"`
	CreatedAt RFC2822Time `json:"created_at"`
	Address   string      `json:"address"`
}

type ComplaintsResponse struct {
	Paging Paging      `json:"paging"`
	Items  []Complaint `json:"items"`
}
