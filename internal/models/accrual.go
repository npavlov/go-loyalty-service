package models

type Accrual struct {
	OrderId string `json:"order"`
	Status  string `json:"status"`
	Accrual *int   `json:"accrual,omitempty"`
}
