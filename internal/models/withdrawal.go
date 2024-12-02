package models

import (
	"encoding/json"
	"time"
)

type Withdrawal struct {
	OrderId   string    `db:"order_num"  json:"order"`
	Sum       *float64  `db:"amount"     json:"sum"`
	CreatedAt time.Time `db:"updated_at" json:"processed_at"`
	UserId    string    `db:"user_id"    json:"-"`
}

// MarshalJSON ensures CreatedAt is formatted as RFC3339.
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
