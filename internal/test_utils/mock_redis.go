package testutils

import (
	"context"
	"errors"
	"sync"
	"time"
)

// MockRedis is a mock implementation of RedisInterface for testing.
type MockRedis struct {
	data  map[string]mockValue
	mutex sync.RWMutex
}

// mockValue holds a value and its expiration time.
type mockValue struct {
	value      string
	expiration time.Time
}

// NewMockRedis initializes a new MockRedis instance.
func NewMockRedis() *MockRedis {
	return &MockRedis{
		data: make(map[string]mockValue),
	}
}

func (m *MockRedis) Ping(_ context.Context) error {
	// Simulate a successful ping.
	return nil
}

func (m *MockRedis) Get(_ context.Context, key string) (string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	value, exists := m.data[key]
	if !exists || (value.expiration.Before(time.Now()) && !value.expiration.IsZero()) {
		return "", errors.New("key not found")
	}

	return value.value, nil
}

func (m *MockRedis) Set(_ context.Context, key string, value string, expiration time.Duration) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	expTime := time.Time{}
	if expiration > 0 {
		expTime = time.Now().Add(expiration)
	}

	m.data[key] = mockValue{
		value:      value,
		expiration: expTime,
	}

	return nil
}
