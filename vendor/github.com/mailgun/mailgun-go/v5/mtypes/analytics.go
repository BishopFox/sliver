package mtypes

type MetricsPagination struct {
	// Colon-separated value indicating column name and sort direction e.g. 'domain:asc'.
	Sort string `json:"sort"`
	// The number of items to skip over when satisfying the request.
	// To get the first page of data set skip to zero.
	// Then increment the skip by the limit for subsequent calls.
	Skip int `json:"skip"`
	// The maximum number of items returned in the response.
	Limit int `json:"limit"`
	// The total number of items in the query result set.
	Total int `json:"total"`
}
