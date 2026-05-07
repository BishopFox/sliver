package mtypes

// A Credential structure describes a principle allowed to send or receive mail at the domain.
type Credential struct {
	CreatedAt RFC2822Time `json:"created_at"`
	Login     string      `json:"login"`
	Password  string      `json:"password"`
}

type CredentialsListResponse struct {
	// is -1 if Next() or First() have not been called
	TotalCount int          `json:"total_count"`
	Items      []Credential `json:"items"`
}
