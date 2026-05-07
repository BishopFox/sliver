package mtypes

// A Subaccount structure holds information about a subaccount.
type Subaccount struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type SubaccountResponse struct {
	Item Subaccount `json:"subaccount"`
}

type ListSubaccountsResponse struct {
	Items []Subaccount `json:"subaccounts"`
	Total int          `json:"total"`
}
