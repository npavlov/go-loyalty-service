package orders_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/npavlov/go-loyalty-service/internal/config"
	"github.com/npavlov/go-loyalty-service/internal/models"
	"github.com/npavlov/go-loyalty-service/internal/orders"
)

func TestSendPostRequestWithMockServer(t *testing.T) {
	t.Parallel()

	logger := zerolog.New(nil)

	t.Run("Success case", func(t *testing.T) {
		t.Parallel()
		// Mock response data
		mockResponse := &models.Accrual{
			OrderID: "12345",
			Status:  "PROCESSED",
			Accrual: float64Ptr(123.45),
		}
		responseBody, _ := json.Marshal(mockResponse)

		// Set up a mock HTTP server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/orders/12345", r.URL.Path)
			assert.Equal(t, "application/json", r.Header.Get("Accept"))

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(responseBody)
		}))
		defer server.Close()

		//nolint:exhaustruct
		cfg := &config.Config{AccrualAddress: server.URL}
		sender := orders.NewSender(cfg, &logger)

		// Call the method
		resp, err := sender.SendPostRequest(context.Background(), "12345")

		require.NoError(t, err)
		assert.Equal(t, mockResponse, resp)
	})

	t.Run("Too many requests", func(t *testing.T) {
		t.Parallel()
		// Set up a mock HTTP server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
		}))
		defer server.Close()

		//nolint:exhaustruct
		cfg := &config.Config{AccrualAddress: server.URL}
		sender := orders.NewSender(cfg, &logger)

		// Call the method
		resp, err := sender.SendPostRequest(context.Background(), "12345")

		assert.Nil(t, resp)
		require.Error(t, err)
		assert.Equal(t, orders.ErrCantProcessError, err)
	})

	t.Run("Invalid response body", func(t *testing.T) {
		t.Parallel()
		// Set up a mock HTTP server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("invalid json"))
		}))
		defer server.Close()

		//nolint:exhaustruct
		cfg := &config.Config{AccrualAddress: server.URL}
		sender := orders.NewSender(cfg, &logger)

		// Call the method
		resp, err := sender.SendPostRequest(context.Background(), "12345")

		assert.Nil(t, resp)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal response")
	})

	t.Run("Internal server error", func(t *testing.T) {
		t.Parallel()
		// Set up a mock HTTP server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		//nolint:exhaustruct
		cfg := &config.Config{AccrualAddress: server.URL}
		sender := orders.NewSender(cfg, &logger)

		// Call the method
		resp, err := sender.SendPostRequest(context.Background(), "12345")

		assert.Nil(t, resp)
		require.Error(t, err)
		assert.Equal(t, orders.ErrCantProcessError, err)
	})
}

func float64Ptr(f float64) *float64 {
	return &f
}
