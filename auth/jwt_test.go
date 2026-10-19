package auth

import (
	"testing"
)

func TestJWTGenerationAndValidation(t *testing.T) {
	secret := "test-secret-key-123456"
	if err := InitJWT(secret); err != nil {
		t.Fatalf("InitJWT failed: %v", err)
	}

	req := TokenRequest{
		Channel: "shop:01931a23-4567-89ab-cdef-0123456789ac",
	}

	res, err := GenerateToken(req)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	if res.Token == "" {
		t.Fatal("Expected non-empty token")
	}
	if res.ExpiresAt <= 0 {
		t.Fatal("Expected positive expires_at")
	}

	claims, err := ValidateToken(res.Token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	if claims["sub"] != req.Channel {
		t.Errorf("Expected sub %s, got %v", req.Channel, claims["sub"])
	}
}

func TestJWTGenerationWithoutChannelReturnsError(t *testing.T) {
	secret := "test-secret-key-123456"
	if err := InitJWT(secret); err != nil {
		t.Fatalf("InitJWT failed: %v", err)
	}

	req := TokenRequest{
		Channel: "",
	}

	_, err := GenerateToken(req)
	if err == nil {
		t.Fatal("Expected error when channel is missing, got nil")
	}
}

func TestJWTValidationInvalidToken(t *testing.T) {
	secret := "test-secret-key-123456"
	if err := InitJWT(secret); err != nil {
		t.Fatalf("InitJWT failed: %v", err)
	}

	_, err := ValidateToken("invalid.token.string")
	if err == nil {
		t.Fatal("Expected error on invalid token, got nil")
	}
}
