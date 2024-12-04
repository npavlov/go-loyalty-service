package testutils_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/npavlov/go-loyalty-service/internal/models"
	"github.com/npavlov/go-loyalty-service/internal/orders"
	testutils "github.com/npavlov/go-loyalty-service/internal/test_utils"
)

func TestMockOrders(t *testing.T) {
	t.Parallel()

	logger := zerolog.New(zerolog.NewConsoleWriter())
	mockStorage := testutils.NewMockStorage()

	t.Run("AddOrder successfully adds to the channel", func(t *testing.T) {
		t.Parallel()

		mockOrders := testutils.NewMockOrders(mockStorage, &logger)

		ctx := context.Background()

		userID := uuid.New()
		err := mockOrders.AddOrder(ctx, "order569", userID.String())
		assert.NoError(t, err, "Expected no error when adding order")
	})

	t.Run("ProcessOrders processes orders correctly", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		mockOrders := testutils.NewMockOrders(mockStorage, &logger)
		userID := uuid.New()
		order := orders.KafkaOrder{OrderNum: "order123", UserID: userID.String()}

		// Add an order to the queue
		_, err := mockStorage.CreateOrder(ctx, order.OrderNum, order.UserID)
		require.NoError(t, err, "Expected no error when creating order")

		err = mockOrders.AddOrder(ctx, order.OrderNum, order.UserID)
		require.NoError(t, err, "Expected no error when adding order")

		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			defer wg.Done()
			mockOrders.ProcessOrders(ctx)
		}()

		// Allow some time for processing
		time.Sleep(500 * time.Millisecond)

		mockOrders.StopProcessing()
		wg.Wait()

		orderCreated, found := mockStorage.GetOrder(ctx, order.OrderNum)

		assert.True(t, found)
		assert.Equal(t, userID, orderCreated.UserID)
		assert.Equal(t, models.Processed, orderCreated.Status)
	})

	t.Run("checkOrderStatus updates status correctly", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		mockOrders := testutils.NewMockOrders(mockStorage, &logger)
		userID := uuid.New()
		order := orders.KafkaOrder{OrderNum: "order456", UserID: userID.String()}
		_, err := mockStorage.CreateOrder(ctx, order.OrderNum, order.UserID)
		require.NoError(t, err, "Expected no error when adding order")

		err = mockOrders.CheckOrderStatus(ctx, order)
		require.NoError(t, err, "Expected no error during checkOrderStatus")

		orderCreated, found := mockStorage.GetOrder(ctx, order.OrderNum)

		assert.True(t, found)
		assert.Equal(t, userID, orderCreated.UserID)
		assert.Equal(t, models.Processed, orderCreated.Status)
	})
}
