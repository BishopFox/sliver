package slack

// TaskCardStatus defines the status of a task card block.
type TaskCardStatus string

const (
	TaskCardStatusPending    TaskCardStatus = "pending"
	TaskCardStatusInProgress TaskCardStatus = "in_progress"
	TaskCardStatusComplete   TaskCardStatus = "complete"
	TaskCardStatusError      TaskCardStatus = "error"
)

// TaskCardSource represents a URL reference in a task card block.
type TaskCardSource struct {
	Type string `json:"type"`
	URL  string `json:"url"`
	Text string `json:"text"`
}

// NewTaskCardSource creates a new TaskCardSource with type "url".
func NewTaskCardSource(url, text string) TaskCardSource {
	return TaskCardSource{
		Type: "url",
		URL:  url,
		Text: text,
	}
}

// TaskCardBlock defines a block of type task_card used by AI agents
// to display thinking steps and task execution.
//
// More Information: https://docs.slack.dev/reference/block-kit/blocks/task-card-block/
type TaskCardBlock struct {
	Type    MessageBlockType `json:"type"`
	BlockID string           `json:"block_id,omitempty"`
	TaskID  string           `json:"task_id"`
	Title   string           `json:"title"`
	Status  TaskCardStatus   `json:"status,omitempty"`
	Details *RichTextBlock   `json:"details,omitempty"`
	Output  *RichTextBlock   `json:"output,omitempty"`
	Sources []TaskCardSource `json:"sources,omitempty"`
}

// BlockType returns the type of the block
func (s TaskCardBlock) BlockType() MessageBlockType {
	return s.Type
}

// ID returns the ID of the block
func (s TaskCardBlock) ID() string {
	return s.BlockID
}

// TaskCardBlockOption allows configuration of options for a new task card block
type TaskCardBlockOption func(*TaskCardBlock)

// TaskCardBlockOptionBlockID sets the block ID for the task card block
func TaskCardBlockOptionBlockID(blockID string) TaskCardBlockOption {
	return func(block *TaskCardBlock) {
		block.BlockID = blockID
	}
}

// NewTaskCardBlock returns a new instance of a task card block
func NewTaskCardBlock(taskID, title string, options ...TaskCardBlockOption) *TaskCardBlock {
	block := TaskCardBlock{
		Type:   MBTTaskCard,
		TaskID: taskID,
		Title:  title,
	}

	for _, option := range options {
		if option != nil {
			option(&block)
		}
	}

	return &block
}

// WithStatus sets the status for the TaskCardBlock
func (s *TaskCardBlock) WithStatus(status TaskCardStatus) *TaskCardBlock {
	s.Status = status
	return s
}

// WithDetails sets the details rich text block for the TaskCardBlock
func (s *TaskCardBlock) WithDetails(details *RichTextBlock) *TaskCardBlock {
	s.Details = details
	return s
}

// WithOutput sets the output rich text block for the TaskCardBlock
func (s *TaskCardBlock) WithOutput(output *RichTextBlock) *TaskCardBlock {
	s.Output = output
	return s
}

// WithSources sets the sources for the TaskCardBlock
func (s *TaskCardBlock) WithSources(sources ...TaskCardSource) *TaskCardBlock {
	s.Sources = sources
	return s
}
