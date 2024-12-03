package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	handlers "github.com/npavlov/go-loyalty-service/internal/handlers/orders"
	"github.com/npavlov/go-loyalty-service/internal/middlewares"
	"github.com/npavlov/go-loyalty-service/internal/models"
	testutils "github.com/npavlov/go-loyalty-service/internal/test_utils"
)

func TestHandlerOrders_GetOrders(t *testing.T) {
	t.Parallel()

	logger := zerolog.New(nil)
	mockStorage := testutils.NewMockStorage()
	orderProcessor := testutils.NewMockOrders(mockStorage, &logger)
	handler := handlers.NewOrdersHandler(mockStorage, orderProcessor, &logger)

	// Set up mock data
	userID := uuid.New().String()
	orderNum := testutils.GenerateLuhnNumber(16)
	orderID, err := mockStorage.CreateOrder(context.Background(), orderNum, userID)
	require.NoError(t, err)

	_, err = uuid.Parse(orderID)
	require.NoError(t, err)

	// Create a request with the userID in context
	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	ctx := context.WithValue(req.Context(), middlewares.UserIDKey, userID)
	req = req.WithContext(ctx)
	resp := httptest.NewRecorder()

	// Call the handler
	handler.GetOrders(resp, req)

	// Check response
	assert.Equal(t, http.StatusOK, resp.Code)

	var orders []models.Order
	err = json.Unmarshal(resp.Body.Bytes(), &orders)
	require.NoError(t, err)
	assert.Len(t, orders, 1)
	assert.Equal(t, orderNum, orders[0].OrderID)
	assert.Equal(t, models.NewStatus, orders[0].Status)
}

func TestHandlerOrders_Create(t *testing.T) {
	t.Parallel()

	logger := zerolog.New(nil)
	mockStorage := testutils.NewMockStorage()
	orderProcessor := testutils.NewMockOrders(mockStorage, &logger)
	handler := handlers.NewOrdersHandler(mockStorage, orderProcessor, &logger)

	userID := uuid.New().String()

	t.Run("Valid order creation", func(t *testing.T) {
		t.Parallel()

		orderID := testutils.GenerateLuhnNumber(16)
		req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader([]byte(orderID)))
		ctx := context.WithValue(req.Context(), middlewares.UserIDKey, userID)
		req = req.WithContext(ctx)
		resp := httptest.NewRecorder()

		handler.Create(resp, req)

		assert.Equal(t, http.StatusAccepted, resp.Code)

		storedOrder, _ := mockStorage.GetOrder(context.Background(), orderID)
		assert.NotNil(t, storedOrder)
		assert.Equal(t, orderID, storedOrder.OrderID)
	})

	t.Run("Invalid order number", func(t *testing.T) {
		t.Parallel()

		invalidOrderID := "12345" // Luhn invalid
		req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader([]byte(invalidOrderID)))
		ctx := context.WithValue(req.Context(), middlewares.UserIDKey, userID)
		req = req.WithContext(ctx)
		resp := httptest.NewRecorder()

		handler.Create(resp, req)

		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	})

	t.Run("Order already exists for same user", func(t *testing.T) {
		t.Parallel()

		orderID := testutils.GenerateLuhnNumber(16)
		_, err := mockStorage.CreateOrder(context.Background(), orderID, userID)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader([]byte(orderID)))
		ctx := context.WithValue(req.Context(), middlewares.UserIDKey, userID)
		req = req.WithContext(ctx)
		resp := httptest.NewRecorder()

		handler.Create(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("Order already exists for different user", func(t *testing.T) {
		t.Parallel()

		orderID := testutils.GenerateLuhnNumber(16)
		otherUserID := uuid.New().String()
		_, err := mockStorage.CreateOrder(context.Background(), orderID, otherUserID)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader([]byte(orderID)))
		ctx := context.WithValue(req.Context(), middlewares.UserIDKey, userID)
		req = req.WithContext(ctx)
		resp := httptest.NewRecorder()

		handler.Create(resp, req)

		assert.Equal(t, http.StatusConflict, resp.Code)
	})
}

func TestMockOrders_ProcessOrders(t *testing.T) {
	t.Parallel()

	logger := zerolog.New(nil)
	mockStorage := testutils.NewMockStorage()

	t.Run("Single Order Processing", func(t *testing.T) {
		t.Parallel()

		userID := uuid.New().String()
		orderID := testutils.GenerateLuhnNumber(16)
		mockOrders := testutils.NewMockOrders(mockStorage, &logger)

		// Create an order in storage
		_, err := mockStorage.CreateOrder(context.Background(), orderID, userID)
		require.NoError(t, err)

		// Add the order to the processing queue
		err = mockOrders.AddOrder(context.Background(), orderID, userID)
		require.NoError(t, err)

		// Start processing in a separate goroutine
		go mockOrders.ProcessOrders(context.Background())

		// Wait for a moment to allow processing
		time.Sleep(500 * time.Millisecond)

		// Check if the order was processed
		order, found := mockStorage.GetOrder(context.Background(), orderID)
		require.True(t, found)
		assert.Equal(t, models.Processed, order.Status)

		mockOrders.StopProcessing()
	})

	t.Run("Multiple Orders Processing", func(t *testing.T) {
		t.Parallel()

		mockOrders := testutils.NewMockOrders(mockStorage, &logger)

		userID := uuid.New().String()
		orderID1 := testutils.GenerateLuhnNumber(16)
		orderID2 := testutils.GenerateLuhnNumber(16)
		orderID3 := testutils.GenerateLuhnNumber(16)

		orderIDs := []string{orderID1, orderID2, orderID3}

		// Add orders to storage and queue
		for _, orderID := range orderIDs {
			_, err := mockStorage.CreateOrder(context.Background(), orderID, userID)
			require.NoError(t, err)

			err = mockOrders.AddOrder(context.Background(), orderID, userID)
			require.NoError(t, err)
		}

		// Start processing
		go mockOrders.ProcessOrders(context.Background())

		// Wait for processing to complete
		time.Sleep(1 * time.Second)

		// Check if all orders were processed
		for _, orderID := range orderIDs {
			order, found := mockStorage.GetOrder(context.Background(), orderID)
			require.True(t, found)
			assert.Equal(t, models.Processed, order.Status)
		}

		mockOrders.StopProcessing()
	})
}
