package utils

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/tarantool/sdvg/internal/generator/cli/render"
)

// ValidateFileFormat returns an error if the file format is not supported.
func ValidateFileFormat(formats ...string) func(string) error {
	return func(filePath string) error {
		if len(formats) == 0 {
			return nil
		}

		ext := filepath.Ext(filePath)
		if !slices.Contains(formats, ext) {
			return errors.Errorf("invalid file extension, supported: %v", formats)
		}

		return nil
	}
}

// ValidateEmptyString returns an error if the string is empty.
func ValidateEmptyString() func(string) error {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return errors.New("string should not be empty")
		}

		return nil
	}
}

// GetPercentage calculates what percentage 'currentValue' is of 'total'.
func GetPercentage(total, currentValue uint64) uint64 {
	if total == 0 {
		return 0
	}

	return currentValue * 100 / total
}

// Map maps slice of elements with type T to slice with type V using function fn.
func Map[T, V any](ts []T, fn func(T) V) []V {
	result := make([]V, len(ts))
	for i, t := range ts {
		result[i] = fn(t)
	}

	return result
}

// ChooseCommand runs interactive menu with selection of available commands.
func ChooseCommand(cmd *cobra.Command, args []string, renderer render.Renderer) error {
	const backNavigation = "back"

	command := cmd

	for len(command.Commands()) > 0 {
		commandNames := Map[*cobra.Command, string](
			command.Commands(),
			func(c *cobra.Command) string { return c.Name() },
		)

		if command.Parent() != nil {
			commandNames = append(commandNames, backNavigation)
		}

		selectedCommandName, err := renderer.SelectionMenu(cmd.Context(), "Select a command", commandNames)
		if err != nil {
			return err
		}

		if selectedCommandName == backNavigation {
			command = command.Parent()
		} else {
			command, _, err = command.Find([]string{selectedCommandName})
			if err != nil {
				return errors.Errorf("command %q not found", selectedCommandName)
			}
		}
	}

	commandPath := strings.Split(command.CommandPath(), " ")

	command.Root().SetArgs(append(commandPath, args...)[1:])

	err := command.Root().ExecuteContext(cmd.Context())
	if err != nil {
		return err //nolint:wrapcheck
	}

	return nil
}
