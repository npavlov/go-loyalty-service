package models

import (
	"encoding/json"
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
	Accrual   *float64  `json:"accrual,omitempty" db:"amount"`
	CreatedAt time.Time `json:"uploaded_at" db:"updated_at"`
}

// MarshalJSON ensures CreatedAt is formatted as RFC3339
func (o Order) MarshalJSON() ([]byte, error) {
	// Create an alias to avoid infinite recursion
	type Alias Order
	return json.Marshal(&struct {
		UploadedAt string `json:"uploaded_at"`
		*Alias
	}{
		Alias:      (*Alias)(&o),
		UploadedAt: o.CreatedAt.Format(time.RFC3339),
	})
}
