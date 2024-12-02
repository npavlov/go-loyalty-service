package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	handlers "github.com/npavlov/go-loyalty-service/internal/handlers/auth"
	testutils "github.com/npavlov/go-loyalty-service/internal/test_utils"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"

	"github.com/npavlov/go-loyalty-service/internal/config"
	"github.com/npavlov/go-loyalty-service/internal/models"
)

func TestHandlerAuth_RegisterHandler(t *testing.T) {
	// Mock storage and Redis
	mockStorage := testutils.NewMockStorage()
	mockRedis := testutils.NewMockRedis()

	logger := zerolog.New(nil)
	cfg := &config.Config{
		JwtSecret: "test-secret",
	}

	authHandler := handlers.NewAuthHandler(mockStorage, cfg, mockRedis, &logger)

	// Prepare the request
	user := models.User{
		Login:    "testuser",
		Password: "password123",
	}
	body, _ := json.Marshal(user)
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Record the response
	rec := httptest.NewRecorder()
	authHandler.RegisterHandler(rec, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rec.Code)
	cookie := rec.Result().Cookies()
	assert.NotNil(t, cookie)
	assert.Contains(t, rec.Header().Get("Authorization"), "eyJhb") // JWT header base64 prefix
}

func TestHandlerAuth_LoginHandler(t *testing.T) {
	// Mock storage and Redis
	mockStorage := testutils.NewMockStorage()
	mockRedis := testutils.NewMockRedis()

	// Add a test user to the mock storage
	username := "testuser"
	password := "password123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	userID := uuid.New().String()
	mockStorage.Users[username] = &models.Login{
		UserId:         uuid.MustParse(userID),
		HashedPassword: string(hashedPassword),
	}

	logger := zerolog.New(nil)
	cfg := &config.Config{
		JwtSecret: "test-secret",
	}

	authHandler := handlers.NewAuthHandler(mockStorage, cfg, mockRedis, &logger)

	// Prepare the request
	user := models.User{
		Login:    username,
		Password: password,
	}
	body, _ := json.Marshal(user)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Record the response
	rec := httptest.NewRecorder()
	authHandler.LoginHandler(rec, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Authorization"), "eyJhb") // JWT header base64 prefix

	// Check Redis for token storage
	token := rec.Header().Get("Authorization")
	storedUserID, err := mockRedis.Get(context.Background(), token)
	assert.NoError(t, err)
	assert.Equal(t, userID, storedUserID)
}

func TestHandlerAuth_LoginHandlerInvalidPassword(t *testing.T) {
	// Mock storage and Redis
	mockStorage := testutils.NewMockStorage()
	mockRedis := testutils.NewMockRedis()

	// Add a test user to the mock storage
	username := "testuser"
	password := "password123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	userID := uuid.New().String()
	mockStorage.Users[username] = &models.Login{
		UserId:         uuid.MustParse(userID),
		HashedPassword: string(hashedPassword),
	}

	logger := zerolog.New(nil)
	cfg := &config.Config{
		JwtSecret: "test-secret",
	}

	authHandler := handlers.NewAuthHandler(mockStorage, cfg, mockRedis, &logger)

	// Prepare the request with an incorrect password
	user := models.User{
		Login:    username,
		Password: "wrongpassword",
	}
	body, _ := json.Marshal(user)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Record the response
	rec := httptest.NewRecorder()
	authHandler.LoginHandler(rec, req)

	// Assertions
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Empty(t, rec.Header().Get("Authorization"))
}

func TestHandlerAuth_RegisterHandler_UserAlreadyExists(t *testing.T) {
	// Mock storage and Redis
	mockStorage := testutils.NewMockStorage()
	mockRedis := testutils.NewMockRedis()

	// Add an existing user to the mock storage
	existingUsername := "existinguser"
	existingPassword := "password123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(existingPassword), bcrypt.DefaultCost)
	_, _ = mockStorage.AddUser(context.Background(), existingUsername, string(hashedPassword))

	logger := zerolog.New(nil)
	cfg := &config.Config{
		JwtSecret: "test-secret",
	}

	authHandler := handlers.NewAuthHandler(mockStorage, cfg, mockRedis, &logger)

	// Prepare the request with the same username
	user := models.User{
		Login:    existingUsername,
		Password: "anotherpassword",
	}
	body, _ := json.Marshal(user)
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Record the response
	rec := httptest.NewRecorder()
	authHandler.RegisterHandler(rec, req)

	// Assertions
	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Contains(t, rec.Body.String(), "Username already exists")
}