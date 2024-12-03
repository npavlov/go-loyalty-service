package balance_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/npavlov/go-loyalty-service/internal/handlers/balance"
	"github.com/npavlov/go-loyalty-service/internal/middlewares"
	"github.com/npavlov/go-loyalty-service/internal/models"
	testutils "github.com/npavlov/go-loyalty-service/internal/test_utils"
)

const mockedUserID = "test-user-id"

func setupHandlerBalance() (*balance.HandlerBalance, *testutils.MockStorage, context.Context) {
	logger := zerolog.New(nil)
	mockStorage := testutils.NewMockStorage()
	handler := balance.NewBalanceHandler(mockStorage, &logger)

	ctx := context.WithValue(context.Background(), middlewares.UserIDKey, mockedUserID)

	return handler, mockStorage, ctx
}

func TestHandlerBalance_GetBalance(t *testing.T) {
	t.Parallel()

	handler, mockStorage, ctx := setupHandlerBalance()

	mockStorage.Balances[mockedUserID] = &models.Balance{
		Balance:   100.0,
		Withdrawn: 20.0,
	}

	// Prepare the request
	req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	// Call the handler
	handler.GetBalance(rec, req)

	// Assert response
	assert.Equal(t, http.StatusOK, rec.Code)

	var response models.Balance
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.InDelta(t, 100.0, response.Balance, 0.001)
	assert.InDelta(t, 20.0, response.Withdrawn, 0.001)
}

func TestHandlerBalance_MakeWithdrawal(t *testing.T) {
	t.Parallel()

	handler, mockStorage, ctx := setupHandlerBalance()

	mockStorage.Balances[mockedUserID] = &models.Balance{
		Balance:   100.0,
		Withdrawn: 20.0,
	}

	// Prepare the request body
	withdrawal := models.MakeWithdrawal{
		Order: testutils.GenerateLuhnNumber(16),
		Sum:   50.0,
	}
	body, err := json.Marshal(withdrawal)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/user/withdraw", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	// Call the handler
	handler.MakeWithdrawal(rec, req)

	// Assert response
	assert.Equal(t, http.StatusOK, rec.Code)

	// Call the handler
	handler.GetBalance(rec, req)

	// Assert response
	assert.Equal(t, http.StatusOK, rec.Code)

	var response models.Balance
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.InDelta(t, 50.0, response.Balance, 0.001)
	assert.InDelta(t, 70.0, response.Withdrawn, 0.001)
}

func TestHandlerBalance_MakeWithdrawal_InsufficientBalance(t *testing.T) {
	t.Parallel()

	handler, mockStorage, ctx := setupHandlerBalance()

	// Mock the storage behavior
	mockStorage.Balances[mockedUserID] = &models.Balance{
		Balance:   30.0,
		Withdrawn: 20.0,
	}

	// Prepare the request body
	withdrawal := models.MakeWithdrawal{
		Order: testutils.GenerateLuhnNumber(16),
		Sum:   50.0,
	}
	body, err := json.Marshal(withdrawal)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/user/withdraw", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	// Call the handler
	handler.MakeWithdrawal(rec, req)

	// Assert response
	assert.Equal(t, http.StatusPaymentRequired, rec.Code)
	assert.Contains(t, rec.Body.String(), "insufficient balance")
}

func TestHandlerBalance_GetWithdrawals(t *testing.T) {
	t.Parallel()

	handler, mockStorage, ctx := setupHandlerBalance()

	mockStorage.Balances[mockedUserID] = &models.Balance{
		Balance:   100.0,
		Withdrawn: 20.0,
	}
	// Mock the storage behavior
	withdrawalOrderNum := testutils.GenerateLuhnNumber(16)
	withdrawals := map[string]*models.Withdrawal{
		withdrawalOrderNum: {OrderID: withdrawalOrderNum, Sum: float64Ptr(50.0), UserID: mockedUserID},
	}
	mockStorage.Withdrawals = withdrawals

	// Prepare the request
	req := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	// Call the handler
	handler.GetWithdrawals(rec, req)

	// Assert response
	assert.Equal(t, http.StatusOK, rec.Code)

	var response []models.Withdrawal
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Len(t, response, 1)
	assert.Equal(t, withdrawalOrderNum, response[0].OrderID)
}

func TestHandlerBalance_GetWithdrawals_NoContent(t *testing.T) {
	t.Parallel()

	handler, mockStorage, ctx := setupHandlerBalance()

	// Mock the storage behavior
	mockStorage.Withdrawals = make(map[string]*models.Withdrawal)

	// Prepare the request
	req := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	// Call the handler
	handler.GetWithdrawals(rec, req)

	// Assert response
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestHandlerBalance_GetBalance_UserNotFound(t *testing.T) {
	t.Parallel()

	handler, _, _ := setupHandlerBalance()

	// Create a request with no user ID in the context
	req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
	rec := httptest.NewRecorder()

	// Call the handler
	handler.GetBalance(rec, req)

	// Assert response
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), "user id not found")
}

func TestHandlerBalance_MakeWithdrawal_UserNotFound(t *testing.T) {
	t.Parallel()

	handler, _, _ := setupHandlerBalance()

	withdrawal := models.MakeWithdrawal{
		Order: testutils.GenerateLuhnNumber(16),
		Sum:   50.0,
	}
	body, err := json.Marshal(withdrawal)
	require.NoError(t, err)

	// Create a request with no user ID in the context
	req := httptest.NewRequest(http.MethodPost, "/api/user/withdraw", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// Call the handler
	handler.MakeWithdrawal(rec, req)

	// Assert response
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), "user id not found")
}

func TestHandlerBalance_GetWithdrawals_UserNotFound(t *testing.T) {
	t.Parallel()

	handler, _, _ := setupHandlerBalance()

	// Create a request with no user ID in the context
	req := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)
	rec := httptest.NewRecorder()

	// Call the handler
	handler.GetWithdrawals(rec, req)

	// Assert response
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), "user id not found")
}

func TestHandlerBalance_MakeWithdrawal_InvalidOrderNumber(t *testing.T) {
	t.Parallel()

	handler, _, ctx := setupHandlerBalance()

	withdrawal := models.MakeWithdrawal{
		Order: "invalid-order", // This is not a valid Luhn number
		Sum:   50.0,
	}
	body, err := json.Marshal(withdrawal)
	require.NoError(t, err)

	// Create a request
	req := httptest.NewRequest(http.MethodPost, "/api/user/withdraw", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	// Call the handler
	handler.MakeWithdrawal(rec, req)

	// Assert response
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid order number")
}

func float64Ptr(f float64) *float64 {
	return &f
}
