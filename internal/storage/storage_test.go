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

const (
	userID   = "user123"
	orderNum = "order123"
)

func TestDBStorage_AddUser(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	mockPool, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer mockPool.Close()

	log := zerolog.New(nil)
	dbStorage := storage.NewDBStorage(mockPool, &log)

	ctx := context.Background()
	username := "testuser"
	userID, _ := uuid.NewUUID()
	expectedUser := &models.Login{
		UserID:         userID,
		HashedPassword: "hashedpassword",
	}

	mockPool.ExpectQuery(`SELECT id, password FROM users WHERE username = \$1`).
		WithArgs(username).
		WillReturnRows(pgxmock.NewRows([]string{"id", "password"}).
			AddRow(expectedUser.UserID, expectedUser.HashedPassword))

	user, found := dbStorage.GetUser(ctx, username)
	require.True(t, found)
	assert.Equal(t, expectedUser, user)

	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestDBStorage_GetOrder(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer mockPool.Close()

	log := zerolog.New(nil)
	dbStorage := storage.NewDBStorage(mockPool, &log)

	ctx := context.Background()

	userID, _ := uuid.NewUUID()
	orderID, _ := uuid.NewUUID()
	expOrd := &models.Order{
		ID:        orderID,
		OrderID:   orderNum,
		UserID:    userID,
		Status:    "processed",
		Accrual:   float64Ptr(100),
		CreatedAt: time.Date(2024, time.December, 3, 9, 9, 40, 0, time.UTC),
	}

	createdAt := expOrd.CreatedAt.Format(time.DateTime)
	mockPool.ExpectQuery(`SELECT id, order_num, user_id, status, amount, created_at::text 
FROM orders WHERE order_num = \$1`).
		WithArgs(orderNum).
		WillReturnRows(pgxmock.NewRows([]string{"id", "order_num", "user_id", "status", "amount", "created_at::text"}).
			AddRow(expOrd.ID, expOrd.OrderID, expOrd.UserID, expOrd.Status, expOrd.Accrual, createdAt))

	order, found := dbStorage.GetOrder(ctx, orderNum)
	require.True(t, found)
	assert.Equal(t, expOrd, order)

	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestDBStorage_GetOrders(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer mockPool.Close()

	log := zerolog.New(nil)
	dbStorage := storage.NewDBStorage(mockPool, &log)

	ctx := context.Background()
	userID, _ := uuid.NewUUID()
	orderID1, _ := uuid.NewUUID()
	orderID2, _ := uuid.NewUUID()
	expectedOrders := []models.Order{
		{
			ID:        orderID1,
			OrderID:   "order1",
			UserID:    userID,
			Status:    "processed",
			Accrual:   float64Ptr(100.0),
			CreatedAt: time.Date(2024, time.December, 3, 9, 9, 40, 0, time.UTC),
		},
		{
			ID:        orderID2,
			OrderID:   "order2",
			UserID:    userID,
			Status:    "pending",
			Accrual:   float64Ptr(50.0),
			CreatedAt: time.Date(2024, time.December, 2, 9, 9, 40, 0, time.UTC),
		},
	}

	mockRows := pgxmock.NewRows([]string{"id", "order_num", "user_id", "status", "amount", "created_at::text"})
	for _, or := range expectedOrders {
		mockRows.AddRow(or.ID, or.OrderID, or.UserID, or.Status, or.Accrual, or.CreatedAt.Format(time.DateTime))
	}

	mockPool.ExpectQuery(`SELECT id, order_num, user_id, status, amount, created_at::text 
FROM orders WHERE user_id = \$1 ORDER BY created_at DESC`).
		WithArgs(userID.String()).
		WillReturnRows(mockRows)

	orders, err := dbStorage.GetOrders(ctx, userID.String())
	require.NoError(t, err)
	assert.Equal(t, expectedOrders, orders)

	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestCreateOrder(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer mockPool.Close()

	log := zerolog.New(nil)
	dbStorage := storage.NewDBStorage(mockPool, &log)

	expectedOrderID := "orderID-456"

	mockPool.ExpectQuery(`INSERT INTO orders`).
		WithArgs(userID, orderNum).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(expectedOrderID))

	orderID, err := dbStorage.CreateOrder(context.Background(), orderNum, userID)
	require.NoError(t, err)
	assert.Equal(t, expectedOrderID, orderID)
}

func TestUpdateOrder(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer mockPool.Close()

	log := zerolog.New(nil)
	dbStorage := storage.NewDBStorage(mockPool, &log)

	update := &models.Accrual{
		OrderID: "order123",
		Status:  string(models.Processed),
		Accrual: float64Ptr(50.0),
	}

	mockPool.ExpectBegin()

	key1, key2 := storage.KeyNameAsHash64("update_order")
	// Simulate lock acquisition
	mockPool.ExpectExec("SELECT pg_advisory_xact_lock").WithArgs(key1, key2).
		WillReturnResult(pgxmock.NewResult("SELECT", 0))

	// Simulate fetching current order status
	mockPool.ExpectQuery(`SELECT status FROM orders`).
		WithArgs(update.OrderID, userID).
		WillReturnRows(pgxmock.NewRows([]string{"status"}).AddRow(models.NewStatus))

	// Simulate updating order status and amount
	mockPool.ExpectExec(`UPDATE orders`).
		WithArgs(update.Status, *update.Accrual, update.OrderID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	// Simulate updating user balance
	mockPool.ExpectExec(`UPDATE users`).
		WithArgs(*update.Accrual, userID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mockPool.ExpectCommit()

	err = dbStorage.UpdateOrder(context.Background(), update, userID)
	require.NoError(t, err)
}

func TestGetBalance(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer mockPool.Close()

	log := zerolog.New(nil)
	dbStorage := storage.NewDBStorage(mockPool, &log)

	expectedBalance := models.Balance{Balance: 100.0, Withdrawn: 20.0}

	mockPool.ExpectQuery(`SELECT balance, withdrawn FROM users WHERE id = \$1`).
		WithArgs(userID).
		WillReturnRows(pgxmock.NewRows([]string{"balance", "withdrawn"}).
			AddRow(expectedBalance.Balance, expectedBalance.Withdrawn))

	balance, err := dbStorage.GetBalance(context.Background(), userID)
	require.NoError(t, err)
	assert.Equal(t, &expectedBalance, balance)
}

func TestMakeWithdrawn(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer mockPool.Close()

	log := zerolog.New(nil)
	dbStorage := storage.NewDBStorage(mockPool, &log)

	sum := 50.0

	mockPool.ExpectBegin()

	key1, key2 := storage.KeyNameAsHash64("make_withdrawn")
	// Simulate lock acquisition
	mockPool.ExpectExec("SELECT pg_advisory_xact_lock").WithArgs(key1, key2).
		WillReturnResult(pgxmock.NewResult("SELECT", 0))

	// Simulate checking user balance
	mockPool.ExpectQuery(`SELECT balance FROM users WHERE id = \$1`).
		WithArgs(userID).
		WillReturnRows(pgxmock.NewRows([]string{"balance"}).AddRow(100.0))

	// Simulate inserting withdrawal record
	mockPool.ExpectExec(`INSERT INTO withdrawals`).
		WithArgs(userID, sum, orderNum).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	// Simulate updating user balance and withdrawn amount
	mockPool.ExpectExec(`UPDATE users`).
		WithArgs(sum, sum, userID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mockPool.ExpectCommit()

	err = dbStorage.MakeWithdrawn(context.Background(), userID, orderNum, sum)
	require.NoError(t, err)
}

func TestGetWithdrawals(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer mockPool.Close()

	log := zerolog.New(nil)
	dbStorage := storage.NewDBStorage(mockPool, &log)

	expecWithdraw := []models.Withdrawal{
		{
			UserID:    "",
			OrderID:   "order1",
			Sum:       float64Ptr(50.0),
			CreatedAt: time.Date(2024, time.December, 2, 9, 9, 40, 0, time.UTC),
		},
		{
			UserID:    "",
			OrderID:   "order2",
			Sum:       float64Ptr(30.0),
			CreatedAt: time.Date(2024, time.December, 3, 9, 9, 40, 0, time.UTC),
		},
	}

	rows := pgxmock.NewRows([]string{"order_num", "sum", "updated_at"}).
		AddRow(expecWithdraw[0].OrderID, expecWithdraw[0].Sum, expecWithdraw[0].CreatedAt.Format(time.DateTime)).
		AddRow(expecWithdraw[1].OrderID, expecWithdraw[1].Sum, expecWithdraw[1].CreatedAt.Format(time.DateTime))

	mockPool.ExpectQuery(`SELECT order_num, sum, updated_at::text FROM withdrawals WHERE`).
		WithArgs(userID).
		WillReturnRows(rows)

	withdrawals, err := dbStorage.GetWithdrawals(context.Background(), userID)
	require.NoError(t, err)
	assert.Equal(t, expecWithdraw, withdrawals)
}

func float64Ptr(f float64) *float64 {
	return &f
}
