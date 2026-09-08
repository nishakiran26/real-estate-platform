package auth

import (
	"strings"
	"testing"
	"time"
)

func TestHashPasswordRoundTrip(t *testing.T) {
	hash, err := HashPassword("Correct@Horse1")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "Correct@Horse1" {
		t.Fatal("password was stored in plain text")
	}
	if !CheckPassword("Correct@Horse1", hash) {
		t.Error("the correct password was rejected")
	}
	if CheckPassword("wrong-password", hash) {
		t.Error("an incorrect password was accepted")
	}
}

func TestHashPasswordIsSalted(t *testing.T) {
	first, _ := HashPassword("same-password")
	second, _ := HashPassword("same-password")
	if first == second {
		t.Error("identical passwords produced identical hashes, so the hash is unsalted")
	}
}

func TestGenerateAndValidateAccessToken(t *testing.T) {
	svc := NewJWTService("test-secret-that-is-long-enough-32", 15, 7, "rera-test")
	clientID := uint(42)

	pair, err := svc.GenerateTokenPair(7, "builder@example.com", "9876543210", "Test Builder",
		[]string{"CLIENT"}, &clientID)
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("token pair is incomplete")
	}

	claims, err := svc.ValidateAccessToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken: %v", err)
	}
	if claims.UserID != 7 {
		t.Errorf("UserID = %d, want 7", claims.UserID)
	}
	if claims.ClientID == nil || *claims.ClientID != 42 {
		t.Error("the builder account id did not survive the round trip")
	}
	if len(claims.Roles) != 1 || claims.Roles[0] != "CLIENT" {
		t.Errorf("Roles = %v, want [CLIENT]", claims.Roles)
	}
}

func TestValidateAccessTokenRejectsTamperedToken(t *testing.T) {
	svc := NewJWTService("test-secret-that-is-long-enough-32", 15, 7, "rera-test")
	pair, _ := svc.GenerateTokenPair(1, "a@example.com", "9000000000", "A", []string{"CUSTOMER"}, nil)

	// Flip a character in the signature.
	parts := strings.Split(pair.AccessToken, ".")
	if len(parts) != 3 {
		t.Fatalf("unexpected token shape: %q", pair.AccessToken)
	}
	tampered := parts[0] + "." + parts[1] + "." + flipLast(parts[2])

	if _, err := svc.ValidateAccessToken(tampered); err == nil {
		t.Error("a token with a broken signature was accepted")
	}
}

func TestValidateAccessTokenRejectsForeignSecret(t *testing.T) {
	issuer := NewJWTService("secret-one-that-is-long-enough-32ch", 15, 7, "rera-test")
	verifier := NewJWTService("secret-two-that-is-long-enough-32ch", 15, 7, "rera-test")

	pair, _ := issuer.GenerateTokenPair(1, "a@example.com", "9000000000", "A", []string{"ADMIN"}, nil)
	if _, err := verifier.ValidateAccessToken(pair.AccessToken); err == nil {
		t.Error("a token signed with a different secret was accepted")
	}
}

func TestValidateAccessTokenRejectsExpiredToken(t *testing.T) {
	svc := NewJWTService("test-secret-that-is-long-enough-32", 15, 7, "rera-test")
	svc.AccessTTL = -time.Minute // already expired when it is issued

	pair, _ := svc.GenerateTokenPair(1, "a@example.com", "9000000000", "A", []string{"CUSTOMER"}, nil)
	if _, err := svc.ValidateAccessToken(pair.AccessToken); err == nil {
		t.Error("an expired token was accepted")
	}
}

func TestValidateRefreshTokenReturnsSubject(t *testing.T) {
	svc := NewJWTService("test-secret-that-is-long-enough-32", 15, 7, "rera-test")
	pair, _ := svc.GenerateTokenPair(99, "a@example.com", "9000000000", "A", []string{"CUSTOMER"}, nil)

	userID, err := svc.ValidateRefreshToken(pair.RefreshToken)
	if err != nil {
		t.Fatalf("ValidateRefreshToken: %v", err)
	}
	if userID != 99 {
		t.Errorf("userID = %d, want 99", userID)
	}
}

func TestRandomTokenIsUniqueAndHashable(t *testing.T) {
	first, err := RandomToken(32)
	if err != nil {
		t.Fatalf("RandomToken: %v", err)
	}
	second, _ := RandomToken(32)
	if first == second {
		t.Error("two random tokens collided")
	}

	// Tokens are stored as digests, so the same input must hash consistently
	// and a different input must not.
	if HashToken(first) != HashToken(first) {
		t.Error("HashToken is not deterministic")
	}
	if HashToken(first) == HashToken(second) {
		t.Error("different tokens produced the same digest")
	}
	if !CompareHash(HashToken(first), HashToken(first)) {
		t.Error("CompareHash rejected two equal digests")
	}
	if CompareHash(HashToken(first), HashToken(second)) {
		t.Error("CompareHash accepted two different digests")
	}
}

func flipLast(s string) string {
	if s == "" {
		return "x"
	}
	last := s[len(s)-1]
	replacement := byte('A')
	if last == 'A' {
		replacement = 'B'
	}
	return s[:len(s)-1] + string(replacement)
}
