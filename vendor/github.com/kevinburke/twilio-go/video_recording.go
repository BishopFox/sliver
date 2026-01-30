package twilio

import (
	"context"
	"net/url"
	"strings"
)

type VideoRecordingService struct {
	client *Client
}

const videoRecordingsPathPart = "Recordings"

func videoMediaPathPart(recordingSid string) string {
	return strings.Join([]string{videoRecordingsPathPart, recordingSid, "Media"}, "/")
}

type VideoRecording struct {
	Sid             string            `json:"sid"`
	Duration        uint              `json:"duration"`
	Status          Status            `json:"status"`
	DateCreated     TwilioTime        `json:"date_created"`
	SourceSid       string            `json:"source_sid"`
	URI             string            `json:"uri"`
	Size            uint              `json:"size"`
	Type            string            `json:"type"`
	ContainerFormat string            `json:"container_format"`
	Codec           string            `json:"codec"`
	GroupingSids    map[string]string `json:"grouping_sids"`
	Links           map[string]string `json:"links"`
}

type VideoMedia struct {
	Location string `json:"location"`
}

type VideoRecordingPage struct {
	Meta       Meta              `json:"meta"`
	Recordings []*VideoRecording `json:"recordings"`
}

type VideoRecordingPageIterator struct {
	p *PageIterator
}

// When you make a request to this URL, Twilio will generate a temporary URL for accessing
// this binary data, and issue an HTTP 302 redirect response to your request. The Recording
// will be returned in the format as described in the metadata.
func (vr *VideoRecordingService) Media(ctx context.Context, sid string) (*VideoMedia, error) {
	media := new(VideoMedia)
	path := videoMediaPathPart(sid)
	err := vr.client.ListResource(ctx, path, nil, media)
	return media, err
}

// Returns the VideoRecording with the given sid.
func (vr *VideoRecordingService) Get(ctx context.Context, sid string) (*VideoRecording, error) {
	recording := new(VideoRecording)
	err := vr.client.GetResource(ctx, videoRecordingsPathPart, sid, recording)
	return recording, err
}

// Delete the VideoRecording with the given sid. If the VideoRecording has already been
// deleted, or does not exist, Delete returns nil. If another error or a
// timeout occurs, the error is returned.
func (vr *VideoRecordingService) Delete(ctx context.Context, sid string) error {
	return vr.client.DeleteResource(ctx, videoRecordingsPathPart, sid)
}

// Returns a list of recordings. For more information on valid values,
// see https://www.twilio.com/docs/api/video/recordings-resource#recordings-list-resource
func (vr *VideoRecordingService) GetPage(ctx context.Context, data url.Values) (*VideoRecordingPage, error) {
	return vr.GetPageIterator(data).Next(ctx)
}

// GetPageIterator returns an iterator which can be used to retrieve pages.
func (vr *VideoRecordingService) GetPageIterator(data url.Values) *VideoRecordingPageIterator {
	iter := NewPageIterator(vr.client, data, videoRecordingsPathPart)
	return &VideoRecordingPageIterator{
		p: iter,
	}
}

// Next returns the next page of resources. If there are no more resources,
// NoMoreResults is returned.
func (vr *VideoRecordingPageIterator) Next(ctx context.Context) (*VideoRecordingPage, error) {
	vrp := new(VideoRecordingPage)
	err := vr.p.Next(ctx, vrp)
	if err != nil {
		return nil, err
	}
	vr.p.SetNextPageURI(vrp.Meta.NextPageURL)
	return vrp, nil
}
