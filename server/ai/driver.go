package ai

import (
	"context"
	"fmt"
)

type completionRequest struct {
	SystemPrompt string
	Messages     []providerMessage
}

type sdkDriver interface {
	Name() string
	Supports(*RuntimeConfig) bool
	CompleteConversation(context.Context, *RuntimeConfig, *completionRequest) (*Completion, error)
}

var completionDrivers = []sdkDriver{
	newOpenAIDriver(),
}

func selectCompletionDriver(runtime *RuntimeConfig) (sdkDriver, error) {
	if runtime == nil {
		return nil, fmt.Errorf("AI runtime config is required")
	}

	for _, driver := range completionDrivers {
		if driver.Supports(runtime) {
			return driver, nil
		}
	}

	return nil, fmt.Errorf(
		"server AI provider %q does not have an available SDK driver; use openai, openai-compat, or openrouter, or add a driver in server/ai",
		runtime.Provider,
	)
}
