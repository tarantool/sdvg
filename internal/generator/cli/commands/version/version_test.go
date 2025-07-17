package version

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"sdvg/internal/generator/cli/options"
	"sdvg/internal/generator/cli/streams"
)

func TestNewVersionCommand(t *testing.T) {
	t.Helper()

	expected := "SDVG version 1.0.0"
	out := new(bytes.Buffer)

	cliOpts := &options.CliOptions{}
	cliOpts.SetVersion("1.0.0")
	cliOpts.SetOut(streams.NewOut(out))

	cmd := NewVersionCommand(cliOpts)
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	require.NoError(t, err)
	require.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(out.String()))
}
