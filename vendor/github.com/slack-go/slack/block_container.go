package slack

import "fmt"

// ContainerWidth controls the rendered width of a ContainerBlock. When unset,
// Slack defaults to ContainerWidthStandard.
type ContainerWidth string

const (
	ContainerWidthNarrow   ContainerWidth = "narrow"
	ContainerWidthStandard ContainerWidth = "standard"
	ContainerWidthWide     ContainerWidth = "wide"
	ContainerWidthFull     ContainerWidth = "full"
)

// containerMaxChildBlocks is the maximum number of child blocks Slack permits
// inside a container block.
const containerMaxChildBlocks = 10

// ContainerBlock groups a set of child blocks so they render together as a
// single, optionally collapsible, unit with a title, subtitle, and icon.
//
// More Information: https://docs.slack.dev/reference/block-kit/blocks/container-block/
type ContainerBlock struct {
	Type    MessageBlockType `json:"type"`
	BlockID string           `json:"block_id,omitempty"`
	// Title is the container heading rendered as a plain_text object. One of
	// Title or RichTextTitle is required; RichTextTitle takes precedence when
	// both are set. Slack requires a maximum of 150 characters.
	Title *TextBlockObject `json:"title,omitempty"`
	// RichTextTitle is the container heading rendered as a rich_text block. It
	// takes precedence over Title when both are set.
	RichTextTitle *RichTextBlock `json:"rich_text_title,omitempty"`
	// Subtitle is descriptive text below the title, rendered as a plain_text or
	// mrkdwn object. Slack requires a maximum of 150 characters.
	Subtitle *TextBlockObject `json:"subtitle,omitempty"`
	// Icon is a small image displayed beside the title and subtitle.
	Icon *ImageBlockElement `json:"icon,omitempty"`
	// Width controls the container width. Slack defaults to standard when unset.
	Width ContainerWidth `json:"width,omitempty"`
	// IsCollapsible enables the container's collapse control.
	IsCollapsible bool `json:"is_collapsible,omitempty"`
	// DefaultCollapsed starts the container collapsed. It only applies when
	// IsCollapsible is true.
	DefaultCollapsed bool `json:"default_collapsed,omitempty"`
	// HasHeaderDivider draws a border below the header. Slack only supports it on
	// non-collapsible containers.
	HasHeaderDivider bool `json:"has_header_divider,omitempty"`
	// ChildBlocks are the blocks rendered inside the container. Slack requires 1
	// to 10 blocks: actions, context, divider, file, header, image, input,
	// rich_text, section, table, and video blocks are supported.
	ChildBlocks Blocks `json:"child_blocks"`
}

// BlockType returns the type of the block.
func (s ContainerBlock) BlockType() MessageBlockType {
	return s.Type
}

// ID returns the ID of the block.
func (s ContainerBlock) ID() string {
	return s.BlockID
}

// Validate checks whether the block satisfies Slack's documented container
// constraints.
func (s ContainerBlock) Validate() error {
	if s.Type != MBTContainer {
		return fmt.Errorf("type must be %q", MBTContainer)
	}
	if s.Title == nil && s.RichTextTitle == nil {
		return fmt.Errorf("one of title or rich_text_title is required")
	}
	if s.Title != nil {
		if s.Title.Type != PlainTextType {
			return fmt.Errorf("title must be a plain_text object")
		}
		if runeLen(s.Title.Text) > 150 {
			return fmt.Errorf("title cannot be longer than 150 characters")
		}
	}
	if s.Subtitle != nil {
		if s.Subtitle.Type != PlainTextType && s.Subtitle.Type != MarkdownType {
			return fmt.Errorf("subtitle must be a plain_text or mrkdwn object")
		}
		if runeLen(s.Subtitle.Text) > 150 {
			return fmt.Errorf("subtitle cannot be longer than 150 characters")
		}
	}
	switch s.Width {
	case "", ContainerWidthNarrow, ContainerWidthStandard, ContainerWidthWide, ContainerWidthFull:
	default:
		return fmt.Errorf("width must be one of narrow, standard, wide, or full")
	}
	if s.Icon != nil {
		if runeLen(s.Icon.AltText) > 2000 {
			return fmt.Errorf("icon alt_text cannot be longer than 2000 characters")
		}
		if s.Icon.ImageURL != nil && runeLen(*s.Icon.ImageURL) > 3000 {
			return fmt.Errorf("icon image_url cannot be longer than 3000 characters")
		}
	}
	if s.HasHeaderDivider && s.IsCollapsible {
		return fmt.Errorf("has_header_divider is only supported on non-collapsible containers")
	}
	if s.DefaultCollapsed && !s.IsCollapsible {
		return fmt.Errorf("default_collapsed requires is_collapsible to be true")
	}
	if n := len(s.ChildBlocks.BlockSet); n < 1 {
		return fmt.Errorf("child_blocks must have at least 1 block")
	} else if n > containerMaxChildBlocks {
		return fmt.Errorf("child_blocks cannot have more than %d blocks", containerMaxChildBlocks)
	}
	return nil
}

// NewContainerBlock returns a new container block wrapping the given child
// blocks. Use the With* methods to set the title, subtitle, and other optional
// fields.
func NewContainerBlock(childBlocks ...Block) *ContainerBlock {
	return &ContainerBlock{
		Type:        MBTContainer,
		ChildBlocks: Blocks{BlockSet: childBlocks},
	}
}

// WithBlockID sets the block ID for the ContainerBlock.
func (s *ContainerBlock) WithBlockID(blockID string) *ContainerBlock {
	s.BlockID = blockID
	return s
}

// WithTitle sets the plain_text title for the ContainerBlock.
func (s *ContainerBlock) WithTitle(title *TextBlockObject) *ContainerBlock {
	s.Title = title
	return s
}

// WithRichTextTitle sets the rich_text title for the ContainerBlock. It takes
// precedence over a plain_text title set with WithTitle.
func (s *ContainerBlock) WithRichTextTitle(title *RichTextBlock) *ContainerBlock {
	s.RichTextTitle = title
	return s
}

// WithSubtitle sets the subtitle for the ContainerBlock.
func (s *ContainerBlock) WithSubtitle(subtitle *TextBlockObject) *ContainerBlock {
	s.Subtitle = subtitle
	return s
}

// WithIcon sets the icon displayed beside the title for the ContainerBlock.
func (s *ContainerBlock) WithIcon(icon *ImageBlockElement) *ContainerBlock {
	s.Icon = icon
	return s
}

// WithWidth sets the rendered width of the ContainerBlock.
func (s *ContainerBlock) WithWidth(width ContainerWidth) *ContainerBlock {
	s.Width = width
	return s
}

// WithCollapsible marks the ContainerBlock collapsible and controls whether it
// starts collapsed.
func (s *ContainerBlock) WithCollapsible(collapsible, defaultCollapsed bool) *ContainerBlock {
	s.IsCollapsible = collapsible
	s.DefaultCollapsed = defaultCollapsed
	return s
}

// WithHeaderDivider draws a border below the container header.
func (s *ContainerBlock) WithHeaderDivider(hasHeaderDivider bool) *ContainerBlock {
	s.HasHeaderDivider = hasHeaderDivider
	return s
}

// AddChildBlock appends a block to the container's child blocks.
func (s *ContainerBlock) AddChildBlock(block Block) *ContainerBlock {
	s.ChildBlocks.BlockSet = append(s.ChildBlocks.BlockSet, block)
	return s
}
