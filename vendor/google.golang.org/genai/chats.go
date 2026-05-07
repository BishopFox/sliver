// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Chats client.

package genai

import (
	"context"
	"fmt"
	"io"
	"iter"
)

// Chats provides util functions for creating a new chat session.
// You don't need to initiate this struct. Create a client instance via NewClient, and
// then access Chats through client.Models field.
type Chats struct {
	apiClient *apiClient
}

// Chat represents a single chat session (multi-turn conversation) with the model.
//
//		client, _ := genai.NewClient(ctx, &genai.ClientConfig{})
//		chat, _ := client.Chats.Create(ctx, "gemini-2.5-flash", nil, nil)
//	  result, err = chat.SendMessage(ctx, genai.Part{Text: "What is 1 + 2?"})
type Chat struct {
	Models
	apiClient *apiClient
	model     string
	config    *GenerateContentConfig
	// Comprehensive history is the full history of the chat, including turns of the invalid contents from the model and their associated inputs.
	comprehensiveHistory []*Content
	// Curated history is the set of valid turns that will be used in the subsequent send requests.
	curatedHistory []*Content
}

func validateContent(content *Content) bool {
	if content == nil || len(content.Parts) == 0 {
		return false
	}

	for _, part := range content.Parts {
		if part == nil {
			return false
		}
		if part.Text != "" {
			continue
		}
		if part.InlineData == nil &&
			part.FileData == nil &&
			part.FunctionCall == nil &&
			part.FunctionResponse == nil &&
			part.ExecutableCode == nil &&
			part.CodeExecutionResult == nil {
			return false
		}
	}
	return true
}

func validateResponse(response *GenerateContentResponse) bool {
	if response == nil || len(response.Candidates) == 0 {
		return false
	}
	if response.Candidates[0].Content == nil {
		return false
	}
	return validateContent(response.Candidates[0].Content)
}

func extractCuratedHistory(comprehensiveHistory []*Content) ([]*Content, error) {
	if len(comprehensiveHistory) == 0 {
		return []*Content{}, nil
	}

	curatedHistory := []*Content{}
	length := len(comprehensiveHistory)
	i := 0
	for i < length {
		currentContent := comprehensiveHistory[i]
		if currentContent.Role != RoleUser && currentContent.Role != RoleModel {
			return nil, fmt.Errorf("Role must be user or model, but got %s", currentContent.Role)
		}

		if currentContent.Role == RoleUser {
			curatedHistory = append(curatedHistory, currentContent)
			i++
		} else {
			var modelOutputs []*Content
			isValid := true
			for i < length && comprehensiveHistory[i].Role == RoleModel {
				modelOutput := comprehensiveHistory[i]
				modelOutputs = append(modelOutputs, modelOutput)
				if isValid && !validateContent(modelOutput) {
					isValid = false
				}
				i++
			}

			if isValid {
				curatedHistory = append(curatedHistory, modelOutputs...)
			} else {
				// Remove the corresponding user input
				if len(curatedHistory) > 0 && curatedHistory[len(curatedHistory)-1].Role == RoleUser {
					curatedHistory = curatedHistory[:len(curatedHistory)-1]
				}
			}
		}
	}
	return curatedHistory, nil
}

// Create initializes a new chat session.
func (c *Chats) Create(ctx context.Context, model string, config *GenerateContentConfig, history []*Content) (*Chat, error) {
	compHistory := history
	if compHistory == nil {
		compHistory = []*Content{}
	}
	curatedHistory, err := extractCuratedHistory(compHistory)
	if err != nil {
		return nil, err
	}
	chat := &Chat{
		apiClient:            c.apiClient,
		model:                model,
		config:               config,
		comprehensiveHistory: compHistory,
		curatedHistory:       curatedHistory,
	}
	chat.Models.apiClient = c.apiClient
	return chat, nil
}

func (c *Chat) recordHistory(ctx context.Context, inputContent *Content, outputContents []*Content, isValid bool) {
	c.comprehensiveHistory = append(c.comprehensiveHistory, inputContent)
	if len(outputContents) == 0 {
		c.comprehensiveHistory = append(c.comprehensiveHistory, &Content{Role: RoleModel, Parts: []*Part{}})
	} else {
		c.comprehensiveHistory = append(c.comprehensiveHistory, outputContents...)
	}

	if isValid {
		c.curatedHistory = append(c.curatedHistory, inputContent)
		if len(outputContents) == 0 {
			c.curatedHistory = append(c.curatedHistory, &Content{Role: RoleModel, Parts: []*Part{}})
		} else {
			c.curatedHistory = append(c.curatedHistory, outputContents...)
		}
	}
}

// History returns the chat history. Returns the curated history if
// curated is true, otherwise returns the comprehensive history.
func (c *Chat) History(curated bool) []*Content {
	if curated {
		return c.curatedHistory
	}
	return c.comprehensiveHistory
}

// SendMessage is a wrapper around Send.
func (c *Chat) SendMessage(ctx context.Context, parts ...Part) (*GenerateContentResponse, error) {
	// Transform Parts to single Content
	p := make([]*Part, len(parts))
	for i, part := range parts {
		p[i] = &part
	}
	return c.Send(ctx, p...)
}

// Send function sends the conversation history with the additional user's message and returns the model's response.
func (c *Chat) Send(ctx context.Context, parts ...*Part) (*GenerateContentResponse, error) {
	inputContent := &Content{Parts: parts, Role: RoleUser}

	// Combine history with input content to send to model
	contents := append(c.curatedHistory, inputContent)

	// Generate Content
	modelOutput, err := c.GenerateContent(ctx, c.model, contents, c.config)
	if err != nil {
		return nil, err
	}

	// Record history. By default, use the first candidate for history.
	var outputContents []*Content
	if len(modelOutput.Candidates) > 0 && modelOutput.Candidates[0].Content != nil {
		outputContents = append(outputContents, modelOutput.Candidates[0].Content)
	}
	c.recordHistory(ctx, inputContent, outputContents, validateResponse(modelOutput))

	return modelOutput, err
}

// SendMessageStream is a wrapper around SendStream.
func (c *Chat) SendMessageStream(ctx context.Context, parts ...Part) iter.Seq2[*GenerateContentResponse, error] {
	// Transform Parts to single Content
	p := make([]*Part, len(parts))
	for i, part := range parts {
		p[i] = &part
	}
	return c.SendStream(ctx, p...)
}

// SendStream function sends the conversation history with the additional user's message and returns the model's response.
func (c *Chat) SendStream(ctx context.Context, parts ...*Part) iter.Seq2[*GenerateContentResponse, error] {
	inputContent := &Content{Parts: parts, Role: RoleUser}

	// Combine history with input content to send to model
	contents := append(c.curatedHistory, inputContent)

	// Generate Content
	response := c.GenerateContentStream(ctx, c.model, contents, c.config)

	// Return a new iterator that will yield the responses and record history with merged response.
	return func(yield func(*GenerateContentResponse, error) bool) {
		var outputContents []*Content
		isValid := true
		finishReason := FinishReasonUnspecified
		for chunk, err := range response {
			if err == io.EOF {
				break
			}
			if err != nil {
				yield(nil, err)
				return
			}
			if !validateResponse(chunk) {
				isValid = false
			}
			if len(chunk.Candidates) > 0 {
				if chunk.Candidates[0].Content != nil {
					outputContents = append(outputContents, chunk.Candidates[0].Content)
				}
				if chunk.Candidates[0].FinishReason != FinishReasonUnspecified {
					finishReason = chunk.Candidates[0].FinishReason
				}
			}
			if !yield(chunk, nil) {
				return
			}
		}
		// Record history. By default, use the first candidate for history.
		finalIsValid := isValid && finishReason != FinishReasonUnspecified
		c.recordHistory(ctx, inputContent, outputContents, finalIsValid)
	}
}
