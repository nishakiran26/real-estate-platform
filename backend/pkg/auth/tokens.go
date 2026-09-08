package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// RandomToken returns a URL-safe random string with n bytes of entropy. It backs
// refresh tokens and password-reset tokens, which are opaque by design: only
// their hash is persisted, so a database leak does not yield usable tokens.
func RandomToken(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// HashToken returns the hex SHA-256 digest used to store a token at rest.
// Tokens are high-entropy random values, so a fast hash is appropriate here;
// passwords use bcrypt instead.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// CompareHash performs a constant-time comparison of two digests.
func CompareHash(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
