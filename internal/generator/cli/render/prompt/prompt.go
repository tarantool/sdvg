package prompt

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh/spinner"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/tarantool/sdvg/internal/generator/cli/render"
	"github.com/tarantool/sdvg/internal/generator/cli/render/assets"
	"github.com/tarantool/sdvg/internal/generator/cli/streams"
)

const backNavigation = "back"
const editorEnvVar = "EDITOR"
const defaultEditor = "nano"

var editorsWhiteList = []string{"nano", "vim", "vi", "nvim", "ne", "emacs", "mcedit", "tilde", "micro"}

// Verify interface compliance in compile time.
var _ render.Renderer = (*Renderer)(nil)

type result struct {
	value string
	err   error
}

// Renderer type is implementation of renderer that using prompts.
type Renderer struct {
	useTTY  bool
	in      *streams.In
	out     *streams.Out
	scanner *bufio.Scanner
}

// NewRenderer creates Renderer object.
func NewRenderer(in *streams.In, out *streams.Out, useTTY bool) *Renderer {
	return &Renderer{
		useTTY:  useTTY,
		in:      in,
		out:     out,
		scanner: bufio.NewScanner(in),
	}
}

// Logo function just display logo.
func (r *Renderer) Logo() {
	_, _ = fmt.Fprint(r.out, assets.LogoText)
}

// SelectionMenu function just display selection menu.
func (r *Renderer) SelectionMenu(ctx context.Context, title string, items []string) (string, error) {
	resultChan := make(chan result, 1)

	go func() {
		title = strings.TrimSpace(title)

		if r.useTTY {
			prompt := r.selectionPrompt(title, items)

			_, value, err := prompt.Run()
			if err != nil {
				resultChan <- result{err: errors.New(err.Error())}

				return
			}

			resultChan <- result{value: value}

			return
		}

		_, _ = fmt.Fprintln(r.out, title)

		itemsMap := make(map[string]string, len(items))

		for i, item := range items {
			itemsMap[strconv.Itoa(i+1)] = item
			_, _ = fmt.Fprintf(r.out, "%d. %s\n", i+1, item)
		}

		for {
			_, _ = fmt.Fprint(r.out, "Write a number: ")

			input, err := r.ReadLine()
			if err != nil {
				resultChan <- result{err: err}

				return
			}

			if !r.in.IsTerminal() {
				_, _ = fmt.Fprintf(r.out, "%s\n", input)
			}

			value, ok := itemsMap[input]
			if ok {
				_, _ = fmt.Fprintf(r.out, "Selected: %s\n", value)

				resultChan <- result{value: value}

				return
			}

			_, _ = fmt.Fprintln(r.out, "invalid input, please try again")
		}
	}()

	select {
	case <-ctx.Done():
		return "", errors.New(ctx.Err().Error())
	case res := <-resultChan:
		return res.value, res.err
	}
}

// InputMenu function just display input menu.
func (r *Renderer) InputMenu(ctx context.Context, title string, validateFunc func(string) error) (string, error) {
	resultChan := make(chan result, 1)

	go func() {
		title = strings.TrimSpace(title)

		if r.useTTY {
			prompt := r.stringInputPrompt(title, validateFunc)

			value, err := prompt.Run()
			if err != nil {
				resultChan <- result{err: errors.New(err.Error())}

				return
			}

			resultChan <- result{value: value}

			return
		}

		for {
			_, _ = fmt.Fprintf(r.out, "%s: ", title)

			input, err := r.ReadLine()
			if err != nil {
				resultChan <- result{err: err}

				return
			}

			if !r.in.IsTerminal() {
				_, _ = fmt.Fprintf(r.out, "%s\n", input)
			}

			err = validateFunc(input)
			if err == nil {
				resultChan <- result{value: input}

				return
			}

			_, _ = fmt.Fprintln(r.out, err.Error())
		}
	}()

	select {
	case <-ctx.Done():
		return "", errors.New(ctx.Err().Error())
	case res := <-resultChan:
		return res.value, res.err
	}
}

func (r *Renderer) TextMenu(ctx context.Context, title string) (string, error) {
	resultChan := make(chan result, 1)

	go func() {
		title = strings.TrimSpace(title)

		if r.useTTY {
			tmpFilePath, err := r.openEditor(title)
			if err != nil {
				resultChan <- result{err: errors.New(err.Error())}

				return
			}

			defer os.Remove(tmpFilePath)

			value, err := r.readFile(tmpFilePath)
			if err != nil {
				resultChan <- result{err: errors.New(err.Error())}

				return
			}

			resultChan <- result{value: value}

			return
		}

		_, _ = fmt.Fprintln(r.out, title)

		input, err := r.readMultiline()
		if err != nil {
			resultChan <- result{err: err}

			return
		}

		if !r.in.IsTerminal() {
			_, _ = fmt.Fprintf(r.out, "%s\n", input)
		}

		resultChan <- result{value: input}
	}()

	select {
	case <-ctx.Done():
		return "", errors.New(ctx.Err().Error())
	case res := <-resultChan:
		return res.value, res.err
	}
}

// WithSpinner starts spinner while function is running.
func (r *Renderer) WithSpinner(title string, fn func()) {
	if r.useTTY {
		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			defer cancel()
			fn()
		}()

		_ = spinner.New().
			Title(title).
			Context(ctx).
			Run()

		return
	}

	_, _ = fmt.Fprintln(r.out, title)

	fn()
}

// ReadLine reads input from stdin.
func (r *Renderer) ReadLine() (string, error) {
	if r.scanner.Scan() {
		return strings.TrimSpace(r.scanner.Text()), nil
	}

	if err := r.scanner.Err(); err != nil {
		return "", errors.New(err.Error())
	}

	return "", errors.New(io.EOF.Error())
}

// IsTerminal returns true if this stream is connected to a terminal.
func (r *Renderer) IsTerminal() bool {
	return r.in.IsTerminal()
}

func (r *Renderer) Read(p []byte) (int, error) {
	return r.in.Read(p)
}

// selectionPrompt returns prompt for selection items.
func (r *Renderer) selectionPrompt(title string, items []string) promptui.Select {
	templates := &promptui.SelectTemplates{
		Label: "{{ . }}",
		Active: fmt.Sprintf(
			"  {{ if eq . \"%s\" }}> {{ . | red }}{{ else }}> {{ . | cyan }}{{ end }}",
			backNavigation),
		Inactive: fmt.Sprintf(
			"{{ if eq . \"%s\" }}  {{ . | red }}{{ else }}  {{ . }}{{ end }}",
			backNavigation),
		Selected: "\U00002714 {{ . | green }}",
	}

	//nolint:mnd
	return promptui.Select{
		Stdin:     r.in,
		Stdout:    r.out,
		Label:     title,
		Items:     items,
		Templates: templates,
		HideHelp:  true,
		Size:      10,
	}
}

// stringInputPrompt returns prompt for string input.
func (r *Renderer) stringInputPrompt(title string, validateFunc func(string) error) promptui.Prompt {
	templates := &promptui.PromptTemplates{
		Prompt:  "{{ . }} ",
		Success: "{{ . | bold }} ",
	}

	return promptui.Prompt{
		Stdin:     r.in,
		Stdout:    r.out,
		Label:     title,
		Templates: templates,
		Validate:  validateFunc,
	}
}

func (r *Renderer) getEditor() (string, error) {
	editor := os.Getenv(editorEnvVar)

	if editor == "" {
		editor = defaultEditor
	}

	if !slices.Contains(editorsWhiteList, editor) {
		return "", errors.New(
			fmt.Sprintf("only '%v' can be used as an editor, got '%s'", editorsWhiteList, editor),
		)
	}

	return editor, nil
}

// openEditor opens temp file in editor.
func (r *Renderer) openEditor(title string) (string, error) {
	tmpFile, err := os.CreateTemp(os.TempDir(), "temp-*.txt")
	if err != nil {
		return "", errors.New(err.Error())
	}

	_, err = tmpFile.WriteString(
		strings.Join(
			[]string{"", title, "# Lines starting with '#' will be ignored."},
			"\n",
		),
	)

	if err != nil {
		return "", errors.New(err.Error())
	}

	err = tmpFile.Close()
	if err != nil {
		return "", errors.New(err.Error())
	}

	editor, err := r.getEditor()
	if err != nil {
		return "", errors.New(err.Error())
	}

	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return "", errors.New(err.Error())
	}

	return tmpFile.Name(), nil
}

// readFile reads lines from file without lines starting with '#'.
func (r *Renderer) readFile(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", errors.New(err.Error())
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")

	var sb strings.Builder

	for _, line := range lines {
		if !strings.HasPrefix(line, "#") {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}

	return strings.TrimSpace(sb.String()), nil
}

func (r *Renderer) readMultiline() (string, error) {
	var sb strings.Builder

	for r.scanner.Scan() {
		sb.WriteString(r.scanner.Text())
		sb.WriteString("\n")
	}

	if err := r.scanner.Err(); err != nil {
		return "", errors.New(err.Error())
	}

	input := strings.TrimSpace(sb.String())

	if input == "" {
		return "", errors.New(io.EOF.Error())
	}

	return input, nil
}
