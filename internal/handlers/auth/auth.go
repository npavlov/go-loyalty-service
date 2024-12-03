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
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"

	"github.com/npavlov/go-loyalty-service/internal/config"
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
}

const (
	tokenExpiration = time.Minute * 60
)

// NewAuthHandler - constructor for AuthHandler.
func NewAuthHandler(st storage.Storage, cfg *config.Config, memSt redis.MemStorage, l *zerolog.Logger) *HandlerAuth {
	return &HandlerAuth{
		logger:     l,
		storage:    st,
		cfg:        cfg,
		validate:   validator.New(),
		memStorage: memSt,
	}
}

func (ah *HandlerAuth) RegisterHandler(writer http.ResponseWriter, request *http.Request) {
	var req models.User
	if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
		ah.logger.Error().Err(err).Msg("error decoding body")
		http.Error(writer, "Invalid request", http.StatusBadRequest)

		return
	}

	// Validate the struct
	if err := ah.validate.Struct(req); err != nil {
		ah.logger.Error().Err(err).Msg("error validating body")
		http.Error(writer, "Invalid input: "+err.Error(), http.StatusBadRequest)

		return
	}

	// Hash the user's password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		ah.logger.Error().Err(err).Msg("error encrypting password")
		http.Error(writer, "Error creating user", http.StatusInternalServerError)

		return
	}

	userID, err := ah.storage.AddUser(request.Context(), req.Login, string(hashedPassword))
	if result := utils.CheckPGConstraint(err); err != nil && result {
		http.Error(writer, "Username already exists", http.StatusConflict)

		return
	}

	if userID == "" || err != nil {
		ah.logger.Error().Msg("Error saving user")

		http.Error(writer, "Error saving user", http.StatusInternalServerError)

		return
	}

	// Generate JWT for the new user
	token, err := ah.generateJWT(userID, ah.cfg.JwtSecret)
	if err != nil {
		log.Err(err).Msg("Error generating JWT")
		http.Error(writer, "Error generating token", http.StatusInternalServerError)

		return
	}

	err = ah.storeInRedis(request.Context(), userID, token)
	if err != nil {
		log.Err(err).Msg("Error storing token in Redis")
		http.Error(writer, "Error storing token in Redis", http.StatusInternalServerError)

		return
	}

	ah.returnToken(writer, token)
}

func (ah *HandlerAuth) LoginHandler(writer http.ResponseWriter, request *http.Request) {
	var req models.User
	if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
		ah.logger.Error().Err(err).Msg("error decoding body")
		http.Error(writer, "Invalid request", http.StatusBadRequest)

		return
	}

	// Validate the struct
	if err := ah.validate.Struct(req); err != nil {
		ah.logger.Error().Err(err).Msg("error validating body")
		http.Error(writer, "Invalid input: "+err.Error(), http.StatusBadRequest)

		return
	}

	// Use the correct field name for username
	username := req.Login // or req.Login if that's the correct field in models.User

	login, found := ah.storage.GetUser(request.Context(), username)
	if !found {
		ah.logger.Error().Msg("Error getting user")
		http.Error(writer, "Error getting user", http.StatusInternalServerError)

		return
	}

	if login == nil {
		ah.logger.Error().Str("username", username).Msg("Login for user not found")
		http.Error(writer, "Invalid username or password", http.StatusUnauthorized)

		return
	}

	// Check if the password is actually retrieved
	if login.HashedPassword == "" {
		ah.logger.Error().Msg("Password not found for user")
		http.Error(writer, "Password not found for user", http.StatusInternalServerError)

		return
	}

	// Verify the provided password
	err := bcrypt.CompareHashAndPassword([]byte(login.HashedPassword), []byte(req.Password))
	if err != nil {
		http.Error(writer, "Invalid username or password", http.StatusUnauthorized)
		ah.logger.Error().Err(err).Msg("Invalid username or password")

		return
	}

	// Generate a JWT token
	token, err := ah.generateJWT(login.UserID.String(), ah.cfg.JwtSecret)
	if err != nil {
		log.Err(err).Msg("Error generating token")
		http.Error(writer, "Error generating token", http.StatusInternalServerError)

		return
	}

	err = ah.storeInRedis(request.Context(), login.UserID.String(), token)
	if err != nil {
		log.Err(err).Msg("Error storing token in Redis")
		http.Error(writer, "Error storing token in Redis", http.StatusInternalServerError)

		return
	}

	ah.returnToken(writer, token)
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
