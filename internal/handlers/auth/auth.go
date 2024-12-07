package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/crypto/bcrypt"

	"github.com/npavlov/go-loyalty-service/internal/config"
	"github.com/npavlov/go-loyalty-service/internal/logger"
	"github.com/npavlov/go-loyalty-service/internal/models"
	"github.com/npavlov/go-loyalty-service/internal/redis"
	"github.com/npavlov/go-loyalty-service/internal/storage"
	"github.com/npavlov/go-loyalty-service/internal/utils"
)

type HandlerAuth struct {
	logger     *zerolog.Logger
	storage    storage.Storage
	cfg        *config.Config
	validate   *validator.Validate
	memStorage redis.MemStorage
	tracer     trace.Tracer
}

const (
	tokenExpiration = time.Minute * 60
)

// NewAuthHandler initializes the AuthHandler with tracing.
func NewAuthHandler(st storage.Storage, cfg *config.Config, memSt redis.MemStorage, l *zerolog.Logger) *HandlerAuth {
	tracer := otel.Tracer("auth-handlers")

	return &HandlerAuth{
		logger:     l,
		storage:    st,
		cfg:        cfg,
		validate:   validator.New(),
		memStorage: memSt,
		tracer:     tracer,
	}
}

// RegisterHandler processes user registration.
//
//nolint:funlen
func (ah *HandlerAuth) RegisterHandler(writer http.ResponseWriter, request *http.Request) {
	ctx, span := ah.tracer.Start(request.Context(), "RegisterHandler")
	defer span.End()

	log := logger.GetWithTrace(ctx, ah.logger)

	var req models.User
	if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
		log.Error().Err(err).Msg("error decoding body")
		http.Error(writer, "Invalid request", http.StatusBadRequest)
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid request body")

		return
	}

	span.SetAttributes(attribute.String("user.login", req.Login))

	if err := ah.validate.Struct(req); err != nil {
		log.Error().Err(err).Msg("error validating body")
		http.Error(writer, "Invalid input: "+err.Error(), http.StatusBadRequest)
		span.RecordError(err)
		span.SetStatus(codes.Error, "Validation failed")

		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Error().Err(err).Msg("error encrypting password")
		http.Error(writer, "Error creating user", http.StatusInternalServerError)
		span.RecordError(err)
		span.SetStatus(codes.Error, "Password encryption failed")

		return
	}

	userID, err := ah.storage.AddUser(ctx, req.Login, string(hashedPassword))
	if result := utils.CheckPGConstraint(err); err != nil && result {
		http.Error(writer, "Username already exists", http.StatusConflict)
		span.SetStatus(codes.Error, "Username conflict")

		return
	}

	if userID == "" || err != nil {
		log.Error().Msg("Error saving user")
		http.Error(writer, "Error saving user", http.StatusInternalServerError)
		span.RecordError(err)
		span.SetStatus(codes.Error, "Database error")

		return
	}

	span.SetAttributes(attribute.String("user.id", userID))

	token, err := ah.generateJWT(userID, ah.cfg.JwtSecret)
	if err != nil {
		log.Err(err).Msg("Error generating JWT")
		http.Error(writer, "Error generating token", http.StatusInternalServerError)
		span.RecordError(err)
		span.SetStatus(codes.Error, "JWT generation failed")

		return
	}

	err = ah.storeInRedis(ctx, userID, token)
	if err != nil {
		log.Err(err).Msg("Error storing token in Redis")
		http.Error(writer, "Error storing token in Redis", http.StatusInternalServerError)
		span.RecordError(err)
		span.SetStatus(codes.Error, "Redis storage failed")

		return
	}

	ah.returnToken(writer, token)
	span.SetStatus(codes.Ok, "User registered successfully")
}

//nolint:funlen
func (ah *HandlerAuth) LoginHandler(writer http.ResponseWriter, request *http.Request) {
	ctx, span := ah.tracer.Start(request.Context(), "LoginHandler")
	defer span.End()

	log := logger.GetWithTrace(ctx, ah.logger)

	var req models.User
	if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
		log.Error().Err(err).Msg("error decoding body")
		http.Error(writer, "Invalid request", http.StatusBadRequest)
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid request body")

		return
	}

	span.SetAttributes(attribute.String("user.login", req.Login))

	if err := ah.validate.Struct(req); err != nil {
		log.Error().Err(err).Msg("error validating body")
		http.Error(writer, "Invalid input: "+err.Error(), http.StatusBadRequest)
		span.RecordError(err)
		span.SetStatus(codes.Error, "Validation failed")

		return
	}

	login, found := ah.storage.GetUser(ctx, req.Login)
	if !found || login.HashedPassword == "" {
		log.Error().Msg("Invalid username or password")
		http.Error(writer, "Invalid username or password", http.StatusUnauthorized)
		span.SetStatus(codes.Error, "Authentication failed")

		return
	}

	err := bcrypt.CompareHashAndPassword([]byte(login.HashedPassword), []byte(req.Password))
	if err != nil {
		http.Error(writer, "Invalid username or password", http.StatusUnauthorized)
		log.Error().Err(err).Msg("Invalid username or password")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Password mismatch")

		return
	}

	token, err := ah.generateJWT(login.UserID.String(), ah.cfg.JwtSecret)
	if err != nil {
		log.Err(err).Msg("Error generating token")
		http.Error(writer, "Error generating token", http.StatusInternalServerError)
		span.RecordError(err)
		span.SetStatus(codes.Error, "JWT generation failed")

		return
	}

	err = ah.storeInRedis(ctx, login.UserID.String(), token)
	if err != nil {
		log.Err(err).Msg("Error storing token in Redis")
		http.Error(writer, "Error storing token in Redis", http.StatusInternalServerError)
		span.RecordError(err)
		span.SetStatus(codes.Error, "Redis storage failed")

		return
	}

	ah.returnToken(writer, token)
	span.SetStatus(codes.Ok, "User logged in successfully")
}

func (ah *HandlerAuth) generateJWT(userID string, jwtSecret string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(tokenExpiration).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	result, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", errors.Wrap(err, "Error generating JWT")
	}

	return result, nil
}

func (ah *HandlerAuth) storeInRedis(ctx context.Context, userID string, token string) error {
	err := ah.memStorage.Set(ctx, token, userID, tokenExpiration)

	return errors.Wrap(err, "Error storing token in Redis")
}

func (ah *HandlerAuth) returnToken(writer http.ResponseWriter, token string) {
	// Set the token in a secure, HTTP-only cookie
	//nolint:exhaustruct
	http.SetCookie(writer, &http.Cookie{
		Name:     "Authorization",
		Value:    token,
		Expires:  time.Now().Add(tokenExpiration),
		HttpOnly: true,
		Secure:   true, // Use `false` for local testing without HTTPS
		Path:     "/",
	})
	// Respond with the JWT
	writer.Header().Set("Authorization", token)

	writer.WriteHeader(http.StatusOK)
}
