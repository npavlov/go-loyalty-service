package router

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/npavlov/go-loyalty-service/internal/config"
	auth "github.com/npavlov/go-loyalty-service/internal/handlers/auth"
	"github.com/npavlov/go-loyalty-service/internal/handlers/balance"
	health "github.com/npavlov/go-loyalty-service/internal/handlers/health"
	orders "github.com/npavlov/go-loyalty-service/internal/handlers/orders"

	"github.com/redis/go-redis/v9"

	"github.com/npavlov/go-loyalty-service/internal/middlewares"
	"github.com/rs/zerolog"
)

const (
	defaultTimeout = 500 * time.Millisecond // Default timeout for metrics handler
)

type Router interface {
	SetHealthRouter(hh *health.HandlerHealth)
	SetAuthRouter(hh *auth.HandlerAuth)
	SetOrdersRouter(hh *orders.HandlerOrders)
	SetBalanceRouter(hb *balance.HandlerBalance)
	SetMiddlewares()
	GetRouter() *chi.Mux
}

type CustomRouter struct {
	router      *chi.Mux
	logger      *zerolog.Logger
	cfg         *config.Config
	redisClient *redis.Client
}

// NewCustomRouter - constructor for CustomRouter.
func NewCustomRouter(cfg *config.Config, redisClient *redis.Client, l *zerolog.Logger) *CustomRouter {
	return &CustomRouter{
		router:      chi.NewRouter(),
		logger:      l,
		cfg:         cfg,
		redisClient: redisClient,
	}
}

func (cr *CustomRouter) SetMiddlewares() {
	cr.router.Use(middlewares.LoggingMiddleware(cr.logger))
	//cr.router.Use(middlewares.TimeoutMiddleware(defaultTimeout))
	cr.router.Use(middleware.Recoverer)
	cr.router.Use(middlewares.GzipMiddleware)
	cr.router.Use(middlewares.BrotliMiddleware)
	cr.router.Use(middlewares.GzipDecompressionMiddleware)
}

func (cr *CustomRouter) SetHealthRouter(hh *health.HandlerHealth) {
	cr.router.Route("/ping", func(router chi.Router) {
		router.With(middlewares.ContentMiddleware("application/text")).
			Get("/", hh.Ping)
	})
}

func (cr *CustomRouter) SetAuthRouter(ha *auth.HandlerAuth) {
	cr.router.Route("/api/user/register", func(router chi.Router) {
		router.With(middlewares.ContentMiddleware("application/json")).
			Post("/", ha.RegisterHandler)
	})
	cr.router.Route("/api/user/login", func(router chi.Router) {
		router.With(middlewares.ContentMiddleware("application/json")).
			Post("/", ha.LoginHandler)
	})
}

func (cr *CustomRouter) SetOrdersRouter(ho *orders.HandlerOrders) {
	authMiddleware := middlewares.AuthMiddleware(cr.cfg.JwtSecret, cr.redisClient)

	cr.router.Route("/api/user/orders", func(router chi.Router) {
		router.With(middlewares.ContentMiddleware("application/json")).
			With(authMiddleware).
			Get("/", ho.GetOrders)
		router.With(middlewares.ContentMiddleware("application/json")).
			With(authMiddleware).
			Post("/", ho.Create)
	})
}

func (cr *CustomRouter) SetBalanceRouter(hb *balance.HandlerBalance) {
	authMiddleware := middlewares.AuthMiddleware(cr.cfg.JwtSecret, cr.redisClient)
	cr.router.Route("/api/user/", func(router chi.Router) {
		router.With(middlewares.ContentMiddleware("application/json")).
			With(authMiddleware).
			Get("/balance", hb.GetBalance)
		router.With(middlewares.ContentMiddleware("application/json")).
			With(authMiddleware).
			Post("/balance/withdraw", hb.MakeWithdrawal)
		router.With(middlewares.ContentMiddleware("application/json")).
			With(authMiddleware).
			Get("/withdrawals", hb.GetWithdrawals)
	})
}

func (cr *CustomRouter) GetRouter() *chi.Mux {
	return cr.router
}
