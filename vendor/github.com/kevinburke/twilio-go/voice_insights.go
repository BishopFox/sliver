package twilio

type VoiceInsightsService struct {
	Summary *CallSummaryService
	Metrics *CallMetricsService
	Events  *CallEventsService
}
