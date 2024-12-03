package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/npavlov/go-loyalty-service/internal/models"
	"github.com/npavlov/go-loyalty-service/internal/storage"
)

func TestDBStorage_AddUser(t *testing.T) {
	mockPool, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer mockPool.Close()

	log := zerolog.New(nil)
	dbStorage := storage.NewDBStorage(mockPool, &log)

	ctx := context.Background()
	username := "testuser"
	passwordHash := "hashedpassword"
	expectedUserID := "123e4567-e89b-12d3-a456-426614174000"

	mockPool.ExpectQuery(`INSERT INTO users`).
		WithArgs(username, passwordHash).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(expectedUserID))

	userID, err := dbStorage.AddUser(ctx, username, passwordHash)
	require.NoError(t, err)
	assert.Equal(t, expectedUserID, userID)

	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestDBStorage_GetUser(t *testing.T) {
	mockPool, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer mockPool.Close()

	log := zerolog.New(nil)
	dbStorage := storage.NewDBStorage(mockPool, &log)

	ctx := context.Background()
	username := "testuser"
	userId, _ := uuid.NewUUID()
	expectedUser := &models.Login{
		UserId:         userId,
		HashedPassword: "hashedpassword",
	}

	mockPool.ExpectQuery(`SELECT id, password FROM users WHERE username = \$1`).
		WithArgs(username).
		WillReturnRows(pgxmock.NewRows([]string{"id", "password"}).
			AddRow(expectedUser.UserId, expectedUser.HashedPassword))

	user, err := dbStorage.GetUser(ctx, username)
	require.NoError(t, err)
	assert.Equal(t, expectedUser, user)

	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestDBStorage_GetOrder(t *testing.T) {
	mockPool, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer mockPool.Close()

	log := zerolog.New(nil)
	dbStorage := storage.NewDBStorage(mockPool, &log)

	ctx := context.Background()
	orderNum := "order123"
	userId, _ := uuid.NewUUID()
	orderId, _ := uuid.NewUUID()
	expectedOrder := &models.Order{
		Id:        orderId,
		OrderId:   orderNum,
		UserId:    userId,
		Status:    "processed",
		Accrual:   float64Ptr(100),
		CreatedAt: time.Date(2024, time.December, 3, 9, 9, 40, 0, time.UTC),
	}

	mockPool.ExpectQuery(`SELECT id, order_num, user_id, status, amount, created_at::text FROM orders WHERE order_num = \$1`).
		WithArgs(orderNum).
		WillReturnRows(pgxmock.NewRows([]string{"id", "order_num", "user_id", "status", "amount", "created_at::text"}).
			AddRow(expectedOrder.Id, expectedOrder.OrderId, expectedOrder.UserId, expectedOrder.Status, expectedOrder.Accrual, expectedOrder.CreatedAt.Format(time.DateTime)))

	order, err := dbStorage.GetOrder(ctx, orderNum)
	require.NoError(t, err)
	assert.Equal(t, expectedOrder, order)

	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestDBStorage_GetOrders(t *testing.T) {
	mockPool, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer mockPool.Close()

	log := zerolog.New(nil)
	dbStorage := storage.NewDBStorage(mockPool, &log)

	ctx := context.Background()
	userId, _ := uuid.NewUUID()
	orderId1, _ := uuid.NewUUID()
	orderId2, _ := uuid.NewUUID()
	expectedOrders := []models.Order{
		{
			Id:        orderId1,
			OrderId:   "order1",
			UserId:    userId,
			Status:    "processed",
			Accrual:   float64Ptr(100.0),
			CreatedAt: time.Date(2024, time.December, 3, 9, 9, 40, 0, time.UTC),
		},
		{
			Id:        orderId2,
			OrderId:   "order2",
			UserId:    userId,
			Status:    "pending",
			Accrual:   float64Ptr(50.0),
			CreatedAt: time.Date(2024, time.December, 2, 9, 9, 40, 0, time.UTC),
		},
	}

	mockRows := pgxmock.NewRows([]string{"id", "order_num", "user_id", "status", "amount", "created_at::text"})
	for _, order := range expectedOrders {
		mockRows.AddRow(order.Id, order.OrderId, order.UserId, order.Status, order.Accrual, order.CreatedAt.Format(time.DateTime))
	}

	mockPool.ExpectQuery(`SELECT id, order_num, user_id, status, amount, created_at::text FROM orders WHERE user_id = \$1 ORDER BY created_at DESC`).
		WithArgs(userId.String()).
		WillReturnRows(mockRows)

	orders, err := dbStorage.GetOrders(ctx, userId.String())
	require.NoError(t, err)
	assert.Equal(t, expectedOrders, orders)

	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestCreateOrder(t *testing.T) {
	mockPool, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer mockPool.Close()

	log := zerolog.New(nil)
	dbStorage := storage.NewDBStorage(mockPool, &log)

	orderNum := "order123"
	userId := "user123"
	expectedOrderID := "orderID-456"

	mockPool.ExpectQuery(`INSERT INTO orders`).
		WithArgs(userId, orderNum).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(expectedOrderID))

	orderID, err := dbStorage.CreateOrder(context.Background(), orderNum, userId)
	require.NoError(t, err)
	assert.Equal(t, expectedOrderID, orderID)
}

func TestUpdateOrder(t *testing.T) {
	mockPool, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer mockPool.Close()

	log := zerolog.New(nil)
	dbStorage := storage.NewDBStorage(mockPool, &log)

	update := &models.Accrual{
		OrderId: "order123",
		Status:  string(models.Processed),
		Accrual: float64Ptr(50.0),
	}
	userId := "user123"

	mockPool.ExpectBegin()

	key1, key2 := storage.KeyNameAsHash64("update_order")
	// Simulate lock acquisition
	mockPool.ExpectExec("SELECT pg_advisory_xact_lock").WithArgs(key1, key2).
		WillReturnResult(pgxmock.NewResult("SELECT", 0))

	// Simulate fetching current order status
	mockPool.ExpectQuery(`SELECT status FROM orders`).
		WithArgs(update.OrderId, userId).
		WillReturnRows(pgxmock.NewRows([]string{"status"}).AddRow(models.NewStatus))

	// Simulate updating order status and amount
	mockPool.ExpectExec(`UPDATE orders`).
		WithArgs(update.Status, *update.Accrual, update.OrderId).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	// Simulate updating user balance
	mockPool.ExpectExec(`UPDATE users`).
		WithArgs(*update.Accrual, userId).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mockPool.ExpectCommit()

	err = dbStorage.UpdateOrder(context.Background(), update, userId)
	require.NoError(t, err)
}

func TestGetBalance(t *testing.T) {
	mockPool, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer mockPool.Close()

	log := zerolog.New(nil)
	dbStorage := storage.NewDBStorage(mockPool, &log)

	userId := "user123"
	expectedBalance := models.Balance{Balance: 100.0, Withdrawn: 20.0}

	mockPool.ExpectQuery(`SELECT balance, withdrawn FROM users WHERE id = \$1`).
		WithArgs(userId).
		WillReturnRows(pgxmock.NewRows([]string{"balance", "withdrawn"}).AddRow(expectedBalance.Balance, expectedBalance.Withdrawn))

	balance, err := dbStorage.GetBalance(context.Background(), userId)
	require.NoError(t, err)
	assert.Equal(t, &expectedBalance, balance)
}

func TestMakeWithdrawn(t *testing.T) {
	mockPool, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer mockPool.Close()

	log := zerolog.New(nil)
	dbStorage := storage.NewDBStorage(mockPool, &log)

	userId := "user123"
	orderNum := "order123"
	sum := 50.0

	mockPool.ExpectBegin()

	key1, key2 := storage.KeyNameAsHash64("make_withdrawn")
	// Simulate lock acquisition
	mockPool.ExpectExec("SELECT pg_advisory_xact_lock").WithArgs(key1, key2).
		WillReturnResult(pgxmock.NewResult("SELECT", 0))

	// Simulate checking user balance
	mockPool.ExpectQuery(`SELECT balance FROM users WHERE id = \$1`).
		WithArgs(userId).
		WillReturnRows(pgxmock.NewRows([]string{"balance"}).AddRow(100.0))

	// Simulate inserting withdrawal record
	mockPool.ExpectExec(`INSERT INTO withdrawals`).
		WithArgs(userId, sum, orderNum).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	// Simulate updating user balance and withdrawn amount
	mockPool.ExpectExec(`UPDATE users`).
		WithArgs(sum, sum, userId).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mockPool.ExpectCommit()

	err = dbStorage.MakeWithdrawn(context.Background(), userId, orderNum, sum)
	require.NoError(t, err)
}

func TestGetWithdrawals(t *testing.T) {
	mockPool, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer mockPool.Close()

	log := zerolog.New(nil)
	dbStorage := storage.NewDBStorage(mockPool, &log)

	userId := "user123"
	expectedWithdrawals := []models.Withdrawal{
		{OrderId: "order1", Sum: float64Ptr(50.0), CreatedAt: time.Date(2024, time.December, 2, 9, 9, 40, 0, time.UTC)},
		{OrderId: "order2", Sum: float64Ptr(30.0), CreatedAt: time.Date(2024, time.December, 3, 9, 9, 40, 0, time.UTC)},
	}

	rows := pgxmock.NewRows([]string{"order_num", "sum", "updated_at"}).
		AddRow(expectedWithdrawals[0].OrderId, expectedWithdrawals[0].Sum, expectedWithdrawals[0].CreatedAt.Format(time.DateTime)).
		AddRow(expectedWithdrawals[1].OrderId, expectedWithdrawals[1].Sum, expectedWithdrawals[1].CreatedAt.Format(time.DateTime))

	mockPool.ExpectQuery(`SELECT order_num, sum, updated_at::text FROM withdrawals WHERE`).
		WithArgs(userId).
		WillReturnRows(rows)

	withdrawals, err := dbStorage.GetWithdrawals(context.Background(), userId)
	require.NoError(t, err)
	assert.Equal(t, expectedWithdrawals, withdrawals)
}

func float64Ptr(f float64) *float64 {
	return &f
}
