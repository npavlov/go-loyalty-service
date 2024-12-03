package storage

import (
	"context"

	"github.com/npavlov/go-loyalty-service/internal/models"
)

type Storage interface {
	AddUser(ctx context.Context, username string, passwordHash string) (string, error)
	GetUser(ctx context.Context, username string) (*models.Login, bool)
	GetOrder(ctx context.Context, orderNum string) (*models.Order, bool)
	GetOrders(ctx context.Context, userID string) ([]models.Order, error)
	CreateOrder(ctx context.Context, orderNum string, userID string) (string, error)
	UpdateOrder(ctx context.Context, update *models.Accrual, userID string) error
	GetBalance(ctx context.Context, userID string) (*models.Balance, error)
	MakeWithdrawn(ctx context.Context, userID string, orderNum string, sum float64) error
	GetWithdrawals(ctx context.Context, userID string) ([]models.Withdrawal, error)
}
