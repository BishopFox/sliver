package server

import (
	"context"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// TaskMetrics contains metrics about task execution.
// This struct is passed to observability hooks to enable monitoring and analysis.
type TaskMetrics struct {
	TaskID        string         // Unique identifier for the task
	ToolName      string         // Name of the tool that created the task
	Status        mcp.TaskStatus // Current status of the task
	StatusMessage string         // Optional status message
	CreatedAt     time.Time      // When the task was created
	CompletedAt   *time.Time     // When the task completed (nil if not completed)
	Duration      time.Duration  // How long the task took (0 if not completed)
	SessionID     string         // Session that owns this task
	Error         error          // Error if task failed (nil otherwise)
}

// OnTaskCreatedHookFunc is called when a new task is created.
// Use this to track task creation metrics, initialize monitoring, or log task starts.
type OnTaskCreatedHookFunc func(ctx context.Context, metrics TaskMetrics)

// OnTaskCompletedHookFunc is called when a task completes successfully.
// Use this to track completion metrics, record duration, or trigger follow-up actions.
type OnTaskCompletedHookFunc func(ctx context.Context, metrics TaskMetrics)

// OnTaskFailedHookFunc is called when a task fails with an error.
// Use this to track failure metrics, alert on errors, or log failure details.
type OnTaskFailedHookFunc func(ctx context.Context, metrics TaskMetrics)

// OnTaskCancelledHookFunc is called when a task is cancelled.
// Use this to track cancellation metrics or clean up resources.
type OnTaskCancelledHookFunc func(ctx context.Context, metrics TaskMetrics)

// OnTaskStatusChangedHookFunc is called whenever a task's status changes.
// This is a catch-all hook that fires for all status transitions.
// Use this for general monitoring or when you need to track all state changes.
type OnTaskStatusChangedHookFunc func(ctx context.Context, metrics TaskMetrics)

// TaskHooks contains lifecycle hooks for task execution.
// These hooks enable observability and monitoring of task-augmented tools.
type TaskHooks struct {
	OnTaskCreated       []OnTaskCreatedHookFunc
	OnTaskCompleted     []OnTaskCompletedHookFunc
	OnTaskFailed        []OnTaskFailedHookFunc
	OnTaskCancelled     []OnTaskCancelledHookFunc
	OnTaskStatusChanged []OnTaskStatusChangedHookFunc
}

// AddOnTaskCreated registers a hook for task creation events.
func (h *TaskHooks) AddOnTaskCreated(hook OnTaskCreatedHookFunc) {
	h.OnTaskCreated = append(h.OnTaskCreated, hook)
}

// AddOnTaskCompleted registers a hook for task completion events.
func (h *TaskHooks) AddOnTaskCompleted(hook OnTaskCompletedHookFunc) {
	h.OnTaskCompleted = append(h.OnTaskCompleted, hook)
}

// AddOnTaskFailed registers a hook for task failure events.
func (h *TaskHooks) AddOnTaskFailed(hook OnTaskFailedHookFunc) {
	h.OnTaskFailed = append(h.OnTaskFailed, hook)
}

// AddOnTaskCancelled registers a hook for task cancellation events.
func (h *TaskHooks) AddOnTaskCancelled(hook OnTaskCancelledHookFunc) {
	h.OnTaskCancelled = append(h.OnTaskCancelled, hook)
}

// AddOnTaskStatusChanged registers a hook for all task status changes.
func (h *TaskHooks) AddOnTaskStatusChanged(hook OnTaskStatusChangedHookFunc) {
	h.OnTaskStatusChanged = append(h.OnTaskStatusChanged, hook)
}

// taskCreated calls all registered task creation hooks.
func (h *TaskHooks) taskCreated(ctx context.Context, metrics TaskMetrics) {
	if h == nil {
		return
	}
	for _, hook := range h.OnTaskCreated {
		hook(ctx, metrics)
	}
	// Also call status changed hook
	h.taskStatusChanged(ctx, metrics)
}

// taskCompleted calls all registered task completion hooks.
func (h *TaskHooks) taskCompleted(ctx context.Context, metrics TaskMetrics) {
	if h == nil {
		return
	}
	for _, hook := range h.OnTaskCompleted {
		hook(ctx, metrics)
	}
	// Also call status changed hook
	h.taskStatusChanged(ctx, metrics)
}

// taskFailed calls all registered task failure hooks.
func (h *TaskHooks) taskFailed(ctx context.Context, metrics TaskMetrics) {
	if h == nil {
		return
	}
	for _, hook := range h.OnTaskFailed {
		hook(ctx, metrics)
	}
	// Also call status changed hook
	h.taskStatusChanged(ctx, metrics)
}

// taskCancelled calls all registered task cancellation hooks.
func (h *TaskHooks) taskCancelled(ctx context.Context, metrics TaskMetrics) {
	if h == nil {
		return
	}
	for _, hook := range h.OnTaskCancelled {
		hook(ctx, metrics)
	}
	// Also call status changed hook
	h.taskStatusChanged(ctx, metrics)
}

// taskStatusChanged calls all registered status change hooks.
func (h *TaskHooks) taskStatusChanged(ctx context.Context, metrics TaskMetrics) {
	if h == nil {
		return
	}
	for _, hook := range h.OnTaskStatusChanged {
		hook(ctx, metrics)
	}
}
