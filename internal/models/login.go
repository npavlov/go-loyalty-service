package models

import "github.com/google/uuid"

type Login struct {
	UserID         uuid.UUID
	HashedPassword string
}
