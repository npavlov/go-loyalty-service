package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"

	"github.com/npavlov/go-loyalty-service/internal/config"
	"github.com/npavlov/go-loyalty-service/internal/dbmanager"
	authHandler "github.com/npavlov/go-loyalty-service/internal/handlers/auth"
	balanceHandler "github.com/npavlov/go-loyalty-service/internal/handlers/balance"
	healthHandler "github.com/npavlov/go-loyalty-service/internal/handlers/health"
	ordersHandler "github.com/npavlov/go-loyalty-service/internal/handlers/orders"
	"github.com/npavlov/go-loyalty-service/internal/logger"
	"github.com/npavlov/go-loyalty-service/internal/orders"
	"github.com/npavlov/go-loyalty-service/internal/queue"
	"github.com/npavlov/go-loyalty-service/internal/redis"
	"github.com/npavlov/go-loyalty-service/internal/router"
	"github.com/npavlov/go-loyalty-service/internal/storage"
	"github.com/npavlov/go-loyalty-service/internal/utils"
)

const orderTopic = "orders"

var (
	ErrDatabaseNotConnected = errors.New("database is not connected")
	ErrJWTisNotPorvided     = errors.New("JWT token is not provided")
)

func main() {
	log := logger.NewLogger().SetLogLevel(zerolog.DebugLevel).Get()

	err := godotenv.Load(".env")
	if err != nil {
		log.Error().Err(err).Msg("Error loading server.env file")
	}

	cfg := config.NewConfigBuilder(log).
		FromEnv().
		FromFlags().Build()

	log.Info().Interface("config", cfg).Msg("Configuration loaded")

	if cfg.JwtSecret == "" {
		panic(ErrJWTisNotPorvided)
	}

	ctx, cancel := utils.WithSignalCancel(context.Background(), log)

	dbManager := dbmanager.NewDBManager(cfg.Database, log).Connect(ctx).ApplyMigrations()
	defer dbManager.Close()
	if dbManager.DB == nil {
		panic(ErrDatabaseNotConnected)
	}

	st := storage.NewDBStorage(dbManager.DB, log)

	memStorage := redis.NewRStorage(*cfg)

	if err := memStorage.Ping(ctx); err != nil {
		log.Error().Err(err).Msg("Error connecting to redis")
	}

	kafkaQueue := queue.NewQueue(cfg, log)
	// Kafka Producers
	orderWriter, orderReader, closeOrder := kafkaQueue.CreateGroup(orderTopic)
	defer closeOrder()
	ordersProcessor := orders.NewOrders(orderWriter, orderReader, log).
		WithSender(cfg).WithStorage(st)
	go ordersProcessor.ProcessOrders(ctx)

	hHandlers := healthHandler.NewHealthHandler(dbManager, log)
	aHandlers := authHandler.NewAuthHandler(st, cfg, memStorage, log)
	oHandlers := ordersHandler.NewOrdersHandler(st, ordersProcessor, log)
	bHandlers := balanceHandler.NewBalanceHandler(st, log)

	var cRouter router.Router = router.NewCustomRouter(cfg, memStorage, log)
	cRouter.SetMiddlewares()
	cRouter.SetHealthRouter(hHandlers)
	cRouter.SetAuthRouter(aHandlers)
	cRouter.SetOrdersRouter(oHandlers)
	cRouter.SetBalanceRouter(bHandlers)

	log.Info().
		Str("server_address", cfg.Address).
		Msg("Server started")

	//nolint:exhaustruct
	server := &http.Server{
		Addr:         cfg.Address,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
		Handler:      cRouter.GetRouter(),
	}

	go func() {
		// Wait for the context to be done (i.e., signal received)
		<-ctx.Done()

		if err := server.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("Error shutting down server")
		}
	}()

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error().Err(err).Msg("Error starting server")
		cancel()
	}

	log.Info().Msg("Server shut down")
}
