package confirm

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync/atomic"

	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/tarantool/sdvg/internal/generator/cli/render"
	"github.com/tarantool/sdvg/internal/generator/cli/utils"
)

var ErrPromptFailed = errors.New("prompt failed")

// Confirm asks user a yes/no question. Returns true for “yes”.
type Confirm func(ctx context.Context, question string) (bool, error)

func BuildConfirmTTY(in io.Reader, out io.Writer) func(ctx context.Context, question string) (bool, error) {
	return func(ctx context.Context, question string) (bool, error) {
		fmt.Fprintln(out)

		cancelableIn := newCancelableReader(in)
		defer cancelableIn.Close()

		prompt := promptui.Prompt{
			Label:   question + " [y/N]: ",
			Default: "y",
			Stdin:   cancelableIn,
			Stdout:  utils.DummyReadWriteCloser{Writer: out},
		}
		validate := func(s string) error {
			if len(s) == 1 && strings.Contains("YyNn", s) || prompt.Default != "" && len(s) == 0 {
				return nil
			}
			return errors.New("invalid input")
		}
		prompt.Validate = validate

		var (
			input          string
			err            error
			promptFinished = make(chan struct{})
		)

		go func() {
			input, err = prompt.Run() // goroutine will block here until user input

			promptFinished <- struct{}{}
		}()

		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-promptFinished:
		}

		if err != nil {
			return false, fmt.Errorf("%w: %v", ErrPromptFailed, err)
		}

		return strings.Contains("Yy", input), nil
	}
}

func BuildConfirmNoTTY(in render.Renderer, out io.Writer, isUpdatePaused *atomic.Bool) func(ctx context.Context, question string) (bool, error) {
	return func(ctx context.Context, question string) (bool, error) {
		// here we pause ProgressLogManager to stop sending progress messages
		isUpdatePaused.Store(true)
		defer isUpdatePaused.Store(false)

		for {
			fmt.Fprintf(out, "%s [y/N]: ", question)

			var (
				input             string
				err               error
				inputReadFinished = make(chan struct{})
			)

			go func() {
				input, err = in.ReadLine() // goroutine will block here until user input

				inputReadFinished <- struct{}{}
			}()

			select {
			case <-ctx.Done():
				return false, ctx.Err()
			case <-inputReadFinished:
			}

			if err != nil {
				return false, err
			}

			if !in.IsTerminal() {
				fmt.Fprintln(out, input)
			}

			switch strings.ToLower(strings.TrimSpace(input)) {
			case "y", "yes":
				return true, nil
			case "", "n", "no":
				return false, nil
			default:
				fmt.Fprintln(out, "Please enter y or n")
			}
		}
	}
}
