package testutils

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/npavlov/go-loyalty-service/internal/models"
	"github.com/npavlov/go-loyalty-service/internal/orders"
)

const (
	bufferSize      = 100
	simulateAccrual = 100.0
	simulateTimeout = 100 * time.Millisecond
)

type MockOrders struct {
	log         *zerolog.Logger
	storage     *MockStorage
	orderChan   chan orders.KafkaOrder
	wg          sync.WaitGroup
	stopChan    chan struct{}
	processing  map[string]bool // Tracks currently processing orders
	processLock sync.Mutex      // Synchronizes access to `processing`
}

func NewMockOrders(storage *MockStorage, log *zerolog.Logger) *MockOrders {
	return &MockOrders{
		log:         log,
		storage:     storage,
		orderChan:   make(chan orders.KafkaOrder, bufferSize), // Buffered channel for async processing
		stopChan:    make(chan struct{}),
		processing:  make(map[string]bool),
		wg:          sync.WaitGroup{},
		processLock: sync.Mutex{},
	}
}

func (mo *MockOrders) AddOrder(ctx context.Context, orderNum string, userID string) error {
	data := orders.KafkaOrder{
		OrderNum: orderNum,
		UserID:   userID,
	}

	mo.log.Info().Msgf("Trying to add order %s for user %s to the queue", orderNum, userID)

	select {
	case mo.orderChan <- data:
		mo.log.Info().Interface("order", data).Msg("New order added to the mock channel")

		return nil
	case <-ctx.Done():
		mo.log.Error().Msg("Context canceled while adding order")

		return errors.Wrap(ctx.Err(), "Context canceled while adding order")
	}
}

func (mo *MockOrders) ProcessOrders(ctx context.Context) {
	mo.wg.Add(1)
	go func() {
		defer mo.wg.Done()
		for {
			select {
			case <-mo.stopChan:
				log.Println("Stopping mock order processing...")

				return
			case order := <-mo.orderChan:
				mo.processOrder(ctx, order)
			}
		}
	}()
}

func (mo *MockOrders) StopProcessing() {
	close(mo.stopChan)
	mo.wg.Wait()
}

func (mo *MockOrders) processOrder(ctx context.Context, order orders.KafkaOrder) {
	mo.log.Info().Interface("order", order).Msg("Mock processing order")

	// Check if the order is already being processed
	mo.processLock.Lock()
	if mo.processing[order.OrderNum] {
		mo.log.Warn().Str("orderNum", order.OrderNum).Msg("Order already being processed")
		mo.processLock.Unlock()

		return
	}
	mo.processing[order.OrderNum] = true
	mo.processLock.Unlock()

	defer func() {
		mo.processLock.Lock()
		delete(mo.processing, order.OrderNum)
		mo.processLock.Unlock()
	}()

	_ = mo.checkOrderStatus(ctx, order)

	mo.log.Info().Interface("order", order).Msg("Order successfully processed in mock")
}

func (mo *MockOrders) checkOrderStatus(ctx context.Context, message orders.KafkaOrder) error {
	mo.log.Info().Interface("OrderNum", message).Msg("Retrieving Order ID (mock)")

	// Simulate status update
	time.Sleep(simulateTimeout) // Simulate processing time
	update := &models.Accrual{
		OrderID: message.OrderNum,
		Status:  string(models.Processed),    // Mock processed status
		Accrual: float64Ptr(simulateAccrual), // Mock accrual value
	}

	err := mo.storage.UpdateOrder(ctx, update, message.UserID)
	if err != nil {
		return err
	}

	mo.log.Info().Str("OrderNum", message.OrderNum).Msg("Order status updated to processed")

	return nil
}
