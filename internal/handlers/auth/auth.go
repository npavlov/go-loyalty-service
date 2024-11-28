package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v4"
	"github.com/npavlov/go-loyalty-service/internal/config"
	"github.com/npavlov/go-loyalty-service/internal/models"
	"github.com/npavlov/go-loyalty-service/internal/storage"
	"github.com/npavlov/go-loyalty-service/internal/utils"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

type HandlerAuth struct {
	logger   *zerolog.Logger
	storage  *storage.DBStorage
	cfg      *config.Config
	validate *validator.Validate
	redis    *redis.Client
}

var (
	tokenExpiration = time.Minute * 60
)

// NewAuthHandler - constructor for AuthHandler.
func NewAuthHandler(storage *storage.DBStorage, cfg *config.Config, redisClient *redis.Client, l *zerolog.Logger) *HandlerAuth {
	return &HandlerAuth{
		logger:   l,
		storage:  storage,
		cfg:      cfg,
		validate: validator.New(),
		redis:    redisClient,
	}
}

func (ah *HandlerAuth) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req models.User
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Validate the struct
	if err := ah.validate.Struct(req); err != nil {
		http.Error(w, "Invalid input: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Hash the user's password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error creating user", http.StatusInternalServerError)
		return
	}

	userID, err := ah.storage.AddUser(r.Context(), req.Login, string(hashedPassword))
	if err != nil {
		ah.logger.Error().Err(err).Msg("Error adding user")
		result := utils.CheckPGConstraint(err)
		if result {
			http.Error(w, "Username already exists", http.StatusConflict)

			return
		}

		http.Error(w, "Error saving user", http.StatusInternalServerError)

		return
	}

	if userID == "" {
		ah.logger.Error().Msg("Error saving user")

		http.Error(w, "Error saving user", http.StatusInternalServerError)

		return
	}

	// Generate JWT for the new user
	token, err := ah.generateJWT(userID, ah.cfg.JwtSecret)
	if err != nil {
		log.Err(err).Msg("Error generating JWT")

		http.Error(w, "Error generating token", http.StatusInternalServerError)

		return
	}

	err = ah.storeInRedis(r.Context(), userID, token)
	if err != nil {
		log.Err(err).Msg("Error storing token in Redis")

		return
	}

	// Respond with the JWT
	_ = json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func (ah *HandlerAuth) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req models.User
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Validate the struct
	if err := ah.validate.Struct(req); err != nil {
		http.Error(w, "Invalid input: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Use the correct field name for username
	username := req.Login // or req.Login if that's the correct field in models.User

	login, err := ah.storage.GetUser(r.Context(), username)
	if err != nil {
		http.Error(w, "Error querying user", http.StatusInternalServerError)

		return
	}

	if login == nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)

		return
	}

	// Check if the password is actually retrieved
	if login.HashedPassword == "" {
		http.Error(w, "Password not found for user", http.StatusInternalServerError)
		return
	}

	// Verify the provided password
	err = bcrypt.CompareHashAndPassword([]byte(login.HashedPassword), []byte(req.Password))
	if err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Generate a JWT token
	token, err := ah.generateJWT(login.UserId.String(), ah.cfg.JwtSecret)
	if err != nil {
		log.Err(err).Msg("Error generating token")

		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}

	err = ah.storeInRedis(r.Context(), login.UserId.String(), token)
	if err != nil {
		log.Err(err).Msg("Error storing token in Redis")

		return
	}

	// Respond with the JWT
	_ = json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func (ah *HandlerAuth) generateJWT(userID string, jwtSecret string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(tokenExpiration).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}

func (ah *HandlerAuth) storeInRedis(ctx context.Context, userID string, token string) error {
	expiration := tokenExpiration
	err := ah.redis.Set(ctx, token, userID, expiration).Err()

	return err
}
