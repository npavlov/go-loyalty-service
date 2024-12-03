package models

type MakeWithdrawal struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}
