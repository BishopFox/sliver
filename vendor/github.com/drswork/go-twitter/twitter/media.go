package twitter

import (
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/dghubble/sling"
)

// The size of a chunk to upload. There isn't any set size, so we
// choose 1M for convenience.
const chunkSize = 1024 * 1024

// This should really be fetched from the status endpoint, but the
// docs say 15M so we'll go with that.
const maxSize = 15 * 1024 * 1024

// MediaService provides methods for accessing twitter media APIs.
type MediaService struct {
	sling *sling.Sling
}

func newMediaService(sling *sling.Sling) *MediaService {
	return &MediaService{
		sling: sling.Path("media/"),
	}
}

type mediaInitResult struct {
	MediaID          int64  `json:"media_id"`
	MediaIDString    string `json:"media_id_string"`
	Size             int    `json:"size"`
	ExpiresAfterSecs int    `json:"expires_after_secs"`
}

type mediaInitParams struct {
	Command    string `url:"command"`
	TotalBytes int    `url:"total_bytes"`
	MediaType  string `url:"media_type"`
}

type mediaAppendParams struct {
	Command      string `url:"command"`
	MediaData    string `url:"media_data"`
	MediaID      int64  `url:"media_id"`
	SegmentIndex int    `url:"segment_index"`
}

// MediaVideoInfo holds information about media identified as videos.
type MediaVideoInfo struct {
	VideoType string `json:"video_type"`
}

// MediaProcessingInfo holds information about pending media uploads.
type MediaProcessingInfo struct {
	State           string                `json:"state"`
	CheckAfterSecs  int                   `json:"check_after_secs"`
	ProgressPercent int                   `json:"progress_percent"`
	Error           *MediaProcessingError `json:"error"`
}

// MediaProcessingError holds information about pending media
// processing failures.
type MediaProcessingError struct {
	Code    int    `json:"code"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

// MediaUploadResult holds information about a successfully completed
// media upload. Note that successful uploads may not be immediately
// usable if twitter is doing background processing on the uploaded
// media.
type MediaUploadResult struct {
	MediaID          int64                `json:"media_id"`
	MediaIDString    string               `json:"media_id_string"`
	Size             int                  `json:"size"`
	ExpiresAfterSecs int                  `json:"expires_after_secs"`
	Video            *MediaVideoInfo      `json:"video"`
	ProcessingInfo   *MediaProcessingInfo `json:"processing_info"`
}

type mediaFinalizeParams struct {
	Command string `url:"command"`
	MediaID int64  `url:"media_id"`
}

// Upload sends a piece of media to twitter. You must provide a byte
// slice containing the file contents and the MIME type of the
// file.
//
// This is a potentially asynchronous call, as some file types require
// extra processing by twitter. In those cases the returned
// MediaFinalizeResult will have the ProcessingInfo field set, and you
// can periodically poll Status with the MediaID to get the status of
// the upload.
func (m *MediaService) Upload(media []byte, mediaType string) (*MediaUploadResult, *http.Response, error) {

	if len(media) > maxSize {
		return nil, nil, fmt.Errorf("file size of %v exceeds twitter maximum %v", len(media), maxSize)
	}

	params := &mediaInitParams{
		Command:    "INIT",
		TotalBytes: len(media),
		MediaType:  mediaType,
	}
	res := new(mediaInitResult)
	apiError := new(APIError)

	resp, err := m.sling.New().Post("upload.json").BodyForm(params).Receive(res, apiError)

	if relevantError(err, *apiError) != nil {
		return nil, resp, relevantError(err, *apiError)
	}

	mediaID := res.MediaID

	segments := int(len(media) / chunkSize)
	for segment := 0; segment <= segments; segment++ {
		start := segment * chunkSize
		end := (segment + 1) * chunkSize
		if end > len(media) {
			end = len(media)
		}
		chunk := media[start:end]

		appendParams := &mediaAppendParams{
			Command:      "APPEND",
			MediaID:      mediaID,
			MediaData:    base64.StdEncoding.EncodeToString(chunk),
			SegmentIndex: segment,
		}

		resp, err = m.sling.New().Post("upload.json").BodyForm(appendParams).Receive(nil, apiError)

		if relevantError(err, *apiError) != nil {
			return nil, resp, relevantError(err, *apiError)
		}
	}

	finalizeParams := &mediaFinalizeParams{
		Command: "FINALIZE",
		MediaID: mediaID,
	}
	finalizeRes := new(MediaUploadResult)

	resp, err = m.sling.New().Post("upload.json").BodyForm(finalizeParams).Receive(finalizeRes, apiError)

	if relevantError(err, *apiError) != nil {
		return nil, resp, relevantError(err, *apiError)
	}

	return finalizeRes, resp, nil
}

// MediaStatusResult holds information about the current status of a
// piece of media.
type MediaStatusResult struct {
	MediaID          int                  `json:"media_id"`
	MediaIDString    string               `json:"media_id_string"`
	ExpiresAfterSecs int                  `json:"expires_after_secs"`
	ProcessingInfo   *MediaProcessingInfo `json:"processing_info"`
	Video            *MediaVideoInfo      `json:"video"`
}

type mediaStatusParams struct {
	Command string `url:"command"`
	MediaID int64  `url:"media_id"`
}

// Status returns the current status of the media specified by the
// media ID. It's only valid to call Status on a request where the
// Upload call returned something in ProcessingInfo.
// https://developer.twitter.com/en/docs/media/upload-media/api-reference/get-media-upload-status
func (m *MediaService) Status(mediaID int64) (*MediaStatusResult, *http.Response, error) {
	params := &mediaStatusParams{
		MediaID: mediaID,
		Command: "STATUS",
	}

	status := new(MediaStatusResult)
	apiError := new(APIError)
	resp, err := m.sling.New().Get("upload.json").QueryStruct(params).Receive(status, apiError)
	return status, resp, relevantError(err, *apiError)

}
