package models

import (
	"encoding/json"
	"time"
)

type Withdrawal struct {
	OrderId   string    `json:"order" db:"order_num"`
	Sum       *float64  `json:"sum" db:"amount"`
	CreatedAt time.Time `json:"processed_at" db:"updated_at"`
}

// MarshalJSON ensures CreatedAt is formatted as RFC3339
func (w Withdrawal) MarshalJSON() ([]byte, error) {
	// Create an alias to avoid infinite recursion
	type Alias Withdrawal
	return json.Marshal(&struct {
		UploadedAt string `json:"processed_at"`
		*Alias
	}{
		Alias:      (*Alias)(&w),
		UploadedAt: w.CreatedAt.Format(time.RFC3339),
	})
}
