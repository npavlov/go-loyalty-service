package orders

import (
	"context"
)

type QueueProcessor interface {
	AddOrder(ctx context.Context, orderNum string, userId string) error
	ProcessOrders(ctx context.Context)
}
