package utils

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func CheckPGConstraint(err error) bool {
	// Handle specific PostgreSQL errors
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// Check for unique constraint violation (SQLSTATE 23505)
		if pgErr.Code == "23505" {
			return true
		}
	}

	return false
}

func CheckNoRows(err error) bool {
	if errors.Is(err, pgx.ErrNoRows) {
		return true
	}

	return false
}
