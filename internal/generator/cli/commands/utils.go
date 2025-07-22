package commands

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	clierrors "github.com/tarantool/sdvg/internal/generator/cli/errors"
)

// NoArgs validates args and returns an error if there are any args.
func NoArgs(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}

	_ = cmd.Help()

	if cmd.HasSubCommands() {
		return clierrors.NewUsageError(errors.Errorf("unknown command: %q for %q", args[0], cmd.Name()))
	}

	return clierrors.NewUsageError(errors.Errorf("%q accepts no arguments", cmd.Name()))
}

// RequiresMaxArgs returns an error if there is not at most max args.
func RequiresMaxArgs(maxArgs int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) <= maxArgs {
			return nil
		}

		_ = cmd.Help()

		return clierrors.NewUsageError(errors.Errorf(
			"%q requires at most %d %s, received %d",
			cmd.Name(),
			maxArgs,
			pluralize("argument", maxArgs),
			len(args),
		))
	}
}

// FlagErrorFunc processes errors of CLI flags.
func FlagErrorFunc(cmd *cobra.Command, err error) error {
	if err == nil {
		return nil
	}

	_ = cmd.Help()

	return clierrors.NewUsageError(err)
}

// pluralize returns a plural word.
func pluralize(word string, number int) string {
	if number == 1 {
		return word
	}

	return word + "s"
}
