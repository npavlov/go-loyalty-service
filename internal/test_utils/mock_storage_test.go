package testutils_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/npavlov/go-loyalty-service/internal/models"
	testutils "github.com/npavlov/go-loyalty-service/internal/test_utils"
)

func TestMockStorage_AddAndRetrieveUser(t *testing.T) {
	t.Parallel()

	mockStorage := testutils.NewMockStorage()

	username := "testuser"
	passwordHash := "hashedpassword"

	userID, err := mockStorage.AddUser(context.Background(), username, passwordHash)
	require.NoError(t, err, "expected no error when adding a user")

	retrievedUser, exists := mockStorage.GetUser(context.Background(), username)
	assert.True(t, exists, "expected user to exist")
	assert.Equal(t, userID, retrievedUser.UserID.String(), "expected user ID to match")
	assert.Equal(t, passwordHash, retrievedUser.HashedPassword, "expected password hash to match")
}

func TestMockStorage_AddDuplicateUser(t *testing.T) {
	t.Parallel()

	mockStorage := testutils.NewMockStorage()

	username := "testuser"
	passwordHash := "hashedpassword"

	_, err := mockStorage.AddUser(context.Background(), username, passwordHash)
	require.NoError(t, err, "expected no error when adding a user")

	_, err = mockStorage.AddUser(context.Background(), username, passwordHash)
	require.Error(t, err, "expected error when adding a duplicate user")
}

func TestMockStorage_CreateAndRetrieveOrder(t *testing.T) {
	t.Parallel()

	mockStorage := testutils.NewMockStorage()

	userID, _ := uuid.NewUUID()
	orderNum := "order-123"

	_, err := mockStorage.CreateOrder(context.Background(), orderNum, userID.String())
	require.NoError(t, err, "expected no error when creating an order")

	order, exists := mockStorage.GetOrder(context.Background(), orderNum)
	assert.True(t, exists, "expected order to exist")
	assert.Equal(t, userID, order.UserID, "expected user ID to match")
	assert.Equal(t, orderNum, order.OrderID, "expected order number to match")
}

func TestMockStorage_WithdrawAndBalance(t *testing.T) {
	t.Parallel()

	mockStorage := testutils.NewMockStorage()

	userID, _ := uuid.NewUUID()
	orderNum := "order-123"
	sum := 50.0

	mockStorage.Balances[userID.String()] = &models.Balance{Balance: 100.0, Withdrawn: 0}

	err := mockStorage.MakeWithdrawn(context.Background(), userID.String(), orderNum, sum)
	require.NoError(t, err, "expected no error on withdrawal")

	balance, _ := mockStorage.GetBalance(context.Background(), userID.String())
	assert.InDelta(t, 50.0, balance.Balance, 0.001, "expected remaining balance to be correct")
	assert.InDelta(t, 50.0, balance.Withdrawn, 0.001, "expected withdrawn amount to be correct")

	withdrawals, _ := mockStorage.GetWithdrawals(context.Background(), userID.String())
	assert.Len(t, withdrawals, 1, "expected one withdrawal record")
	assert.InDelta(t, sum, *withdrawals[0].Sum, 0.001, "expected withdrawal sum to match")
}
