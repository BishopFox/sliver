package fantasy

import (
	"context"
)

// Provider represents a provider of language models.
type Provider interface {
	Name() string
	LanguageModel(ctx context.Context, modelID string) (LanguageModel, error)
}
