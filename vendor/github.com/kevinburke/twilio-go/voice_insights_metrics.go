package twilio

import (
	"context"
	"fmt"
	"net/url"
)

const MetricsPathPart = "Metrics"

type CallMetricsService struct {
	callSid string
	client  *Client
}

type CallMetricsPage struct {
	Metrics []CallMetric `json:"metrics"`
	Meta    Meta         `json:"meta"`
}

type CallMetric struct {
	AccountSid  string       `json:"account_sid"`
	CallSid     string       `json:"call_sid"`
	CarrierEdge *EdgeMetrics `json:"carrier_edge,omitempty"`
	ClientEdge  *EdgeMetrics `json:"client_edge,omitempty"`
	Direction   string       `json:"direction"`
	Edge        string       `json:"edge"`
	SDKEdge     *EdgeMetrics `json:"sdk_edge,omitempty"`
	SIPEdge     *EdgeMetrics `json:"sip_edge,omitempty"`
	Timestamp   TwilioTime   `json:"timestamp"`
}

type EdgeMetadata struct {
	Region     string `json:"region"`
	ExternalIP string `json:"external_ip"`
	TwilioIP   string `json:"twilio_ip"`
}
type EdgeMetrics struct {
	Codec      int               `json:"codec"`
	CodecName  string            `json:"codec_name"`
	Cumulative CumulativeMetrics `json:"cumulative"`
	Interval   *IntervalMetrics  `json:"interval,omitempty"`
	Metadata   EdgeMetadata      `json:"metadata"`
}

type CumulativeMetrics struct {
	Jitter          MetricsSummary `json:"jitter"`
	PacketsReceived int            `json:"packets_received"`
	PacketsLost     int            `json:"packets_lost"`
}

type FloatValue struct {
	Value float64 `json:"value"`
}

type IntervalMetrics struct {
	AudioOut              FloatValue `json:"audio_out"`
	AudioIn               FloatValue `json:"audio_in"`
	Jitter                FloatValue `json:"jitter"`
	MOS                   FloatValue `json:"mos"`
	RTT                   FloatValue `json:"rtt"`
	PacketsReceived       int        `json:"packets_received"`
	PacketsLost           int        `json:"packets_lost"`
	PacketsLossPercentage float64    `json:"packets_loss_percentage"`
}

// Returns a list of metrics for a call. For more information on valid values,
// See https://www.twilio.com/docs/voice/insights/api/call-metrics-resource#get-call-metrics
func (s *CallMetricsService) GetPage(ctx context.Context, data url.Values) (*CallMetricsPage, error) {
	return s.GetPageIterator(data).Next(ctx)
}

type CallMetricsPageIterator struct {
	p *PageIterator
}

func (s *CallMetricsService) GetPageIterator(data url.Values) *CallMetricsPageIterator {
	iter := NewPageIterator(s.client, data, fmt.Sprintf("Voice/%s/%s", s.callSid, MetricsPathPart))
	return &CallMetricsPageIterator{
		p: iter,
	}
}

func (c *CallMetricsPageIterator) Next(ctx context.Context) (*CallMetricsPage, error) {
	cp := new(CallMetricsPage)
	err := c.p.Next(ctx, cp)
	if err != nil {
		return nil, err
	}
	c.p.SetNextPageURI(cp.Meta.NextPageURL)
	return cp, nil
}
