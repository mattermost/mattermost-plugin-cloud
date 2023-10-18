package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateRandomPassword(t *testing.T) {
	user1Pass := generateRandomPassword("user")

	t.Run("user 1 password valid", func(t *testing.T) {
		parts := strings.Split(user1Pass, "@")
		require.Len(t, parts, 2)
		assert.Equal(t, "user", parts[0])
		assert.Len(t, parts[1], 26)
	})

	user2Pass := generateRandomPassword("user")

	t.Run("user 2 password valid", func(t *testing.T) {
		parts := strings.Split(user2Pass, "@")
		require.Len(t, parts, 2)
		assert.Equal(t, "user", parts[0])
		assert.Len(t, parts[1], 26)
	})

	t.Run("user passwords differ", func(t *testing.T) {
		require.NotEqual(t, user1Pass, user2Pass)
	})

	adminPass := generateRandomPassword("admin")

	t.Run("admin prefix valid", func(t *testing.T) {
		parts := strings.Split(adminPass, "@")
		require.Len(t, parts, 2)
		assert.Equal(t, "admin", parts[0])
		assert.Len(t, parts[1], 26)
	})
}
