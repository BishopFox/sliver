package twilio

import (
	"context"
	"fmt"
	"net/url"
)

const SummaryPathPart = "Summary"

type CallSummaryService struct {
	callSid string
	client  *Client
}

type CallSummary struct {
	AccountSid      string                `json:"account_sid"`
	Attributes      CallSummaryAttributes `json:"attributes"`
	CallSid         string                `json:"call_sid"`
	CallState       Status                `json:"call_state"`
	CallType        string                `json:"call_type"`
	CarrierEdge     *Edge                 `json:"carrier_edge,omitempty"`
	ClientEdge      *Edge                 `json:"client_edge,omitempty"`
	ConnectDuration int                   `json:"connect_duration"`
	Duration        int                   `json:"duration"`
	EndTime         TwilioTime            `json:"end_time"`
	From            CallParty             `json:"from"`
	ProcessingState string                `json:"processing_state"`
	Properties      CallProperties        `json:"properties"`
	SDKEdge         *SDKEdge              `json:"sdk_edge,omitempty"`
	SIPEdge         *Edge                 `json:"sip_edge,omitempty"`
	StartTime       TwilioTime            `json:"start_time"`
	Tags            []string              `json:"tags"`
	To              CallParty             `json:"to"`
	URL             string                `json:"url"`
}

type CallSummaryAttributes struct {
	ConferenceParticipant bool `json:"conference_participant"`
}

type Edge struct {
	Metrics struct {
		Inbound  EdgeSummary `json:"inbound"`
		Outbound EdgeSummary `json:"outbound"`
	} `json:"metrics"`
	Properties EdgeProperties `json:"properties"`
}

type EdgeSummary struct {
	Codec                 int            `json:"codec"`
	CodecName             string         `json:"codec_name"`
	PacketsReceived       int            `json:"packets_received"`
	PacketsSent           int            `json:"packets_sent"`
	PacketsLost           int            `json:"packets_lost"`
	PacketsLossPercentage float64        `json:"packets_loss_percentage"`
	Jitter                MetricsSummary `json:"jitter"`
	Latency               MetricsSummary `json:"latency"`
	PacketDelayVariation  map[string]int `json:"packet_delay_variation"`
}

type EdgeProperties struct {
	Direction           string `json:"direction"`
	MediaRegion         string `json:"media_region"`
	SignalingRegion     string `json:"signaling_region"`
	TwilioMediaIP       string `json:"twilio_media_ip"`
	TwilioSignalingIP   string `json:"twilio_signaling_ip"`
	ExternalMediaIP     string `json:"external_media_ip"`
	ExternalSignalingIP string `json:"external_signaling_ip"`
	SIPCallID           string `json:"sip_call_id"`
	UserAgent           string `json:"user_agent"`
	SelectedRegion      string `json:"selected_region"`
	Region              string `json:"region"`
	DisconnectedBy      string `json:"disconnected_by"`
	TrunkSID            string `json:"trunk_sid"`
}

type SDKEdge struct {
	Metrics struct {
		Inbound  SDKEdgeSummary `json:"inbound"`
		Outbound SDKEdgeSummary `json:"outbound"`
	} `json:"metrics"`
	Properties SDKEdgeProperties `json:"properties"`
	Events     SDKEdgeEvents     `json:"events"`
}

type SDKEdgeSummary struct {
	EdgeSummary
	MOS      MetricsSummary `json:"mos"`
	RTT      MetricsSummary `json:"rtt"`
	Tags     []string       `json:"tags"`
	AudioOut MetricsSummary `json:"audio_out"`
	AudioIn  MetricsSummary `json:"audio_in"`
}

type SDKEdgeProperties struct {
	Direction string `json:"direction"`
	Settings  struct {
		DSCP              bool     `json:"dscp"`
		ICERestartEnabled bool     `json:"ice_restart_enabled"`
		Edge              string   `json:"edge"`
		SelectedEdges     []string `json:"selected_edges"`
	} `json:"settings"`
}

type SDKEdgeEvents struct {
	Levels map[string]int `json:"levels"`
	Groups map[string]int `json:"groups"`
	Errors map[string]int `json:"errors"`
}

type MetricsSummary struct {
	Max float64 `json:"max"`
	Avg float64 `json:"avg"`
	Min float64 `json:"min"`
}

type CallProperties struct {
	Direction          Direction `json:"direction"`
	DisconnectedBy     string    `json:"disconnected_by"`
	PostDialDelayMs    int       `json:"pdd_ms"`
	LastSIPResponseNum int       `json:"last_sip_response_num"`
}

type CallParty struct {
	Callee             string `json:"callee"`
	Caller             string `json:"caller"`
	Carrier            string `json:"carrier"`
	Connection         string `json:"connection"`
	CountryCode        string `json:"country_code"`
	CountrySubdivision string `json:"country_subdivision"`
	City               string `json:"city"`
	Location           struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	} `json:"location"`
	IPAddress    string `json:"ip_address"`
	NumberPrefix string `json:"number_prefix"`
}

// Gets a call summary resource from the Voice Insights API.
// See https://www.twilio.com/docs/voice/insights/api/call-summary-resource#get-call-summary
func (s *CallSummaryService) Get(ctx context.Context) (*CallSummary, error) {
	summary := new(CallSummary)
	err := s.client.ListResource(ctx, fmt.Sprintf("Voice/%s/%s", s.callSid, SummaryPathPart), url.Values{}, summary)
	return summary, err
}

// Gets a partial call summary resource from the Voice Insights API.
// A completed call summary may take up to a half hour to generate,
// but a partial summary record will be available within ten minutes of a call ending.
// See https://www.twilio.com/docs/voice/insights/api/call-summary-resource#get-call-summary
func (s *CallSummaryService) GetPartial(ctx context.Context) (*CallSummary, error) {
	params := url.Values{}
	params.Add("ProcessingState", "partial")

	summary := new(CallSummary)
	err := s.client.ListResource(ctx, fmt.Sprintf("Voice/%s/%s", s.callSid, SummaryPathPart), params, summary)
	return summary, err
}
