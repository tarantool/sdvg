package progress

import "github.com/tarantool/sdvg/internal/generator/usecase"

// Tracker interface implementation should display progress of tasks.
type Tracker interface {
	// AddTask function should add task progress of which should be displayed.
	AddTask(name string, title string, total uint64)
	// UpdateProgress function should update progress of tracked task.
	UpdateProgress(name string, progress usecase.Progress)
	// Wait function should wait for all tracked tasks to complete.
	Wait()
}
