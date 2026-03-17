// Package compat provides common types used across the application.
package compat

import tea "charm.land/bubbletea/v2"

// Model is a bubbletea v1 [tea.Model].
type Model interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (Model, tea.Cmd)
	View() string
}

// ViewHook is a function that modifies a [tea.View].
type ViewHook = func(tea.View) tea.View

// ViewModel wraps a [Model] and [ViewHook].
type ViewModel struct {
	Model
	ViewHook ViewHook
}

// Update implements [tea.Model].
func (w ViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m, cmd := w.Model.Update(msg)
	return ViewModel{
		Model:    m,
		ViewHook: w.ViewHook,
	}, cmd
}

// View implements [tea.Model].
func (w ViewModel) View() tea.View {
	var view tea.View
	if w.ViewHook != nil {
		view = w.ViewHook(view)
	}
	view.SetContent(w.Model.View())
	return view
}
