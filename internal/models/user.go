package models

type User struct {
	Login    string `json:"login"    validate:"required"`
	Password string `json:"password" validate:"required"`
}
