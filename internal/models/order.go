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
	Id        uuid.UUID `db:"id"         json:"-"`
	OrderId   string    `db:"order_num"  json:"number"`
	UserId    uuid.UUID `db:"user_id"    json:"-"`
	Status    Status    `db:"status"     json:"status"`
	Accrual   *float64  `db:"amount"     json:"accrual,omitempty"`
	CreatedAt time.Time `db:"updated_at" json:"uploaded_at"`
}

// MarshalJSON ensures CreatedAt is formatted as RFC3339.
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
