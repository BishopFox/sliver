# Gemini API Coding Guidelines (Go)

You are a Gemini API coding expert. Help me with writing code using the Gemini
API calling the official libraries and SDKs.

Please follow the following guidelines when generating code.

**Official Documentation:** [ai.google.dev/gemini-api/docs](https://ai.google.dev/gemini-api/docs)

## Golden Rule: Use the Correct and Current SDK

Always use the **Google GenAI SDK** (`google.golang.org/genai`), which is the unified
standard library for all Gemini API requests (AI Studio/Gemini Developer API
and Vertex AI) as of 2025. Do not use legacy libraries and SDKs.

-   **Library Name:** Google GenAI SDK
-   **Go Module:** `google.golang.org/genai`
-   **Legacy Library**: (`github.com/google/generative-ai-go`) is deprecated.

**Installation:**

```bash
go get google.golang.org/genai
```

**APIs and Usage:**

-   **Incorrect:** `import "github.com/google/generative-ai-go/genai"` -> **Correct:** `import "google.golang.org/genai"`
-   **Incorrect:** `genai.NewClient(ctx, option.WithAPIKey(...))` -> **Correct:** `genai.NewClient(ctx, &genai.ClientConfig{APIKey: "...", Backend: genai.BackendGeminiAPI})`
-   **Incorrect:** `model := client.GenerativeModel(...)`
-   **Incorrect:** `model.GenerateContent(...)` -> **Correct:** `client.Models.GenerateContent(ctx, modelName, contents, config)`

## Initialization and API Key

The `google.golang.org/genai` library requires creating a client object for all API calls.

-   Use `genai.NewClient(ctx, config)` to create a client object.
-   Set `GEMINI_API_KEY` (or `GOOGLE_API_KEY`) environment variable, which will be picked up
    automatically if `APIKey` is omitted in `ClientConfig`.
-   For Vertex AI, set `Backend: genai.BackendVertexAI` and provide `Project` and `Location`.

```go
import (
	"context"

	"google.golang.org/genai"
)

ctx := context.Background()

// Best practice: implicitly use env variable
client, err := genai.NewClient(ctx, nil)

// Alternative: explicit key
// client, err := genai.NewClient(ctx, &genai.ClientConfig{
// 	APIKey:  "YOUR_API_KEY",
// 	Backend: genai.BackendGeminiAPI,
// })
```

## Models

-   By default, use the following models when using `google.golang.org/genai`:
    -   **General Text & Multimodal Tasks:** `gemini-3-flash-preview`
    -   **Coding and Complex Reasoning Tasks:** `gemini-3-pro-preview`
    -   **Low Latency & High Volume Tasks:** `gemini-2.5-flash-lite`
    -   **Fast Image Generation and Editing:** `gemini-2.5-flash-image`
    -   **High-Quality Image Generation and Editing:** `gemini-3-pro-image-preview`
    -   **High-Fidelity Video Generation:** `veo-3.0-generate-001` or `veo-3.1-generate-preview`
    -   **Fast Video Generation:** `veo-3.0-fast-generate-001` or `veo-3.1-fast-generate-preview`

-   It is also acceptable to use following models if explicitly requested by the
    user:
    -   **Gemini 2.0 Series**: `gemini-2.0-flash`, `gemini-2.0-flash-lite`
    -   **Gemini 2.5 Series**: `gemini-2.5-flash`, `gemini-2.5-pro`

-   Do not use the following prohibited models:
    -   **Prohibited:** `gemini-1.5-flash`
    -   **Prohibited:** `gemini-1.5-pro`
    -   **Prohibited:** `gemini-pro`

## Basic Inference (Text Generation)

Calls are stateless using the `client.Models` accessor. Use `genai.Text("...")` for simple prompts.

```go
result, err := client.Models.GenerateContent(ctx, "gemini-3-flash-preview", genai.Text("Why is the sky blue?"), nil)
if err != nil {
	log.Fatal(err)
}

fmt.Println(result.Text())
```

## Multimodal Inputs

Pass images as bytes or file URIs.

### Using Bytes

```go
data, _ := os.ReadFile("image.jpg")
parts := []*genai.Part{
	{Text: "Describe this image."},
	{InlineData: &genai.Blob{Data: data, MIMEType: "image/jpeg"}},
}
contents := []*genai.Content{{Parts: parts}}

result, err := client.Models.GenerateContent(ctx, "gemini-3-flash-preview", contents, nil)
```

### File API (For Large Files)

For video files or long audio, upload to the File API first.

```go
// Upload
myFile, err := client.Files.UploadFromPath(ctx, "video.mp4", nil)

// Generate
parts := []*genai.Part{
	{Text: "What happens in this video?"},
	{FileData: &genai.FileData{FileURI: myFile.URI, MIMEType: myFile.MIMEType}},
}
contents := []*genai.Content{{Parts: parts}}
result, err := client.Models.GenerateContent(ctx, "gemini-3-flash-preview", contents, nil)

// Delete
err = client.Files.Delete(ctx, myFile.Name, nil)
```

## Advanced Capabilities

### Thinking (Reasoning)

Gemini 2.5 and 3 series models support explicit "thinking" for complex logic.

#### Gemini 3

Thinking is on by default for `gemini-3-pro-preview` and `gemini-3-flash-preview`.

```go
config := &genai.GenerateContentConfig{
	ThinkingConfig: &genai.ThinkingConfig{
		ThinkingLevel: genai.ThinkingLevelHigh,
	},
}
result, err := client.Models.GenerateContent(ctx, "gemini-3-pro-preview", genai.Text("What is AI?"), config)

for _, part := range result.Candidates[0].Content.Parts {
	if part.Thought {
		fmt.Printf("Thought: %s\n", part.Text)
	} else {
		fmt.Printf("Response: %s\n", part.Text)
	}
}
```

#### Gemini 2.5

Thinking is on by default. Setting `ThinkingBudget` to zero turns it off.

```go
config := &genai.GenerateContentConfig{
	ThinkingConfig: &genai.ThinkingConfig{
		ThinkingBudget: genai.Ptr[int32](0),
	},
}
```

### System Instructions

```go
config := &genai.GenerateContentConfig{
	SystemInstruction: &genai.Content{
		Parts: []*genai.Part{{Text: "You are a pirate"}},
	},
}
result, err := client.Models.GenerateContent(ctx, "gemini-3-flash-preview", genai.Text("Explain quantum physics."), config)
```

### Hyperparameters

Use `genai.Ptr` for optional fields in `GenerateContentConfig`.

```go
config := &genai.GenerateContentConfig{
	Temperature:     genai.Ptr[float32](0.5),
	MaxOutputTokens: 1024,
}
```

### Streaming

```go
for result, err := range client.Models.GenerateContentStream(ctx, "gemini-3-flash-preview", genai.Text("Write a long story."), nil) {
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(result.Text())
}
```

### Chat

For multi-turn conversations, use the `Chats` service.

```go
chat, err := client.Chats.Create(ctx, "gemini-3-flash-preview", nil, nil)

result, err := chat.SendMessage(ctx, genai.Part{Text: "I have a cat named Whiskers."})
fmt.Println(result.Text())

result, err = chat.SendMessage(ctx, genai.Part{Text: "What is my pet's name?"})
fmt.Println(result.Text())
```

### Structured Outputs (JSON Schema)

Enforce a specific JSON schema using `ResponseSchema` or `ResponseJsonSchema`.

```go
recipeSchema := &genai.Schema{
	Type: genai.TypeObject,
	Properties: map[string]*genai.Schema{
		"name":        {Type: genai.TypeString},
		"ingredients": {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
	},
	Required: []string{"name", "ingredients"},
}

config := &genai.GenerateContentConfig{
	ResponseMIMEType: "application/json",
	ResponseSchema:   recipeSchema,
}

result, err := client.Models.GenerateContent(ctx, "gemini-3-flash-preview", genai.Text("Recipe for cookies"), config)

// Parse manually
var recipe map[string]any
json.Unmarshal([]byte(result.Text()), &recipe)
```

### Function Calling

```go
var tools = []*genai.Tool{
	{
		FunctionDeclarations: []*genai.FunctionDeclaration{
			{
				Name:        "get_weather",
				Description: "Get weather for a city",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"city": {Type: genai.TypeString},
					},
					Required: []string{"city"},
				},
			},
		},
	},
}

config := &genai.GenerateContentConfig{Tools: tools}
result, err := client.Models.GenerateContent(ctx, "gemini-3-flash-preview", genai.Text("Weather in London?"), config)

if calls := result.FunctionCalls(); len(calls) > 0 {
	// Execute function and send back response...
}
```

### Grounding (Google Search)

```go
config := &genai.GenerateContentConfig{
	Tools: []*genai.Tool{
		{GoogleSearchRetrieval: &genai.GoogleSearchRetrieval{}},
	},
}
result, err := client.Models.GenerateContent(ctx, "gemini-3-flash-preview", genai.Text("Latest news?"), config)
```

## Media Generation

### Generate Images

```go
result, err := client.Models.GenerateContent(ctx, "gemini-2.5-flash-image", genai.Text("A futuristic city"), nil)

for _, part := range result.Candidates[0].Content.Parts {
	if part.InlineData != nil {
		os.WriteFile("output.png", part.InlineData.Data, 0644)
	}
}
```

### Video Generation (Veo)

```go
operation, err := client.Models.GenerateVideos(ctx, "veo-3.0-fast-generate-001", "Kitten playing", nil, nil, nil)

// Poll for completion
for !operation.Done {
	time.Sleep(20 * time.Second)
	operation, err = client.Operations.Get(ctx, operation.Name, nil)
}

// Download result
for _, v := range operation.Response.GeneratedVideos {
	file, _ := client.Files.Download(ctx, v.Video.Name, nil)
	os.WriteFile("video.mp4", file, 0644)
}
```

## Content and Part Hierarchy

The `genai.Text("...")` helper is a shorthand for:

```go
contents := []*genai.Content{
	{
		Role: "user",
		Parts: []*genai.Part{
			{Text: "How does AI work?"},
		},
	},
}
```

## Useful Links

-   Documentation: [ai.google.dev/gemini-api/docs](https://ai.google.dev/gemini-api/docs)
-   API Keys: [ai.google.dev/gemini-api/docs/api-key](https://ai.google.dev/gemini-api/docs/api-key)
-   Models: [ai.google.dev/models](https://ai.google.dev/models)
