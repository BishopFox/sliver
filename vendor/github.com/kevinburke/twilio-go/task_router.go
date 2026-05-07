package twilio

// WorkspaceService lets you interact with a TaskRouter Workspace.
type WorkspaceService struct {
	Activities *ActivityService
	Queues     *TaskQueueService
	Workflows  *WorkflowService
	Workers    *WorkerService
}
