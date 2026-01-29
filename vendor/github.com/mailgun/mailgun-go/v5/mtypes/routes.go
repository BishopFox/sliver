package mtypes

// A Route structure contains information on a configured or to-be-configured route.
// When creating a new route, the SDK only uses a subset of the fields of this structure.
// In particular, CreatedAt and ID are meaningless in this context, and will be ignored.
// Only Priority, Description, Expression, and Actions need be provided.
type Route struct {
	// The Priority field indicates how soon the route works relative to other configured routes.
	// Routes of equal priority are consulted in chronological order.
	Priority int `json:"priority,omitempty"`
	// The Description field provides a human-readable description for the route.
	// Mailgun ignores this field except to provide the description when viewing the Mailgun web control panel.
	Description string `json:"description,omitempty"`
	// The Expression field lets you specify a pattern to match incoming messages against.
	Expression string `json:"expression,omitempty"`
	// The Actions field contains strings specifying what to do
	// with any message which matches the provided expression.
	Actions []string `json:"actions,omitempty"`

	// The CreatedAt field provides a time-stamp for when the route came into existence.
	CreatedAt RFC2822Time `json:"created_at,omitempty"`
	// ID field provides a unique identifier for this route.
	Id string `json:"id,omitempty"`
}

type RoutesListResponse struct {
	// is -1 if Next() or First() have not been called
	TotalCount int     `json:"total_count"`
	Items      []Route `json:"items"`
}

type CreateRouteResp struct {
	Message string `json:"message"`
	Route   `json:"route"`
}
