package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashPassword(t *testing.T) {
	password := "my_secret_password"

	hash, err := HashPassword(password)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash, "Hash should not be equal to plain text password")

	// Verify it generates different hashes for the same password due to random salt
	hash2, err := HashPassword(password)
	assert.NoError(t, err)
	assert.NotEqual(t, hash, hash2, "Hashes should be different for the same password due to salting")
}

func TestCheckPasswordHash(t *testing.T) {
	password := "another_secret"
	wrongPassword := "wrong_password"

	// Create a valid hash
	hash, err := HashPassword(password)
	assert.NoError(t, err)

	// Test correct password
	match := CheckPasswordHash(password, hash)
	assert.True(t, match, "Valid password should match the hash")

	// Test wrong password
	match = CheckPasswordHash(wrongPassword, hash)
	assert.False(t, match, "Wrong password should not match the hash")

	// Test invalid hash format
	invalidHash := "not_a_valid_bcrypt_hash"
	match = CheckPasswordHash(password, invalidHash)
	assert.False(t, match, "Invalid hash format should return false")
}
