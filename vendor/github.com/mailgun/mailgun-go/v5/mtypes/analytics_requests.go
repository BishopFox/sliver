package mtypes

type MetricsRequest struct {
	// A start date (default: 7 days before current time).
	Start RFC2822Time `json:"start,omitempty"`
	// An end date (default: current time).
	End RFC2822Time `json:"end,omitempty"`
	// A resolution in the format of 'day' 'hour' 'month'. Default is day.
	Resolution Resolution `json:"resolution,omitempty"`
	// A duration in the format of '1d' '2h' '2m'.
	// If duration is provided then it is calculated from the end date and overwrites the start date.
	Duration string `json:"duration,omitempty"`
	// Attributes of the metric data such as 'time' 'domain' 'ip' 'ip_pool' 'recipient_domain' 'tag' 'country' 'subaccount'.
	Dimensions []string `json:"dimensions,omitempty"`
	// Name of the metrics to receive the stats for such as 'accepted_count' 'delivered_count' 'accepted_rate'.
	Metrics []string `json:"metrics,omitempty"`
	// Filters to apply to the query.
	Filter MetricsFilterPredicateGroup `json:"filter,omitempty"`
	// Include stats from all subaccounts.
	IncludeSubaccounts bool `json:"include_subaccounts,omitempty"`
	// Include top-level aggregate metrics.
	IncludeAggregates bool `json:"include_aggregates,omitempty"`
	// Attributes used for pagination and sorting.
	Pagination MetricsPagination `json:"pagination,omitempty"`
}

type MetricsLabeledValue struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type MetricsFilterPredicate struct {
	Attribute     string                `json:"attribute"`
	Comparator    string                `json:"comparator"`
	LabeledValues []MetricsLabeledValue `json:"values,omitempty"`
}

type MetricsFilterPredicateGroup struct {
	BoolGroupAnd []MetricsFilterPredicate `json:"AND,omitempty"`
}
