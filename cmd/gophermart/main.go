package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	authHandler "github.com/npavlov/go-loyalty-service/internal/handlers/auth"
	ordersHandler "github.com/npavlov/go-loyalty-service/internal/handlers/orders"
	"github.com/npavlov/go-loyalty-service/internal/orders"
	"github.com/npavlov/go-loyalty-service/internal/queue"
	"github.com/npavlov/go-loyalty-service/internal/storage"
	"github.com/redis/go-redis/v9"

	"github.com/joho/godotenv"
	"github.com/npavlov/go-loyalty-service/internal/config"
	"github.com/npavlov/go-loyalty-service/internal/dbmanager"
	healthHandler "github.com/npavlov/go-loyalty-service/internal/handlers/health"
	"github.com/npavlov/go-loyalty-service/internal/logger"
	"github.com/npavlov/go-loyalty-service/internal/router"
	"github.com/npavlov/go-loyalty-service/internal/utils"
	"github.com/rs/zerolog"
)

var (
	orderTopic      = "orders"
	withdrawalTopic = "withdrawals"
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
		panic(errors.New("jwt secret not provided"))
	}

	ctx, cancel := utils.WithSignalCancel(context.Background(), log)

	dbManager := dbmanager.NewDBManager(cfg.Database, log).Connect(ctx).ApplyMigrations()
	defer dbManager.Close()
	if err != nil {
		log.Error().Err(err).Msg("Error initialising db manager")
	}

	st := storage.NewDBStorage(dbManager.DB, log)

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis, // use default Addr
		Password: "",        // no password set
		DB:       0,         // use default DB
	})

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatal().Err(err).Msg("Error connecting to redis")
	}

	kafkaQueue := queue.NewQueue(cfg)
	// Kafka Producers
	orderWriter, orderReader, closeOrder := kafkaQueue.CreateGroup(orderTopic)
	defer closeOrder()
	ordersProcessor := orders.NewOrders(orderWriter, orderReader, log).
		WithSender(cfg).WithStorage(st)
	go ordersProcessor.ProcessOrders(ctx)

	//withdrawalWriter, withdrawalReader, closeWithdrawal := kafkaQueue.CreateGroup(withdrawalTopic)
	//defer closeWithdrawal()

	hHandlers := healthHandler.NewHealthHandler(dbManager, log)
	aHandlers := authHandler.NewAuthHandler(st, cfg, redisClient, log)
	oHandlers := ordersHandler.NewOrdersHandler(st, ordersProcessor, log)

	var cRouter router.Router = router.NewCustomRouter(cfg, redisClient, log)
	cRouter.SetMiddlewares()
	cRouter.SetHealthRouter(hHandlers)
	cRouter.SetAuthRouter(aHandlers)
	cRouter.SetOrdersRouter(oHandlers)

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
