package slack

// CardBlock defines a block of type card used to display a rich, self-contained
// piece of content with an optional hero image, icon (or Slack icon), title,
// subtitle, body, subtext, and action buttons. Cards can stand alone or be
// grouped inside a CarouselBlock.
//
// Only one of Icon and SlackIcon can be set, as they render in the same
// location.
//
// More Information: https://docs.slack.dev/reference/block-kit/blocks/card-block/
type CardBlock struct {
	Type      MessageBlockType   `json:"type"`
	BlockID   string             `json:"block_id,omitempty"`
	HeroImage *ImageBlockElement `json:"hero_image,omitempty"`
	Icon      *ImageBlockElement `json:"icon,omitempty"`
	SlackIcon *SlackIconObject   `json:"slack_icon,omitempty"`
	Title     *TextBlockObject   `json:"title,omitempty"`
	Subtitle  *TextBlockObject   `json:"subtitle,omitempty"`
	Body      *TextBlockObject   `json:"body,omitempty"`
	Subtext   *TextBlockObject   `json:"subtext,omitempty"`
	Actions   *BlockElements     `json:"actions,omitempty"`
}

// BlockType returns the type of the block
func (s CardBlock) BlockType() MessageBlockType {
	return s.Type
}

// ID returns the ID of the block
func (s CardBlock) ID() string {
	return s.BlockID
}

// CardBlockOption allows configuration of options for a new card block
type CardBlockOption func(*CardBlock)

// CardBlockOptionBlockID sets the block ID for the card block
func CardBlockOptionBlockID(blockID string) CardBlockOption {
	return func(block *CardBlock) {
		block.BlockID = blockID
	}
}

// NewCardBlock returns a new instance of a card block. Use the chainable
// With* methods or provide options to populate its fields.
func NewCardBlock(options ...CardBlockOption) *CardBlock {
	block := CardBlock{
		Type: MBTCard,
	}

	for _, option := range options {
		if option != nil {
			option(&block)
		}
	}

	return &block
}

// WithTitle sets the title text for the CardBlock
func (s *CardBlock) WithTitle(title *TextBlockObject) *CardBlock {
	s.Title = title
	return s
}

// WithSubtitle sets the subtitle text for the CardBlock
func (s *CardBlock) WithSubtitle(subtitle *TextBlockObject) *CardBlock {
	s.Subtitle = subtitle
	return s
}

// WithBody sets the body text for the CardBlock
func (s *CardBlock) WithBody(body *TextBlockObject) *CardBlock {
	s.Body = body
	return s
}

// WithSubtext sets the subtext displayed below the body of the CardBlock
func (s *CardBlock) WithSubtext(subtext *TextBlockObject) *CardBlock {
	s.Subtext = subtext
	return s
}

// WithIcon sets the icon image for the CardBlock. It is mutually exclusive with
// SlackIcon, as both render in the same location.
func (s *CardBlock) WithIcon(icon *ImageBlockElement) *CardBlock {
	s.Icon = icon
	return s
}

// WithSlackIcon sets the built-in Slack icon for the CardBlock. It is mutually
// exclusive with Icon, as both render in the same location.
func (s *CardBlock) WithSlackIcon(slackIcon *SlackIconObject) *CardBlock {
	s.SlackIcon = slackIcon
	return s
}

// WithHeroImage sets the hero image for the CardBlock
func (s *CardBlock) WithHeroImage(heroImage *ImageBlockElement) *CardBlock {
	s.HeroImage = heroImage
	return s
}

// WithActions sets the action buttons displayed at the bottom of the card
func (s *CardBlock) WithActions(elements ...BlockElement) *CardBlock {
	s.Actions = &BlockElements{ElementSet: elements}
	return s
}
