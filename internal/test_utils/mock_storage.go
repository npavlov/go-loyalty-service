package testutils

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"

	"github.com/npavlov/go-loyalty-service/internal/models"
)

type MockStorage struct {
	mu          sync.Mutex
	Users       map[string]*models.Login
	orders      map[string]*models.Order
	Withdrawals map[string]*models.Withdrawal
	Balances    map[string]*models.Balance
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		Users:       make(map[string]*models.Login),
		orders:      make(map[string]*models.Order),
		Withdrawals: make(map[string]*models.Withdrawal),
		Balances:    make(map[string]*models.Balance),
		mu:          sync.Mutex{},
	}
}

func (m *MockStorage) AddUser(_ context.Context, username string, passwordHash string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.Users[username]; exists {
		//nolint:exhaustruct
		pgErr := pgconn.PgError{
			Code:    "23505",
			Message: "User with username already exists",
		}

		return "", &pgErr
	}

	userID, _ := uuid.NewUUID()
	m.Users[username] = &models.Login{
		UserId:         userID,
		HashedPassword: passwordHash,
	}

	return userID.String(), nil
}

func (m *MockStorage) GetUser(_ context.Context, username string) (*models.Login, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.Users[username]
	if !exists {
		return nil, false
	}

	return user, true
}

func (m *MockStorage) GetOrder(_ context.Context, orderNum string) (*models.Order, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	order, exists := m.orders[orderNum]
	if !exists {
		return nil, false
	}

	return order, true
}

func (m *MockStorage) GetOrders(_ context.Context, userID string) ([]models.Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var orders []models.Order
	for _, order := range m.orders {
		if order.UserId.String() == userID {
			orders = append(orders, *order)
		}
	}

	return orders, nil
}

func (m *MockStorage) CreateOrder(_ context.Context, orderNum string, userId string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.orders[orderNum]; exists {
		return "", errors.New("order already exists")
	}

	parsedUserID, err := uuid.Parse(userId)
	if err != nil {
		return "", errors.Wrap(err, "parsing user id")
	}

	orderID, _ := uuid.NewUUID()
	m.orders[orderNum] = &models.Order{
		Id:        orderID,
		OrderId:   orderNum,
		UserId:    parsedUserID,
		Status:    models.NewStatus,
		Accrual:   float64Ptr(0),
		CreatedAt: time.Now(),
	}

	return orderID.String(), nil
}

func (m *MockStorage) UpdateOrder(_ context.Context, update *models.Accrual, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	order, exists := m.orders[update.OrderId]
	if !exists || order.UserId.String() != userID {
		return errors.New("order not found or unauthorized access")
	}

	order.Status = models.Status(update.Status)
	if update.Accrual != nil {
		order.Accrual = update.Accrual
	}
	m.orders[update.OrderId] = order

	return nil
}

func (m *MockStorage) GetBalance(_ context.Context, userID string) (*models.Balance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	balance, exists := m.Balances[userID]
	if !exists {
		return &models.Balance{
			Balance:   0,
			Withdrawn: 0,
		}, nil
	}

	return balance, nil
}

func (m *MockStorage) MakeWithdrawn(_ context.Context, userId string, orderNum string, sum float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	balance, exists := m.Balances[userId]
	if !exists || balance.Balance < sum {
		return errors.New("insufficient balance")
	}

	// Deduct the balance and add a withdrawal record
	balance.Balance -= sum
	balance.Withdrawn += sum

	m.Withdrawals[orderNum] = &models.Withdrawal{
		OrderId:   orderNum,
		Sum:       float64Ptr(sum),
		CreatedAt: time.Now(),
		UserId:    userId,
	}

	return nil
}

func (m *MockStorage) GetWithdrawals(_ context.Context, userId string) ([]models.Withdrawal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var withdrawals []models.Withdrawal
	for _, wd := range m.Withdrawals {
		if wd.UserId == userId {
			withdrawals = append(withdrawals, *wd)
		}
	}

	return withdrawals, nil
}

func float64Ptr(f float64) *float64 {
	return &f
}
