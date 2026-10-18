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
		EntityID:   "01931a23-4567-89ab-cdef-0123456789ab",
		EntityType: "user",
		Channel:    "shop:01931a23-4567-89ab-cdef-0123456789ac",
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

	if claims["sub"] != req.EntityID {
		t.Errorf("Expected sub %s, got %v", req.EntityID, claims["sub"])
	}
	if claims["entityType"] != req.EntityType {
		t.Errorf("Expected entityType %s, got %v", req.EntityType, claims["entityType"])
	}
	if claims["channel"] != req.Channel {
		t.Errorf("Expected channel %s, got %v", req.Channel, claims["channel"])
	}
}

func TestJWTGenerationWithoutChannel(t *testing.T) {
	secret := "test-secret-key-123456"
	if err := InitJWT(secret); err != nil {
		t.Fatalf("InitJWT failed: %v", err)
	}

	req := TokenRequest{
		EntityID:   "01931a23-4567-89ab-cdef-0123456789ab",
		EntityType: "user",
	}

	res, err := GenerateToken(req)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	claims, err := ValidateToken(res.Token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	if _, ok := claims["channel"]; ok {
		t.Errorf("Expected no channel claim, got %v", claims["channel"])
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
