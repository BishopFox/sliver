# Tool Helpers

The SDK provides helper functions for defining tools and automatically running the conversation loop between Claude and your tools until Claude produces a final response.

## Defining Tools

Use the `toolrunner` package to create a `BetaTool` that combines the tool definition with its execution handler. There are three ways to create tools:

- `NewBetaToolFromJSONSchema` - Automatically generates schema from a struct with `jsonschema` tags (recommended)
- `NewBetaToolFromBytes` - Creates a tool from JSON schema bytes
- `NewBetaTool` - Creates a tool from an explicit `BetaToolInputSchemaParam`

The generic type parameter is automatically inferred from your handler function's signature, so you don't need to specify it explicitly.

### Automatic Schema Generation from Structs (Recommended)

The easiest approach is to use `NewBetaToolFromJSONSchema`, which automatically generates the schema from your struct using `jsonschema` tags:

```go
type GetWeatherInput struct {
	City  string `json:"city" jsonschema:"required,description=The city name"`
	Units string `json:"units,omitempty" jsonschema:"enum=celsius,enum=fahrenheit,description=Temperature units"`
}

weatherTool, err := toolrunner.NewBetaToolFromJSONSchema(
	"get_weather",
	"Get current weather for a city",
	func(ctx context.Context, input GetWeatherInput) (anthropic.BetaToolResultBlockParamContentUnion, error) {
		return anthropic.BetaToolResultBlockParamContentUnion{
			OfText: &anthropic.BetaTextBlockParam{
				Text: fmt.Sprintf("Weather in %s: 72°F, sunny", input.City),
			},
		}, nil
	},
)
```

### Using JSON Bytes

You can provide the schema as JSON bytes using `NewBetaToolFromBytes`:

```go
type GetWeatherInput struct {
	City string `json:"city"`
}

weatherTool, err := toolrunner.NewBetaToolFromBytes(
	"get_weather",
	"Get current weather for a city",
	[]byte(`{
		"type": "object",
		"properties": {
			"city": {"type": "string", "description": "The city name"}
		},
		"required": ["city"]
	}`),
	func(ctx context.Context, input GetWeatherInput) (anthropic.BetaToolResultBlockParamContentUnion, error) {
		// Your handler here
	},
)
```

### Using an Explicit Schema

For full control, use `NewBetaTool` with a `BetaToolInputSchemaParam` directly:

```go
weatherTool := toolrunner.NewBetaTool(
	"get_weather",
	"Get current weather for a city",
	anthropic.BetaToolInputSchemaParam{
		Properties: map[string]any{
			"city": map[string]any{
				"type":        "string",
				"description": "The city name",
			},
		},
	},
	handler,
)
```

### Raw JSON Input

If you prefer to handle JSON parsing yourself, use `json.RawMessage` or `[]byte` as the input type:

```go
rawTool, err := toolrunner.NewBetaToolFromBytes(
	"process_data",
	"Process raw JSON data",
	schemaBytes,
	func(ctx context.Context, input json.RawMessage) (anthropic.BetaToolResultBlockParamContentUnion, error) {
		// Parse the JSON yourself
		var data map[string]any
		json.Unmarshal(input, &data)
		// ...
	},
)

## Tool Runner

The `BetaToolRunner` automatically handles the conversation loop between Claude and your tools. On each iteration, it:

1. Sends the current messages to Claude
2. If Claude responds with tool calls, executes them in parallel
3. Adds the tool results to the conversation
4. Repeats until Claude produces a final response (no tool calls)

### Basic Usage

```go
tools := []anthropic.BetaTool{weatherTool}

runner := client.Beta.Messages.NewToolRunner(tools, anthropic.BetaToolRunnerParams{
	BetaMessageNewParams: anthropic.BetaMessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_20250514,
		MaxTokens: 1024,
		Messages: []anthropic.BetaMessageParam{
			anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock("What's the weather in Tokyo?")),
		},
	},
})

// Run the entire conversation to completion
message, err := runner.RunToCompletion(context.Background())
```

### Iterating Over Messages

Use `All()` to iterate over each message in the conversation:

```go
for message, err := range runner.All(ctx) {
	if err != nil {
		log.Fatal(err)
	}
	for _, block := range message.Content {
		switch b := block.AsAny().(type) {
		case anthropic.BetaTextBlock:
			fmt.Println("[assistant]:", b.Text)
		case anthropic.BetaToolUseBlock:
			fmt.Printf("[tool call]: %s(%v)\n", b.Name, b.Input)
		}
	}
}
```

### Step-by-Step Iteration

For more control, use `NextMessage()` to advance one turn at a time:

```go
for {
	message, err := runner.NextMessage(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if message == nil {
		break // Conversation complete
	}
	// Process the message...
}
```

### Streaming

Use `BetaToolRunnerStreaming` via `NewToolRunnerStreaming()` for streaming responses:

```go
runner := client.Beta.Messages.NewToolRunnerStreaming(tools, anthropic.BetaToolRunnerParams{
	BetaMessageNewParams: anthropic.BetaMessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_20250514,
		MaxTokens: 1024,
		Messages: []anthropic.BetaMessageParam{
			anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock("What's the weather in Tokyo?")),
		},
	},
})

for eventsIterator := range runner.AllStreaming(ctx) {
	for event, err := range eventsIterator {
		if err != nil {
			log.Fatal(err)
		}
		switch e := event.AsAny().(type) {
		case anthropic.BetaRawContentBlockDeltaEvent:
			switch delta := e.Delta.AsAny().(type) {
			case anthropic.BetaTextDelta:
				fmt.Print(delta.Text)
			}
		}
	}
}
```

Or use `NextStreaming()` for step-by-step streaming:

```go
for !runner.IsCompleted() {
	for event, err := range runner.NextStreaming(ctx) {
		// Handle streaming events...
	}
}
```

## Configuration

### Max Iterations

Limit the number of API calls to prevent runaway loops. When set to 0 (the default), there is no limit and the runner continues until the model stops using tools:

```go
runner := client.Beta.Messages.NewToolRunner(tools, anthropic.BetaToolRunnerParams{
	// ...
	MaxIterations: 10, // Stop after 10 API calls (0 = no limit)
})
```

### Modifying Parameters Mid-Conversation

The `Params` field is exported, so you can modify parameters directly:

```go
// Update maximum tokens
runner.Params.MaxTokens = 2048

// Update maximum iterations
runner.Params.MaxIterations = 10

// Update system prompt
runner.Params.System = []anthropic.BetaTextBlockParam{
	{Text: "You are a helpful assistant."},
}

// Add messages to the conversation (direct field access)
runner.Params.Messages = append(runner.Params.Messages, anthropic.NewBetaUserMessage(
	anthropic.NewBetaTextBlock("Now check the weather in London too"),
))

// Or use the convenience method
runner.AppendMessages(anthropic.NewBetaUserMessage(
	anthropic.NewBetaTextBlock("Now check the weather in London too"),
))
```

### Inspecting State

```go
// Get most recent assistant message
lastMsg := runner.LastMessage()

// Get full conversation history (returns a copy)
messages := runner.Messages()

// Check iteration count
count := runner.IterationCount()

// Check if completed
if runner.IsCompleted() {
	// ...
}
```

## Error Handling

Tool execution errors are automatically converted to error results and sent back to Claude, allowing it to recover or try a different approach:

```go
func handler(ctx context.Context, input MyInput) (anthropic.BetaToolResultBlockParamContentUnion, error) {
	if input.City == "" {
		return anthropic.BetaToolResultBlockParamContentUnion{}, errors.New("city is required")
	}
	// ...
}
```

The error message will be sent to Claude as a tool result with `is_error: true`.

## Parallel Tool Execution

When Claude requests multiple tool calls in a single message, they are executed in parallel using an `errgroup`. This provides:

- Concurrent execution for better performance
- Proper context cancellation handling
- Results returned in the correct order

## Examples

See the [examples](./examples) directory for complete working examples:

- [examples/tool-runner](./examples/tool-runner) - Basic tool runner usage
- [examples/tool-runner-streaming](./examples/tool-runner-streaming) - Streaming with tool runner
