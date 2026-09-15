//go:build client

package opfor

import (
	"context"
	"fmt"
	"strings"

	"github.com/bishopfox/sliver/client/forms"
	opforengine "github.com/sliverarmory/opfor"
)

// promptUI is deliberately small and injectable. The production adapter uses
// Sliver's existing Bubble Tea/Huh forms; tests can exercise callback behavior
// without taking over the terminal.
type promptUI interface {
	Confirm(title string, defaultValue bool) (bool, error)
	Input(title, defaultValue string) (string, error)
}

type formsPromptUI struct{}

func (formsPromptUI) Confirm(title string, defaultValue bool) (bool, error) {
	value := defaultValue
	err := forms.Confirm(title, &value)
	return value, err
}

func (formsPromptUI) Input(title, defaultValue string) (string, error) {
	value := defaultValue
	err := forms.Input(title, &value)
	return value, err
}

func (manager *Manager) presentPrompt(
	ctx context.Context,
	presentation opforengine.AggressorPromptPresentation,
	responder opforengine.AggressorPromptResponder,
) error {
	if responder == nil {
		return errorsPromptResponderNil()
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	switch presentation.Kind {
	case opforengine.AggressorPromptConfirm:
		title := joinPromptTitle(presentation.Title, presentation.Text)
		answer, err := manager.confirmPrompt(title, false)
		if err != nil {
			return err
		}
		if !answer {
			return responder.Dismiss()
		}
		_, err = responder.Accept(ctx)
		return err

	case opforengine.AggressorPromptText,
		opforengine.AggressorPromptDirectoryOpen,
		opforengine.AggressorPromptFileOpen,
		opforengine.AggressorPromptFileSave:
		title := presentation.Text
		if title == "" {
			title = presentation.Title
		}
		if title == "" {
			title = presentation.Name
		}
		defaultValue := ""
		if presentation.HasDefault {
			defaultValue = presentation.Default.String()
		}
		answer, err := manager.inputPrompt(title, defaultValue)
		if err != nil {
			return err
		}
		_, err = responder.Accept(ctx, opforengine.String(answer))
		return err

	default:
		return fmt.Errorf("opfor: unsupported prompt kind %q", presentation.Kind)
	}
}

func (manager *Manager) confirmPrompt(title string, defaultValue bool) (bool, error) {
	manager.promptMu.Lock()
	defer manager.promptMu.Unlock()
	return manager.ui.Confirm(title, defaultValue)
}

func (manager *Manager) inputPrompt(title, defaultValue string) (string, error) {
	manager.promptMu.Lock()
	defer manager.promptMu.Unlock()
	return manager.ui.Input(title, defaultValue)
}

func joinPromptTitle(parts ...string) string {
	nonempty := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			nonempty = append(nonempty, part)
		}
	}
	return strings.Join(nonempty, ": ")
}

func errorsPromptResponderNil() error {
	return fmt.Errorf("opfor: prompt responder is nil")
}
