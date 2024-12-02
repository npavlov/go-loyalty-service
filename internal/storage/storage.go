package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/npavlov/go-loyalty-service/internal/dbmanager"
	"github.com/npavlov/go-loyalty-service/internal/models"
	"github.com/npavlov/go-loyalty-service/internal/utils"
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

func (storage *DBStorage) GetOrders(ctx context.Context, userID string) ([]models.Order, error) {
	// Prepare the query using Squirrel
	sql, args, err := squirrel.
		Select("id", "order_num", "user_id", "status", "amount", "created_at::text").
		From("orders").
		Where(squirrel.Eq{"user_id": userID}).
		OrderBy("created_at DESC"). // Order by created_at in descending order
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	// Execute the query
	rows, err := storage.dbCon.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	var orders []models.Order

	// Iterate over rows and populate the orders slice
	for rows.Next() {
		var order models.Order
		var createdAt string

		err := rows.Scan(&order.Id, &order.OrderId, &order.UserId, &order.Status, &order.Accrual, &createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Parse created_at
		order.CreatedAt, err = time.Parse(time.DateTime, createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse created_at: %w", err)
		}

		orders = append(orders, order)
	}

	// Check if any errors occurred during row iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	// Return an empty array if no orders are found
	if len(orders) == 0 {
		return []models.Order{}, nil
	}

	return orders, nil
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

		var currentStatus models.Status

		// Query the database for the user's hashed password and ID
		sql, args, err := squirrel.
			Select("status").
			From("orders").
			Where(squirrel.Eq{"order_num": update.OrderId}).
			Where(squirrel.Eq{"user_id": userId}).
			PlaceholderFormat(squirrel.Dollar).
			ToSql()
		if err != nil {
			return err
		}

		// Execute the query and scan the results
		err = storage.dbCon.QueryRow(ctx, sql, args...).
			Scan(&currentStatus)
		if err != nil {
			return errors.New("Order does not exist, failed to update")
		}

		if currentStatus == models.Processed || currentStatus == models.Invalid {
			storage.log.Info().Msg("order is already processed")

			return nil
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
		sql, args, err = query.PlaceholderFormat(squirrel.Dollar).ToSql()
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

func (storage *DBStorage) GetBalance(ctx context.Context, userID string) (*models.Balance, error) {
	var balance models.Balance

	// Query the database for the user's balance and withdrawn amounts
	sql, args, err := squirrel.
		Select("balance", "withdrawn").
		From("users").
		Where(squirrel.Eq{"id": userID}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return nil, err
	}

	// Execute the query and scan the results
	err = storage.dbCon.QueryRow(ctx, sql, args...).Scan(&balance.Balance, &balance.Withdrawn)
	if err != nil {
		return nil, err
	}

	return &balance, nil
}

func (storage *DBStorage) MakeWithdrawn(ctx context.Context, userId string, orderNum string, sum float64) error {
	err := WithTx(ctx, storage.dbCon, func(ctx context.Context, tx pgx.Tx) error {
		key1, key2 := KeyNameAsHash64("make_withdrawn")
		err := AcquireBlockingLock(ctx, tx, key1, key2, storage.log)
		if err != nil {
			storage.log.Error().Err(err).Msg("failed to acquire lock")
			return errors.Wrap(err, "failed to acquire lock")
		}

		// Check if the user has enough balance
		var currentBalance float64
		sql, args, err := squirrel.
			Select("balance").
			From("users").
			Where(squirrel.Eq{"id": userId}).
			PlaceholderFormat(squirrel.Dollar).
			ToSql()
		if err != nil {
			return fmt.Errorf("failed to build balance query: %w", err)
		}

		err = tx.QueryRow(ctx, sql, args...).Scan(&currentBalance)
		if err != nil {
			return fmt.Errorf("failed to fetch current balance: %w", err)
		}

		if currentBalance < sum {
			storage.log.Error().Msgf("insufficient balance: available %.2f, required %.2f", currentBalance, sum)

			return fmt.Errorf("insufficient balance: available %.2f, required %.2f", currentBalance, sum)
		}

		// Insert a new withdrawal record
		withdrawalSQL, withdrawalArgs, err := squirrel.
			Insert("withdrawals").
			Columns("user_id", "sum", "order_num").
			Values(userId, sum, orderNum).
			PlaceholderFormat(squirrel.Dollar).
			ToSql()
		if err != nil {
			return fmt.Errorf("failed to build withdrawal insert query: %w", err)
		}

		_, err = tx.Exec(ctx, withdrawalSQL, withdrawalArgs...)
		if err != nil {
			return fmt.Errorf("failed to insert withdrawal record: %w", err)
		}

		// Update the user's balance and withdrawn amount
		updateSQL, updateArgs, err := squirrel.
			Update("users").
			Set("balance", squirrel.Expr("balance - ?", sum)).
			Set("withdrawn", squirrel.Expr("withdrawn + ?", sum)).
			Where(squirrel.Eq{"id": userId}).
			PlaceholderFormat(squirrel.Dollar).
			ToSql()
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, updateSQL, updateArgs...)
		if err != nil {
			return err
		}

		return nil
	})

	return err
}

func (storage *DBStorage) GetWithdrawal(ctx context.Context, orderNum string) (*models.Withdrawal, error) {
	sql, args, err := squirrel.
		Select("order_num", "sum", "updated_at::text").
		From("withdrawals").
		Where(squirrel.Eq{"order_num": orderNum}).
		OrderBy("created_at DESC").
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := storage.dbCon.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	var withdrawal models.Withdrawal

	var createdAt string
	err = storage.dbCon.QueryRow(ctx, sql, args...).
		Scan(&withdrawal.OrderId, &withdrawal.Sum, &createdAt)
	if err != nil {
		if utils.CheckNoRows(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to scan row: %w", err)
	}

	// Parse created_at
	withdrawal.CreatedAt, err = time.Parse(time.DateTime, createdAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse created_at: %w", err)
	}

	return &withdrawal, nil
}

func (storage *DBStorage) GetWithdrawals(ctx context.Context, userId string) ([]models.Withdrawal, error) {
	sql, args, err := squirrel.
		Select("order_num", "sum", "updated_at::text").
		From("withdrawals").
		Where(squirrel.Eq{"user_id": userId}).
		OrderBy("created_at DESC").
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := storage.dbCon.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	var withdrawals []models.Withdrawal

	for rows.Next() {
		var wd models.Withdrawal

		var createdAt string
		err := rows.Scan(&wd.OrderId, &wd.Sum, &createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Parse created_at
		wd.CreatedAt, err = time.Parse(time.DateTime, createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse created_at: %w", err)
		}

		withdrawals = append(withdrawals, wd)
	}

	return withdrawals, nil
}
