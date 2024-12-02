package models

import "github.com/google/uuid"

type Login struct {
	UserId         uuid.UUID
	HashedPassword string
}
