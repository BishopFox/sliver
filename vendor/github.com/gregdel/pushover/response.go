package pushover

import "fmt"

// Response represents a response from the API.
type Response struct {
	Status  int    `json:"status"`
	ID      string `json:"request"`
	Errors  Errors `json:"errors"`
	Receipt string `json:"receipt"`
	Group   string `json:"group"`
	Limit   *Limit
}

// String represents a printable form of the response.
func (r Response) String() string {
	ret := fmt.Sprintf("Status: %d\n", r.Status)
	ret += fmt.Sprintf("Request id: %s\n", r.ID)
	if r.Receipt != "" {
		ret += fmt.Sprintf("Receipt: %s\n", r.Receipt)
	}
	if r.Group != "" {
		ret += fmt.Sprintf("Group: %s\n", r.Group)
	}
	if r.Limit != nil {
		ret += fmt.Sprintf("Usage %d/%d messages\nNext reset : %s",
			r.Limit.Remaining, r.Limit.Total, r.Limit.NextReset)
	}
	return ret
}
