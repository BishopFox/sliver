package slack

// PlanBlock defines a block of type plan used by AI agents
// to group multiple task cards under a shared title.
//
// More Information: https://docs.slack.dev/reference/block-kit/blocks/plan-block/
type PlanBlock struct {
	Type    MessageBlockType `json:"type"`
	BlockID string           `json:"block_id,omitempty"`
	Title   string           `json:"title"`
	Tasks   []TaskCardBlock  `json:"tasks,omitempty"`
}

// BlockType returns the type of the block
func (s PlanBlock) BlockType() MessageBlockType {
	return s.Type
}

// ID returns the ID of the block
func (s PlanBlock) ID() string {
	return s.BlockID
}

// PlanBlockOption allows configuration of options for a new plan block
type PlanBlockOption func(*PlanBlock)

// PlanBlockOptionBlockID sets the block ID for the plan block
func PlanBlockOptionBlockID(blockID string) PlanBlockOption {
	return func(block *PlanBlock) {
		block.BlockID = blockID
	}
}

// NewPlanBlock returns a new instance of a plan block
func NewPlanBlock(title string, options ...PlanBlockOption) *PlanBlock {
	block := PlanBlock{
		Type:  MBTPlan,
		Title: title,
	}

	for _, option := range options {
		if option != nil {
			option(&block)
		}
	}

	return &block
}

// WithTasks sets the tasks for the PlanBlock
func (s *PlanBlock) WithTasks(tasks ...*TaskCardBlock) *PlanBlock {
	s.Tasks = make([]TaskCardBlock, len(tasks))
	for i, t := range tasks {
		s.Tasks[i] = *t
	}
	return s
}
