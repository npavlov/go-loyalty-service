package balance_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/npavlov/go-loyalty-service/internal/handlers/balance"
	testutils "github.com/npavlov/go-loyalty-service/internal/test_utils"
	"github.com/stretchr/testify/assert"

	"github.com/npavlov/go-loyalty-service/internal/models"
	"github.com/rs/zerolog"
)

var mockedUserId = "test-user-id"

func setupHandlerBalance() (*balance.HandlerBalance, *testutils.MockStorage, context.Context) {
	logger := zerolog.New(nil)
	mockStorage := testutils.NewMockStorage()
	handler := balance.NewBalanceHandler(mockStorage, &logger)
	ctx := context.WithValue(context.Background(), "userID", mockedUserId)
	return handler, mockStorage, ctx
}

func TestHandlerBalance_GetBalance(t *testing.T) {
	handler, mockStorage, ctx := setupHandlerBalance()

	mockStorage.Balances[mockedUserId] = &models.Balance{
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
	assert.NoError(t, err)
	assert.Equal(t, 100.0, response.Balance)
	assert.Equal(t, 20.0, response.Withdrawn)
}

func TestHandlerBalance_MakeWithdrawal(t *testing.T) {
	handler, mockStorage, ctx := setupHandlerBalance()

	mockStorage.Balances[mockedUserId] = &models.Balance{
		Balance:   100.0,
		Withdrawn: 20.0,
	}

	// Prepare the request body
	withdrawal := models.MakeWithdrawal{
		Order: testutils.GenerateLuhnNumber(16),
		Sum:   50.0,
	}
	body, _ := json.Marshal(withdrawal)
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
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 50.0, response.Balance)
	assert.Equal(t, 70.0, response.Withdrawn)
}

func TestHandlerBalance_MakeWithdrawal_InsufficientBalance(t *testing.T) {
	handler, mockStorage, ctx := setupHandlerBalance()

	// Mock the storage behavior
	mockStorage.Balances[mockedUserId] = &models.Balance{
		Balance:   30.0,
		Withdrawn: 20.0,
	}

	// Prepare the request body
	withdrawal := models.MakeWithdrawal{
		Order: testutils.GenerateLuhnNumber(16),
		Sum:   50.0,
	}
	body, _ := json.Marshal(withdrawal)
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
	handler, mockStorage, ctx := setupHandlerBalance()

	mockStorage.Balances[mockedUserId] = &models.Balance{
		Balance:   100.0,
		Withdrawn: 20.0,
	}
	// Mock the storage behavior
	withdrawalOrderNum := testutils.GenerateLuhnNumber(16)
	withdrawals := map[string]*models.Withdrawal{
		withdrawalOrderNum: {OrderId: withdrawalOrderNum, Sum: float64Ptr(50.0), UserId: mockedUserId},
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
	assert.NoError(t, err)
	assert.Len(t, response, 1)
	assert.Equal(t, withdrawalOrderNum, response[0].OrderId)
}

func TestHandlerBalance_GetWithdrawals_NoContent(t *testing.T) {
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

func float64Ptr(f float64) *float64 {
	return &f
}
