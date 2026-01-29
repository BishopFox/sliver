package mtypes

// Mailing list members have an attribute that determines if they've subscribed to the mailing list or not.
// This attribute may be used to filter the results returned by GetSubscribers().
// All, Subscribed, and Unsubscribed provides a convenient and readable syntax for specifying the scope of the search.
// We use a pointer to boolean as a kind of trinary data type:
// if nil, the relevant data type remains unspecified.
// Otherwise, its value is either true or false.
var (
	All          *bool
	Subscribed   = ptr(true)
	Unsubscribed = ptr(false)
)

// A Member structure represents a member of the mailing list.
// The Vars field can represent any JSON-encodable data.
type Member struct {
	Address    string         `json:"address,omitempty"`
	Name       string         `json:"name,omitempty"`
	Subscribed *bool          `json:"subscribed,omitempty"`
	Vars       map[string]any `json:"vars,omitempty"`
}

type MemberListResponse struct {
	Lists  []Member `json:"items"`
	Paging Paging   `json:"paging"`
}

type MemberResponse struct {
	Member Member `json:"member"`
}
