package utils_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/npavlov/go-loyalty-service/internal/utils"
)

// MockOperation is a helper type to simulate operations with retry logic.
type MockOperation struct {
	mock.Mock
}

func (m *MockOperation) Execute() error {
	args := m.Called()

	return args.Error(0)
}

func TestRetryOperation_SuccessFirstAttempt(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockOperation := new(MockOperation)

	mockOperation.On("Execute").Return(nil).Once()

	err := utils.RetryOperation(ctx, mockOperation.Execute)
	require.NoError(t, err, "Expected no error on first attempt")
	mockOperation.AssertExpectations(t)
}

func TestRetryOperation_SuccessAfterRetries(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockOperation := new(MockOperation)

	// Simulate failure twice, then success
	//nolint:err113
	mockOperation.On("Execute").Return(errors.New("temporary error")).Twice()
	mockOperation.On("Execute").Return(nil).Once()

	err := utils.RetryOperation(ctx, mockOperation.Execute)
	require.NoError(t, err, "Expected no error after retries")
	mockOperation.AssertExpectations(t)
}

func TestRetryOperation_FailureAfterExhaustingRetries(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockOperation := new(MockOperation)

	// Simulate failure for all attempts
	//nolint:err113
	mockOperation.On("Execute").Return(errors.New("permanent error")).Times(4)

	err := utils.RetryOperation(ctx, mockOperation.Execute)
	require.Error(t, err, "Expected error after exhausting retries")
	assert.Contains(t, err.Error(), "failed to execute operation after retry", "Error message should wrap the failure")
	mockOperation.AssertExpectations(t)
}

func TestRetryOperation_ContextCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	mockOperation := new(MockOperation)

	// Simulate a single call and cancel the context
	//nolint:err113
	mockOperation.On("Execute").Return(errors.New("temporary error")).Once()
	cancel()

	err := utils.RetryOperation(ctx, mockOperation.Execute)
	require.Error(t, err, "Expected error when context is canceled")
	assert.Contains(t, err.Error(), "failed to execute operation after retry", "Error message should wrap the failure")
	mockOperation.AssertExpectations(t)
}
