package log

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tarantool/sdvg/internal/generator/cli/progress"
	"github.com/tarantool/sdvg/internal/generator/cli/utils"
	"github.com/tarantool/sdvg/internal/generator/usecase"
)

// Verify interface compliance in compile time.
var _ progress.Tracker = (*ProgressLogManager)(nil)

const (
	intervals = 50
	template  = "%s %d%% (%d / %d) ETA %s"
)

// task type used to describe task for tracking.
type task struct {
	title           string
	total           uint64
	current         uint64
	lastUpdate      time.Time
	durations       []time.Duration
	completed       []uint64
	currentInterval uint
}

// isDone checks if task is ready.
func (t *task) isDone() bool {
	return t.current == t.total
}

// ProgressLogManager type is implementation of progress.Tracker that using logger.
type ProgressLogManager struct {
	ctx   context.Context //nolint:containedctx
	tasks map[string]*task
	wg    sync.WaitGroup

	isUpdatePaused *atomic.Bool
}

// NewProgressLogManager creates NewProgressLogManager object. isUpdatePaused is used to pause UpdateProgress.
func NewProgressLogManager(ctx context.Context, isUpdatePaused *atomic.Bool) progress.Tracker {
	return &ProgressLogManager{
		ctx:            ctx,
		tasks:          make(map[string]*task),
		isUpdatePaused: isUpdatePaused,
	}
}

// AddTask adds task to manager.
func (p *ProgressLogManager) AddTask(name, title string, total uint64) {
	if _, ok := p.tasks[name]; ok {
		return
	}

	p.tasks[name] = &task{
		title:      title,
		total:      total,
		current:    0,
		lastUpdate: time.Now(),
		durations:  make([]time.Duration, intervals),
		completed:  make([]uint64, intervals),
	}

	p.wg.Add(1)
}

// UpdateProgress updates progress for task with passed name.
func (p *ProgressLogManager) UpdateProgress(name string, progress usecase.Progress) {
	t := p.tasks[name]

	if t.isDone() {
		return
	}

	for p.isUpdatePaused.Load() {
		if t.isDone() {
			return
		}
	}

	p.updateIntervals(t, progress.Done)

	t.current = progress.Done
	t.lastUpdate = time.Now()

	percentage := utils.GetPercentage(progress.Total, progress.Done)
	averageETA := p.eta(t)

	slog.Info(fmt.Sprintf(template, t.title, percentage, t.current, t.total, averageETA))

	if t.isDone() {
		p.wg.Done()
	}
}

// Wait waits for all tasks to complete.
func (p *ProgressLogManager) Wait() {
	done := make(chan struct{})

	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-p.ctx.Done():
	case <-done:
	}
}

func (p *ProgressLogManager) updateIntervals(t *task, done uint64) {
	t.durations[t.currentInterval] = time.Since(t.lastUpdate)
	t.completed[t.currentInterval] = done - t.current
	t.currentInterval = (t.currentInterval + 1) % intervals
}

//nolint:mnd
func (p *ProgressLogManager) eta(t *task) string {
	var (
		remaining        time.Duration
		overallDuration  time.Duration
		overallCompleted uint64
	)

	for i := range intervals {
		overallDuration += t.durations[i]
		overallCompleted += t.completed[i]
	}

	if overallCompleted > 0 {
		averageDurationPerItem := math.Round(float64(overallDuration) / float64(overallCompleted))
		remaining = time.Duration((t.total - t.current) * uint64(averageDurationPerItem))
	}

	hours := int64(remaining/time.Hour) % 60
	minutes := int64(remaining/time.Minute) % 60
	seconds := int64(remaining/time.Second) % 60

	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

// Write writes to default stdout.
func (p *ProgressLogManager) Write(b []byte) (int, error) {
	return os.Stdout.Write(b)
}
