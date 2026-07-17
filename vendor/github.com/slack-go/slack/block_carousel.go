package slack

// CarouselBlock defines a block of type carousel that displays a scrollable
// list of cards. A carousel must contain between 1 and 10 cards.
//
// More Information: https://docs.slack.dev/reference/block-kit/blocks/carousel-block/
type CarouselBlock struct {
	Type     MessageBlockType `json:"type"`
	BlockID  string           `json:"block_id,omitempty"`
	Elements []*CardBlock     `json:"elements"`
}

// BlockType returns the type of the block
func (s CarouselBlock) BlockType() MessageBlockType {
	return s.Type
}

// ID returns the ID of the block
func (s CarouselBlock) ID() string {
	return s.BlockID
}

// NewCarouselBlock returns a new instance of a carousel block containing the
// given cards.
func NewCarouselBlock(cards ...*CardBlock) *CarouselBlock {
	return &CarouselBlock{
		Type:     MBTCarousel,
		Elements: cards,
	}
}

// WithBlockID sets the block ID for the CarouselBlock
func (s *CarouselBlock) WithBlockID(blockID string) *CarouselBlock {
	s.BlockID = blockID
	return s
}

// AddCard appends a card to the carousel
func (s *CarouselBlock) AddCard(card *CardBlock) *CarouselBlock {
	s.Elements = append(s.Elements, card)
	return s
}
