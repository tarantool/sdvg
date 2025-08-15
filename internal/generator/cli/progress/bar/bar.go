package bar

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/pkg/errors"
	"github.com/tarantool/sdvg/internal/generator/cli/progress"
	"github.com/tarantool/sdvg/internal/generator/usecase"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

const intervals = 50

// Verify interface compliance in compile time.
var _ progress.Tracker = (*ProgressBarManager)(nil)

type task struct {
	bar        *mpb.Bar
	lastUpdate time.Time
}

// ProgressBarManager type is implementation of progress.Tracker that using progress bar.
type ProgressBarManager struct {
	progressManager *mpb.Progress
	tasks           map[string]*task
}

// NewProgressBarManager creates ProgressBarManager object.
func NewProgressBarManager(ctx context.Context) progress.Tracker {
	return &ProgressBarManager{
		progressManager: mpb.NewWithContext(ctx),
		tasks:           make(map[string]*task),
	}
}

// AddTask adds progress bar for task to manager.
func (p *ProgressBarManager) AddTask(name, title string, total uint64) {
	if _, ok := p.tasks[name]; ok {
		return
	}

	currentTime := time.Now().Format("2006/01/02 15:04:05")
	barMessage := fmt.Sprintf("%s INFO %s", currentTime, title)

	bar, err := p.progressManager.Add(
		int64(total),
		mpb.BarStyle().Build(),
		mpb.PrependDecorators(
			decor.Name(barMessage, decor.WC{C: decor.DSyncSpaceR}),
			decor.CountersNoUnit("%d / %d"),
		),
		mpb.AppendDecorators(
			decor.Percentage(decor.WC{C: decor.DSyncSpaceR}),
			decor.Name("ETA", decor.WC{C: decor.DSyncSpaceR}),
			decor.EwmaETA(decor.ET_STYLE_HHMMSS, intervals),
		),
	)
	if err != nil && errors.Is(err, mpb.DoneError) {
		slog.Error("failed to add progress bar", slog.String("error", err.Error()))
	}

	p.tasks[name] = &task{bar: bar, lastUpdate: time.Now()}
}

// UpdateProgress updates progress for task with passed name.
func (p *ProgressBarManager) UpdateProgress(name string, progress usecase.Progress) {
	t := p.tasks[name]
	t.bar.EwmaSetCurrent(int64(progress.Done), time.Since(t.lastUpdate))
	t.lastUpdate = time.Now()
}

// Wait waits for all progress bars to complete.
func (p *ProgressBarManager) Wait() {
	p.progressManager.Wait()
}

// Write writes to stdout.
func (p *ProgressBarManager) Write(b []byte) (int, error) {
	return p.progressManager.Write(b) //nolint:wrapcheck
}
