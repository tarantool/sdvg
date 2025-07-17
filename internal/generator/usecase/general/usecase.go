package general

import (
	"context"
	"sync"

	"github.com/pkg/errors"

	"sdvg/internal/generator/usecase"
)

// Verify interface compliance in compile time.
var _ usecase.UseCase = (*UseCase)(nil)

// UseCase type is implementation of common use case.
type UseCase struct {
	tasks map[string]*Task
	mutex *sync.RWMutex
}

// UseCaseConfig type is used to describe config for common usecase.
type UseCaseConfig struct{}

// NewUseCase function creates UseCase object.
func NewUseCase(_ UseCaseConfig) *UseCase {
	return &UseCase{
		tasks: make(map[string]*Task),
		mutex: &sync.RWMutex{},
	}
}

// Setup function do nothing.
func (uc *UseCase) Setup() error {
	return nil
}

// CreateTask function receive model from delivery, generate data and send it to output.
// It works asynchronously and returns string task ID to get results later.
func (uc *UseCase) CreateTask(ctx context.Context, config usecase.TaskConfig) (string, error) {
	task, err := NewTask(config)
	if err != nil {
		return "", err
	}

	uc.mutex.Lock()
	uc.tasks[task.ID] = task
	uc.mutex.Unlock()

	task.RunTask(ctx, func() { uc.removeTask(task.ID) })

	return task.ID, nil
}

// GetProgress function returns current progresses of task by ID.
func (uc *UseCase) GetProgress(taskID string) (map[string]usecase.Progress, error) {
	uc.mutex.RLock()
	task, ok := uc.tasks[taskID]
	uc.mutex.RUnlock()

	if !ok {
		return nil, errors.Errorf("no task with id %s", taskID)
	}

	return task.GetProgress(), nil
}

// GetResult function returns error of task by ID.
func (uc *UseCase) GetResult(taskID string) (bool, error) {
	uc.mutex.RLock()
	task, ok := uc.tasks[taskID]
	uc.mutex.RUnlock()

	if !ok {
		return false, errors.Errorf("no task with task id %s", taskID)
	}

	return task.GetError()
}

// WaitResult function waits task by ID end and returns it error.
func (uc *UseCase) WaitResult(taskID string) error {
	uc.mutex.RLock()
	task, ok := uc.tasks[taskID]
	uc.mutex.RUnlock()

	if !ok {
		return errors.Errorf("no task with task id %s", taskID)
	}

	return task.WaitError()
}

// removeTask function removes task from local storage.
func (uc *UseCase) removeTask(taskID string) {
	uc.mutex.Lock()
	delete(uc.tasks, taskID)
	uc.mutex.Unlock()
}

// Teardown function wait all generation processes.
func (uc *UseCase) Teardown() error {
	uc.mutex.RLock()
	for _, task := range uc.tasks {
		_ = task.WaitError()
	}
	uc.mutex.RUnlock()

	return nil
}
