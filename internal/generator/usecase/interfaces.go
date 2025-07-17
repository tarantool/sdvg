package usecase

import (
	"context"
)

// UseCase interface implementation should listen delivery and generate data by params.
//
//go:generate go run github.com/vektra/mockery/v2@v2.51.1 --name=UseCase --output=mock --outpkg=mock
type UseCase interface {
	// Setup function should configure some use case parameters.
	Setup() error
	// CreateTask function should start task to generate data and send it to output.
	CreateTask(ctx context.Context, config TaskConfig) (string, error)
	// GetProgress should return progress of data generation
	GetProgress(taskID string) (map[string]Progress, error)
	// GetResult should return task status (completed or not) and an error if necessary.
	GetResult(taskID string) (bool, error)
	// WaitResult should wait data generation and return error if needed
	WaitResult(taskID string) error
	// Teardown function should wait generation finish
	Teardown() error
}
