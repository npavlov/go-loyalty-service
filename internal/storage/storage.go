package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/npavlov/go-loyalty-service/internal/dbmanager"
	"github.com/npavlov/go-loyalty-service/internal/models"
	"github.com/npavlov/go-loyalty-service/internal/utils"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type DBStorage struct {
	log   *zerolog.Logger
	dbCon dbmanager.PgxPool
}

// NewDBStorage initializes a new DBStorage instance.
func NewDBStorage(dbCon dbmanager.PgxPool, log *zerolog.Logger) *DBStorage {
	return &DBStorage{
		dbCon: dbCon,
		log:   log,
	}
}

func (storage *DBStorage) AddUser(ctx context.Context, username string, passwordHash string) (string, error) {
	// Insert the user into the database using Squirrel
	sql, args, err := squirrel.
		Insert("users").
		Columns("username", "password").
		Values(username, passwordHash).
		Suffix("RETURNING id").
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return "", err
	}

	// Execute the query
	var userID string // Use string for UUIDs
	err = storage.dbCon.QueryRow(ctx, sql, args...).Scan(&userID)
	if err != nil {
		return "", err
	}

	return userID, nil
}

func (storage *DBStorage) GetUser(ctx context.Context, username string) (*models.Login, error) {
	var login models.Login

	// Query the database for the user's hashed password and ID
	sql, args, err := squirrel.
		Select("id", "password").
		From("users").
		Where(squirrel.Eq{"username": username}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return nil, err
	}

	// Execute the query and scan the results
	err = storage.dbCon.QueryRow(ctx, sql, args...).Scan(&login.UserId, &login.HashedPassword)
	if err != nil {
		if utils.CheckNoRows(err) {
			return nil, nil
		}

		return nil, err
	}

	return &login, nil
}

func (storage *DBStorage) GetOrder(ctx context.Context, orderNum string) (*models.Order, error) {
	var order models.Order

	// Query the database for the user's hashed password and ID
	sql, args, err := squirrel.
		Select("id", "order_num", "user_id", "status", "amount", "created_at::text").
		From("orders").
		Where(squirrel.Eq{"order_num": orderNum}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return nil, err
	}

	var createdAt string

	// Execute the query and scan the results
	err = storage.dbCon.QueryRow(ctx, sql, args...).
		Scan(&order.Id, &order.OrderId, &order.UserId, &order.Status, &order.Accrual, &createdAt)
	if err != nil {
		if utils.CheckNoRows(err) {
			return nil, nil
		}

		return nil, err
	}

	// Parse created_at
	order.CreatedAt, err = time.Parse(time.DateTime, createdAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse created_at: %w", err)
	}

	return &order, nil
}

func (storage *DBStorage) CreateOrder(ctx context.Context, orderNum string, userId string) (string, error) {
	sql, args, err := squirrel.Insert("orders").
		Columns("user_id", "order_num").
		Values(userId, orderNum).
		Suffix("RETURNING id").
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return "", err
	}

	// Prepare to capture the returned order ID
	var orderID string
	err = storage.dbCon.QueryRow(ctx, sql, args...).Scan(&orderID)

	if err != nil {
		return "", err
	}

	return orderID, nil
}

func (storage *DBStorage) UpdateOrder(ctx context.Context, update *models.Accrual, userId string) error {
	err := WithTx(ctx, storage.dbCon, func(ctx context.Context, tx pgx.Tx) error {
		key1, key2 := KeyNameAsHash64("update_order")
		err := AcquireBlockingLock(ctx, tx, key1, key2, storage.log)
		if err != nil {
			storage.log.Error().Err(err).Msg("failed to acquire lock")

			return errors.Wrap(err, "failed to acquire lock")
		}

		// Initialize an update builder with squirrel
		query := squirrel.Update("orders").
			Set("status", update.Status).
			Where(squirrel.Eq{"order_num": update.OrderId})

		// Conditionally add the amount field if it is provided
		if update.Accrual != nil {
			query = query.Set("amount", *update.Accrual)
		}

		// Build the SQL query
		sql, args, err := query.PlaceholderFormat(squirrel.Dollar).ToSql()
		if err != nil {
			return fmt.Errorf("failed to build update query: %w", err)
		}

		// Execute the update query
		commandTag, err := tx.Exec(ctx, sql, args...)
		if err != nil {
			return fmt.Errorf("failed to execute update query: %w", err)
		}

		// Check if any rows were affected
		if commandTag.RowsAffected() == 0 {
			return fmt.Errorf("no rows updated for order number: %s", update.OrderId)
		}

		if update.Accrual != nil {
			// Increment the user's balance
			balanceQuery := squirrel.Update("users").
				Set("balance", squirrel.Expr("balance + ?", *update.Accrual)).
				Where(squirrel.Eq{"id": userId}).
				PlaceholderFormat(squirrel.Dollar)

			balanceSQL, balanceArgs, err := balanceQuery.ToSql()
			if err != nil {
				return fmt.Errorf("failed to build balance update query: %w", err)
			}

			_, err = tx.Exec(ctx, balanceSQL, balanceArgs...)
			if err != nil {
				return fmt.Errorf("failed to update user balance: %w", err)
			}
		}
		return nil
	})

	return err
}
