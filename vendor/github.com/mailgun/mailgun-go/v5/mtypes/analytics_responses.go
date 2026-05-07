package mtypes

type MetricsResponse struct {
	Start      RFC2822Time       `json:"start"`
	End        RFC2822Time       `json:"end"`
	Resolution Resolution        `json:"resolution"`
	Duration   string            `json:"duration"`
	Dimensions []string          `json:"dimensions"`
	Aggregates MetricsAggregates `json:"aggregates"`
	Items      []MetricsItem     `json:"items"`
	Pagination MetricsPagination `json:"pagination"`
}

type MetricsItem struct {
	Dimensions []MetricsDimension `json:"dimensions"`
	Metrics    Metrics            `json:"metrics"`
}

type MetricsAggregates struct {
	Metrics Metrics `json:"metrics"`
}

type Metrics struct {
	AcceptedIncomingCount         *uint64 `json:"accepted_incoming_count,omitempty"`
	AcceptedOutgoingCount         *uint64 `json:"accepted_outgoing_count,omitempty"`
	AcceptedCount                 *uint64 `json:"accepted_count,omitempty"`
	DeliveredSMTPCount            *uint64 `json:"delivered_smtp_count,omitempty"`
	DeliveredHTTPCount            *uint64 `json:"delivered_http_count,omitempty"`
	DeliveredOptimizedCount       *uint64 `json:"delivered_optimized_count,omitempty"`
	DeliveredCount                *uint64 `json:"delivered_count,omitempty"`
	StoredCount                   *uint64 `json:"stored_count,omitempty"`
	ProcessedCount                *uint64 `json:"processed_count,omitempty"`
	SentCount                     *uint64 `json:"sent_count,omitempty"`
	OpenedCount                   *uint64 `json:"opened_count,omitempty"`
	ClickedCount                  *uint64 `json:"clicked_count,omitempty"`
	UniqueOpenedCount             *uint64 `json:"unique_opened_count,omitempty"`
	UniqueClickedCount            *uint64 `json:"unique_clicked_count,omitempty"`
	UnsubscribedCount             *uint64 `json:"unsubscribed_count,omitempty"`
	ComplainedCount               *uint64 `json:"complained_count,omitempty"`
	FailedCount                   *uint64 `json:"failed_count,omitempty"`
	TemporaryFailedCount          *uint64 `json:"temporary_failed_count,omitempty"`
	PermanentFailedCount          *uint64 `json:"permanent_failed_count,omitempty"`
	TemporaryFailedESPBlockCount  *uint64 `json:"temporary_failed_esp_block_count,omitempty"`
	PermanentFailedESPBlockCount  *uint64 `json:"permanent_failed_esp_block_count,omitempty"`
	WebhookCount                  *uint64 `json:"webhook_count,omitempty"`
	PermanentFailedOptimizedCount *uint64 `json:"permanent_failed_optimized_count,omitempty"`
	PermanentFailedOldCount       *uint64 `json:"permanent_failed_old_count,omitempty"`
	BouncedCount                  *uint64 `json:"bounced_count,omitempty"`
	HardBouncesCount              *uint64 `json:"hard_bounces_count,omitempty"`
	SoftBouncesCount              *uint64 `json:"soft_bounces_count,omitempty"`
	DelayedBounceCount            *uint64 `json:"delayed_bounce_count,omitempty"`
	SuppressedBouncesCount        *uint64 `json:"suppressed_bounces_count,omitempty"`
	SuppressedUnsubscribedCount   *uint64 `json:"suppressed_unsubscribed_count,omitempty"`
	SuppressedComplaintsCount     *uint64 `json:"suppressed_complaints_count,omitempty"`
	DeliveredFirstAttemptCount    *uint64 `json:"delivered_first_attempt_count,omitempty"`
	DelayedFirstAttemptCount      *uint64 `json:"delayed_first_attempt_count,omitempty"`
	DeliveredSubsequentCount      *uint64 `json:"delivered_subsequent_count,omitempty"`
	DeliveredTwoPlusAttemptsCount *uint64 `json:"delivered_two_plus_attempts_count,omitempty"`

	DeliveredRate     string `json:"delivered_rate,omitempty"`
	OpenedRate        string `json:"opened_rate,omitempty"`
	ClickedRate       string `json:"clicked_rate,omitempty"`
	UniqueOpenedRate  string `json:"unique_opened_rate,omitempty"`
	UniqueClickedRate string `json:"unique_clicked_rate,omitempty"`
	UnsubscribedRate  string `json:"unsubscribed_rate,omitempty"`
	ComplainedRate    string `json:"complained_rate,omitempty"`
	BounceRate        string `json:"bounce_rate,omitempty"`
	FailRate          string `json:"fail_rate,omitempty"`
	PermanentFailRate string `json:"permanent_fail_rate,omitempty"`
	TemporaryFailRate string `json:"temporary_fail_rate,omitempty"`
	DelayedRate       string `json:"delayed_rate,omitempty"`

	// usage metrics
	EmailValidationCount        *uint64 `json:"email_validation_count,omitempty"`
	EmailValidationPublicCount  *uint64 `json:"email_validation_public_count,omitempty"`
	EmailValidationValidCount   *uint64 `json:"email_validation_valid_count,omitempty"`
	EmailValidationSingleCount  *uint64 `json:"email_validation_single_count,omitempty"`
	EmailValidationBulkCount    *uint64 `json:"email_validation_bulk_count,omitempty"`
	EmailValidationListCount    *uint64 `json:"email_validation_list_count,omitempty"`
	EmailValidationMailgunCount *uint64 `json:"email_validation_mailgun_count,omitempty"`
	EmailValidationMailjetCount *uint64 `json:"email_validation_mailjet_count,omitempty"`
	EmailPreviewCount           *uint64 `json:"email_preview_count,omitempty"`
	EmailPreviewFailedCount     *uint64 `json:"email_preview_failed_count,omitempty"`
	LinkValidationCount         *uint64 `json:"link_validation_count,omitempty"`
	LinkValidationFailedCount   *uint64 `json:"link_validation_failed_count,omitempty"`
	SeedTestCount               *uint64 `json:"seed_test_count,omitempty"`
}

type MetricsDimension struct {
	// The dimension
	Dimension string `json:"dimension"`
	// The dimension value
	Value string `json:"value"`
	// The dimension value in displayable form
	DisplayValue string `json:"display_value"`
}
