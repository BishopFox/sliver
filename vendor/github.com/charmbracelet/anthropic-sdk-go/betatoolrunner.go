package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"

	"github.com/charmbracelet/anthropic-sdk-go/option"
	"golang.org/x/sync/errgroup"
)

// BetaTool represents a tool that can be executed by the BetaToolRunner.
type BetaTool interface {
	// Name returns the tool's name
	Name() string
	// Description returns the tool's description
	Description() string
	// InputSchema returns the JSON schema for the tool's input
	InputSchema() BetaToolInputSchemaParam
	// Execute runs the tool with raw JSON input and returns the result
	Execute(ctx context.Context, input json.RawMessage) (BetaToolResultBlockParamContentUnion, error)
}

// BetaToolRunnerParams contains parameters for creating a BetaToolRunner or BetaToolRunnerStreaming.
type BetaToolRunnerParams struct {
	BetaMessageNewParams
	// MaxIterations limits the number of API calls. When set to 0 (the default),
	// there is no limit and the runner continues until the model stops using tools.
	MaxIterations int
}

// betaToolRunnerBase holds state and logic shared by BetaToolRunner and BetaToolRunnerStreaming.
type betaToolRunnerBase struct {
	messageService *BetaMessageService
	// Params contains the configuration for the tool runner.
	// This field is exported so users can modify parameters directly.
	Params         BetaToolRunnerParams
	toolMap        map[string]BetaTool
	iterationCount int
	lastMessage    *BetaMessage
	completed      bool
	opts           []option.RequestOption
	err            error
}

func newBetaToolRunnerBase(messageService *BetaMessageService, tools []BetaTool, params BetaToolRunnerParams, opts []option.RequestOption) betaToolRunnerBase {
	toolMap := make(map[string]BetaTool)
	apiTools := make([]BetaToolUnionParam, len(tools))

	for i, tool := range tools {
		toolMap[tool.Name()] = tool
		apiTools[i] = BetaToolUnionParam{
			OfTool: &BetaToolParam{
				Name:        tool.Name(),
				Description: String(tool.Description()),
				InputSchema: tool.InputSchema(),
			},
		}
	}

	// Add tools to the API params
	params.BetaMessageNewParams.Tools = apiTools
	params.Messages = append([]BetaMessageParam{}, params.Messages...)

	return betaToolRunnerBase{
		messageService: messageService,
		Params:         params,
		toolMap:        toolMap,
		opts:           opts,
	}
}

// LastMessage returns the most recent assistant message, or nil if no messages have been received yet.
func (b *betaToolRunnerBase) LastMessage() *BetaMessage {
	return b.lastMessage
}

// AppendMessages adds messages to the conversation history.
// This is a convenience method equivalent to:
//
//	runner.Params.Messages = append(runner.Params.Messages, messages...)
func (b *betaToolRunnerBase) AppendMessages(messages ...BetaMessageParam) {
	b.Params.Messages = append(b.Params.Messages, messages...)
}

// Messages returns a copy of the current conversation history.
// The returned slice can be safely modified without affecting the runner's state.
func (b *betaToolRunnerBase) Messages() []BetaMessageParam {
	result := make([]BetaMessageParam, len(b.Params.Messages))
	copy(result, b.Params.Messages)
	return result
}

// IterationCount returns the number of API calls made so far.
// This is incremented each time a turn makes an API call.
func (b *betaToolRunnerBase) IterationCount() int {
	return b.iterationCount
}

// IsCompleted returns true if the conversation has finished, either because
// the model stopped using tools or the maximum iteration limit was reached.
func (b *betaToolRunnerBase) IsCompleted() bool {
	return b.completed
}

// Err returns the last error that occurred during iteration, if any.
// This is useful when using All() or AllStreaming() to check for errors
// after the iteration completes.
func (b *betaToolRunnerBase) Err() error {
	return b.err
}

// executeTools processes any tool use blocks in the given message and returns a tool result message.
// Returns:
//   - (result, nil) if tools executed successfully
//   - (nil, nil) if no tools to execute
//   - (nil, ctx.Err()) if context was cancelled
func (b *betaToolRunnerBase) executeTools(ctx context.Context, message *BetaMessage) (*BetaMessageParam, error) {
	var toolUseBlocks []BetaToolUseBlock

	// Find all tool use blocks in the message
	for _, block := range message.Content {
		if block.Type == "tool_use" {
			toolUseBlocks = append(toolUseBlocks, block.AsToolUse())
		}
	}

	if len(toolUseBlocks) == 0 {
		return nil, nil
	}

	// Execute all tools in parallel using errgroup for proper cancellation handling
	results := make([]BetaContentBlockParamUnion, len(toolUseBlocks))

	g, gctx := errgroup.WithContext(ctx)
	for i, toolUse := range toolUseBlocks {
		g.Go(func() error {
			// Check for cancellation before executing tool
			select {
			case <-gctx.Done():
				return gctx.Err()
			default:
			}
			result := b.executeToolUse(gctx, toolUse)
			results[i] = BetaContentBlockParamUnion{OfToolResult: &result}
			return nil // tool errors become result content, not Go errors
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Create user message with tool results
	userMessage := NewBetaUserMessage(results...)
	return &userMessage, nil
}

func newBetaToolResultErrorBlockParam(toolUseID string, errorText string) BetaToolResultBlockParam {
	return NewBetaToolResultTextBlockParam(toolUseID, errorText, true)
}

// executeToolUse executes a single tool use block and returns the result.
func (b *betaToolRunnerBase) executeToolUse(ctx context.Context, toolUse BetaToolUseBlock) BetaToolResultBlockParam {
	tool, exists := b.toolMap[toolUse.Name]
	if !exists {
		return newBetaToolResultErrorBlockParam(
			toolUse.ID,
			fmt.Sprintf("Error: Tool '%s' not found", toolUse.Name),
		)
	}

	// Parse and execute the tool
	inputBytes, err := json.Marshal(toolUse.Input)
	if err != nil {
		return newBetaToolResultErrorBlockParam(
			toolUse.ID,
			fmt.Sprintf("Error: Failed to marshal tool input: %v", err),
		)
	}

	result, err := tool.Execute(ctx, inputBytes)
	if err != nil {
		return newBetaToolResultErrorBlockParam(
			toolUse.ID,
			fmt.Sprintf("Error: %v", err),
		)
	}

	return BetaToolResultBlockParam{
		ToolUseID: toolUse.ID,
		Content:   []BetaToolResultBlockParamContentUnion{result},
	}
}

// BetaToolRunner manages the automatic conversation loop between the assistant and tools
// using non-streaming API calls. It implements an iterator pattern for processing
// conversation turns.
//
// A BetaToolRunner is NOT safe for concurrent use. All methods must be called
// from a single goroutine. However, tool handlers ARE called concurrently
// when multiple tools are invoked in a single turn - ensure your handlers
// are thread-safe.
type BetaToolRunner struct {
	betaToolRunnerBase
}

// NewToolRunner creates a BetaToolRunner that automatically handles the loop between
// the model generating tool calls, executing those tool calls, and sending the
// results back to the model until a final answer is produced or the maximum
// number of iterations is reached.
func (r *BetaMessageService) NewToolRunner(tools []BetaTool, params BetaToolRunnerParams, opts ...option.RequestOption) *BetaToolRunner {
	return &BetaToolRunner{
		betaToolRunnerBase: newBetaToolRunnerBase(r, tools, params, opts),
	}
}

// NextMessage advances the conversation by one turn. It executes any pending tool calls
// from the previous message, then makes an API call to get the model's next response.
//
// Returns:
//   - (message, nil) on success with the assistant's response
//   - (nil, nil) when the conversation is complete (no more tool calls or max iterations reached)
//   - (nil, error) if an error occurred during tool execution or API call
func (r *BetaToolRunner) NextMessage(ctx context.Context) (*BetaMessage, error) {
	if r.completed {
		return nil, nil
	}

	// Check iteration limit
	if r.Params.MaxIterations > 0 && r.iterationCount >= r.Params.MaxIterations {
		r.completed = true
		return r.lastMessage, nil
	}

	// Execute any pending tool calls from the last message
	if r.lastMessage != nil {
		toolMessage, err := r.executeTools(ctx, r.lastMessage)
		if err != nil {
			r.err = err
			return nil, err
		}
		if toolMessage == nil {
			// No tools to execute, conversation is complete
			r.completed = true
			return r.lastMessage, nil
		}
		r.Params.Messages = append(r.Params.Messages, *toolMessage)
	}

	// Make API call
	r.iterationCount++
	messageParams := r.Params.BetaMessageNewParams
	messageParams.Messages = r.Params.Messages

	message, err := r.messageService.New(ctx, messageParams, r.opts...)
	if err != nil {
		r.err = err
		return nil, fmt.Errorf("failed to get next message: %w", err)
	}

	r.lastMessage = message
	r.Params.Messages = append(r.Params.Messages, message.ToParam())

	return message, nil
}

// RunToCompletion repeatedly calls NextMessage until the conversation is complete,
// either because the model stopped using tools or the maximum iteration limit was reached.
//
// Returns the final assistant message and any error that occurred.
func (r *BetaToolRunner) RunToCompletion(ctx context.Context) (*BetaMessage, error) {
	for {
		message, err := r.NextMessage(ctx)
		if err != nil {
			return nil, err
		}
		if message == nil {
			return r.lastMessage, nil
		}
	}
}

// All returns an iterator that yields all messages until the conversation completes.
// This is a convenience method for iterating over the entire conversation.
//
// Example usage:
//
//	for message, err := range runner.All(ctx) {
//	    if err != nil {
//	        return err
//	    }
//	    // process message
//	}
func (r *BetaToolRunner) All(ctx context.Context) iter.Seq2[*BetaMessage, error] {
	return func(yield func(*BetaMessage, error) bool) {
		for {
			message, err := r.NextMessage(ctx)
			r.err = err
			if message == nil {
				if err != nil {
					yield(nil, err)
				}
				return
			}
			if !yield(message, err) {
				return
			}
		}
	}
}

// BetaToolRunnerStreaming manages the automatic conversation loop between the assistant
// and tools using streaming API calls. It implements an iterator pattern for processing
// streaming events across conversation turns.
//
// A BetaToolRunnerStreaming is NOT safe for concurrent use. All methods must be called
// from a single goroutine. However, tool handlers ARE called concurrently
// when multiple tools are invoked in a single turn - ensure your handlers
// are thread-safe.
type BetaToolRunnerStreaming struct {
	betaToolRunnerBase
}

// NewToolRunnerStreaming creates a BetaToolRunnerStreaming that automatically handles
// the loop between the model generating tool calls, executing those tool calls, and
// sending the results back to the model using streaming API calls until a final answer
// is produced or the maximum number of iterations is reached.
func (r *BetaMessageService) NewToolRunnerStreaming(tools []BetaTool, params BetaToolRunnerParams, opts ...option.RequestOption) *BetaToolRunnerStreaming {
	return &BetaToolRunnerStreaming{
		betaToolRunnerBase: newBetaToolRunnerBase(r, tools, params, opts),
	}
}

// NextStreaming advances the conversation by one turn with streaming. It executes any
// pending tool calls from the previous message, then makes a streaming API call.
//
// Returns an iterator that yields streaming events as they arrive. The iterator should
// be fully consumed to ensure the message is properly accumulated for subsequent turns.
//
// If an error occurs, it will be yielded as the second value in the iterator pair.
// Check IsCompleted() after consuming the iterator to determine if the conversation
// has finished.
func (r *BetaToolRunnerStreaming) NextStreaming(ctx context.Context) iter.Seq2[BetaRawMessageStreamEventUnion, error] {
	return func(yield func(BetaRawMessageStreamEventUnion, error) bool) {
		if r.completed {
			return
		}

		// Check iteration limit
		if r.Params.MaxIterations > 0 && r.iterationCount >= r.Params.MaxIterations {
			r.completed = true
			return
		}

		// Execute any pending tool calls from the last message
		if r.lastMessage != nil {
			toolMessage, err := r.executeTools(ctx, r.lastMessage)
			if err != nil {
				r.err = err
				yield(BetaRawMessageStreamEventUnion{}, err)
				return
			}
			if toolMessage == nil {
				// No tools to execute, conversation is complete
				r.completed = true
				return
			}
			r.Params.Messages = append(r.Params.Messages, *toolMessage)
		}

		// Make streaming API call
		r.iterationCount++
		streamParams := r.Params.BetaMessageNewParams
		streamParams.Messages = r.Params.Messages

		stream := r.messageService.NewStreaming(ctx, streamParams, r.opts...)
		defer stream.Close()

		// We need to collect the final message from the stream for the next iteration
		finalMessage := &BetaMessage{}
		for stream.Next() {
			event := stream.Current()
			err := finalMessage.Accumulate(event)
			if err != nil {
				r.err = fmt.Errorf("failed to accumulate streaming event: %w", err)
				yield(BetaRawMessageStreamEventUnion{}, r.err)
				return
			}

			if !yield(event, nil) {
				return
			}
		}

		// Check for stream errors after the loop exits
		if stream.Err() != nil {
			r.err = stream.Err()
			yield(BetaRawMessageStreamEventUnion{}, r.err)
			return
		}

		r.lastMessage = finalMessage
		r.Params.Messages = append(r.Params.Messages, finalMessage.ToParam())
	}
}

// AllStreaming returns an iterator of iterators, where each inner iterator yields
// streaming events for a single turn of the conversation. The outer iterator continues
// until the conversation completes.
//
// Example usage:
//
//	for events, err := range runner.AllStreaming(ctx) {
//	    if err != nil {
//	        return err
//	    }
//	    for event, err := range events {
//	        if err != nil {
//	            return err
//	        }
//	        // process streaming event
//	    }
//	}
func (r *BetaToolRunnerStreaming) AllStreaming(ctx context.Context) iter.Seq2[iter.Seq2[BetaRawMessageStreamEventUnion, error], error] {
	return func(yield func(iter.Seq2[BetaRawMessageStreamEventUnion, error], error) bool) {
		for !r.completed {
			eventSeq := r.NextStreaming(ctx)
			if !yield(eventSeq, nil) {
				return
			}
		}
	}
}
