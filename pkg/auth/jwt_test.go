package auth

import (
	"testing"
	"time"
)

func TestGenerateToken_And_ValidateToken_Success(t *testing.T) {
	// Arrange
	manager, err := NewJWTManager("secret-key-123", 1)
	if err != nil {
		t.Fatalf("failed to create JWTManager: %v", err)
	}

	expectedUserID := 42
	expectedEmail := "user@example.com"
	expectedUsername := "testuser"

	// Act: генерируем токен
	tokenString, expTime, err := manager.GenerateToken(expectedUserID, expectedEmail, expectedUsername)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if tokenString == "" {
		t.Error("tokenString should not be empty")
	}
	if expTime.IsZero() {
		t.Error("expTime should not be zero")
	}

	// Act: валидируем токен
	claims, err := manager.ValidateToken(tokenString)
	if err != nil {
		t.Fatalf("ValidateToken failed on valid token: %v", err)
	}

	// Assert: проверяем данные в claims
	if claims.UserID != expectedUserID {
		t.Errorf("UserID mismatch: expected %d, got %d", expectedUserID, claims.UserID)
	}
	if claims.Email != expectedEmail {
		t.Errorf("Email mismatch: expected %q, got %q", expectedEmail, claims.Email)
	}
	if claims.Username != expectedUsername {
		t.Errorf("Username mismatch: expected %q, got %q", expectedUsername, claims.Username)
	}

	// Assert: время истечения должно быть примерно через 1 час
	now := time.Now().UTC()
	diff := expTime.Sub(now)
	expectedTTL := time.Hour
	if diff < expectedTTL-5*time.Second || diff > expectedTTL+5*time.Second {
		t.Errorf("expiration time mismatch: expected ~%v, got ~%v", expectedTTL, diff)
	}
}
