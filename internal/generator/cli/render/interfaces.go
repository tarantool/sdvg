package render

import (
	"context"
)

// Renderer interface implementation should render interactive menu.
//
//go:generate go run github.com/vektra/mockery/v2@v2.51.1 --name=Renderer --output=mock --outpkg=mock
type Renderer interface {
	// Logo should display application logo.
	Logo()
	// SelectionMenu should display menu for selection.
	SelectionMenu(ctx context.Context, title string, items []string) (string, error)
	// InputMenu should display menu for input.
	InputMenu(ctx context.Context, title string, validateFunc func(string) error) (string, error)
	// TextMenu should display menu for text input.
	TextMenu(ctx context.Context, title string) (string, error)
	// WithSpinner should display spinner.
	WithSpinner(title string, fn func())
	// IsTerminal should return true if renderer is connected to a terminal.
	IsTerminal() bool
	// ReadLine should read input from input stream.
	ReadLine() (string, error)
	// Read should read from input stream.
	Read(p []byte) (int, error)
}
