package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/sdk/trace"

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
	"github.com/npavlov/go-loyalty-service/internal/tracer"
	"github.com/npavlov/go-loyalty-service/internal/utils"
)

const orderTopic = "orders"

var (
	ErrDatabaseNotConnected = errors.New("database is not connected")
	ErrJWTisNotPorvided     = errors.New("JWT token is not provided")
)

func main() {
	log := setupLogger()

	cfg := loadConfig(log)

	ctx, cancel := utils.WithSignalCancel(context.Background(), log)
	defer cancel()

	tp := initializeTracer(ctx, cfg, log)
	defer shutdownTracer(ctx, tp, log)

	dbManager := setupDatabase(ctx, cfg, log)
	defer dbManager.Close()

	st, memStorage := setupStorage(ctx, cfg, dbManager, log)

	kafkaQueue := queue.NewQueue(cfg, log)
	orderWriter, orderReader, closeOrder := kafkaQueue.CreateGroup(orderTopic)
	defer closeOrder()

	ordersProcessor := orders.NewOrders(orderWriter, orderReader, log).WithSender(cfg).WithStorage(st)
	go ordersProcessor.ProcessOrders(ctx)

	cRouter := setupRouter(cfg, memStorage, dbManager, st, ordersProcessor, log)

	startServer(ctx, cfg, cRouter, log)
}

func setupLogger() *zerolog.Logger {
	log := logger.NewLogger().SetLogLevel(zerolog.DebugLevel).Get()

	return log
}

func loadConfig(log *zerolog.Logger) *config.Config {
	if err := godotenv.Load(".env"); err != nil {
		log.Error().Err(err).Msg("Error loading .env file")
	}

	cfg := config.NewConfigBuilder(log).FromEnv().FromFlags().Build()
	log.Info().Interface("config", cfg).Msg("Configuration loaded")

	if cfg.JwtSecret == "" {
		panic(ErrJWTisNotPorvided)
	}

	return cfg
}

func initializeTracer(ctx context.Context, cfg *config.Config, log *zerolog.Logger) *trace.TracerProvider {
	tp, err := tracer.NewTracer(cfg).InitTracer(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize tracer")
	}

	return tp
}

func shutdownTracer(ctx context.Context, tp *trace.TracerProvider, log *zerolog.Logger) {
	if err := tp.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Error shutting down tracer provider")
	}
}

func setupDatabase(ctx context.Context, cfg *config.Config, log *zerolog.Logger) *dbmanager.DBManager {
	dbManager := dbmanager.NewDBManager(cfg.Database, log).Connect(ctx).ApplyMigrations()
	if dbManager.DB == nil {
		log.Fatal().Err(ErrDatabaseNotConnected).Msg("Database is not connected")
	}

	return dbManager
}

func setupStorage(
	ctx context.Context,
	cfg *config.Config,
	dbManager *dbmanager.DBManager,
	log *zerolog.Logger,
) (*storage.DBStorage, *redis.RStorage) {
	st := storage.NewDBStorage(dbManager.DB, log)
	memStorage := redis.NewRStorage(*cfg, log)

	if err := memStorage.Ping(ctx); err != nil {
		log.Error().Err(err).Msg("Error connecting to redis")
	}

	return st, memStorage
}

func setupRouter(
	cfg *config.Config,
	memStorage *redis.RStorage,
	dbManager *dbmanager.DBManager,
	st *storage.DBStorage,
	ordersProcessor *orders.Orders,
	log *zerolog.Logger,
) *router.CustomRouter {
	hHandlers := healthHandler.NewHealthHandler(dbManager, log)
	aHandlers := authHandler.NewAuthHandler(st, cfg, memStorage, log)
	oHandlers := ordersHandler.NewOrdersHandler(st, ordersProcessor, log)
	bHandlers := balanceHandler.NewBalanceHandler(st, log)

	cRouter := router.NewCustomRouter(cfg, memStorage, log)
	cRouter.SetMiddlewares()
	cRouter.SetHealthRouter(hHandlers)
	cRouter.SetAuthRouter(aHandlers)
	cRouter.SetOrdersRouter(oHandlers)
	cRouter.SetBalanceRouter(bHandlers)

	return cRouter
}

func startServer(ctx context.Context, cfg *config.Config, cRouter router.Router, log *zerolog.Logger) {
	log.Info().Str("server_address", cfg.Address).Msg("Server started")

	otelRouter := otelhttp.NewHandler(cRouter.GetRouter(), "HTTP Router")

	//nolint:exhaustruct
	server := &http.Server{
		Addr:         cfg.Address,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
		Handler:      otelRouter,
	}

	go func() {
		<-ctx.Done()
		if err := server.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("Error shutting down server")
		}
	}()

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error().Err(err).Msg("Error starting server")
	}
	log.Info().Msg("Server shut down")
}
