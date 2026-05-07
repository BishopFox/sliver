package twitter

import (
	"net/http"

	"github.com/dghubble/sling"
)

// Config represents the current configuration used by Twitter
// https://developer.twitter.com/en/docs/developer-utilities/configuration
type Config struct {
	CharactersReservedPerMedia int         `json:"characters_reserved_per_media"`
	DMTextCharacterLimit       int         `json:"dm_text_character_limit"`
	MaxMediaPerUpload          int         `json:"max_media_per_upload"`
	PhotoSizeLimit             int         `json:"photo_size_limit"`
	PhotoSizes                 *PhotoSizes `json:"photo_sizes"`
	ShortURLLength             int         `json:"short_url_length"`
	ShortURLLengthHTTPS        int         `json:"short_url_length_https"`
	NonUsernamePaths           []string    `json:"non_username_paths"`
}

// PhotoSizes holds data for the four sizes of images that twitter supports.
type PhotoSizes struct {
	Large  *SinglePhotoSize `json:"large"`
	Medium *SinglePhotoSize `json:"medium"`
	Small  *SinglePhotoSize `json:"small"`
	Thumb  *SinglePhotoSize `json:"thumb"`
}

// SinglePhotoSize holds the information for a single photo size.
type SinglePhotoSize struct {
	Height int    `json:"h"`
	Width  int    `json:"w"`
	Resize string `json:"resize"`
}

// ConfigService provides methods for accessing Twitter's config API endpoint.
type ConfigService struct {
	sling *sling.Sling
}

// newConfigService returns a new ConfigService.
func newConfigService(sling *sling.Sling) *ConfigService {
	return &ConfigService{
		sling: sling.Path("help/"),
	}
}

// Get fetches the current configuration.
func (c *ConfigService) Get() (*Config, *http.Response, error) {
	config := new(Config)
	apiError := new(APIError)
	resp, err := c.sling.New().Get("configuration.json").Receive(config, apiError)
	return config, resp, relevantError(err, *apiError)
}
