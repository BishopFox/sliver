package slack

// ImageBlock defines data required to display an image as a block element
//
// More Information: https://api.slack.com/reference/messaging/blocks#image
type ImageBlock struct {
	Type      MessageBlockType `json:"type"`
	ImageURL  string           `json:"image_url,omitempty"`
	AltText   string           `json:"alt_text"`
	BlockID   string           `json:"block_id,omitempty"`
	Title     *TextBlockObject `json:"title,omitempty"`
	SlackFile *SlackFileObject `json:"slack_file,omitempty"`
}

// ID returns the ID of the block
func (s ImageBlock) ID() string {
	return s.BlockID
}

// SlackFileObject Defines an object containing Slack file information to be used in an
// image block or image element.
//
// More Information: https://api.slack.com/reference/block-kit/composition-objects#slack_file
type SlackFileObject struct {
	ID  string `json:"id,omitempty"`
	URL string `json:"url,omitempty"`
}

// BlockType returns the type of the block
func (s ImageBlock) BlockType() MessageBlockType {
	return s.Type
}

// NewImageBlock returns an instance of a new Image Block type
func NewImageBlock(imageURL, altText, blockID string, title *TextBlockObject) *ImageBlock {
	return &ImageBlock{
		Type:     MBTImage,
		ImageURL: imageURL,
		AltText:  altText,
		BlockID:  blockID,
		Title:    title,
	}
}

// NewImageBlockSlackFile returns an instance of a new Image Block type
// TODO: BREAKING CHANGE - This should be combined with the function above
func NewImageBlockSlackFile(slackFile *SlackFileObject, altText string, blockID string, title *TextBlockObject) *ImageBlock {
	return &ImageBlock{
		Type:      MBTImage,
		SlackFile: slackFile,
		AltText:   altText,
		BlockID:   blockID,
		Title:     title,
	}
}
