package models

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	Processed  Status = "PROCESSED"
	Processing Status = "PROCESSING"
	Invalid    Status = "INVALID"
	NewStatus  Status = "NEW"
)

type Order struct {
	Id        uuid.UUID `json:"-" db:"id"`
	OrderId   string    `json:"number" db:"order_num"`
	UserId    uuid.UUID `json:"-" db:"user_id"`
	Status    Status    `json:"status" db:"status"`
	Accrual   int       `json:"accrual,omitempty" db:"amount"`
	CreatedAt time.Time `json:"uploaded_at" db:"updated_at"`
}
