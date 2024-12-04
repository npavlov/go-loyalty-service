package testutils_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	testutils "github.com/npavlov/go-loyalty-service/internal/test_utils"
)

func TestMockRedis_SetAndGet(t *testing.T) {
	t.Parallel()

	mockRedis := testutils.NewMockRedis()

	key := "test-key"
	value := "test-value"
	expiration := time.Second * 2

	err := mockRedis.Set(context.Background(), key, value, expiration)
	require.NoError(t, err, "expected no error when setting a value")

	retrievedValue, err := mockRedis.Get(context.Background(), key)
	require.NoError(t, err, "expected no error when getting a value")
	assert.Equal(t, value, retrievedValue, "expected value to match set value")

	time.Sleep(expiration + time.Second)

	_, err = mockRedis.Get(context.Background(), key)
	assert.ErrorIs(t, err, testutils.ErrKeyNotFound, "expected error when key is expired")
}

func TestMockRedis_GetNonExistentKey(t *testing.T) {
	t.Parallel()

	mockRedis := testutils.NewMockRedis()

	_, err := mockRedis.Get(context.Background(), "non-existent-key")
	assert.ErrorIs(t, err, testutils.ErrKeyNotFound, "expected error for non-existent key")
}
