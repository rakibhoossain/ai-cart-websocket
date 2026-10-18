package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var hmacSecret []byte

const DefaultTokenTTL = 1 * time.Minute

type TokenRequest struct {
	EntityID   string `json:"entity_id"`
	EntityType string `json:"entity_type"`
	Channel    string `json:"channel,omitempty"`
}

type TokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}

func InitJWT(secret string) error {
	if secret == "" {
		return fmt.Errorf("jwt secret cannot be empty")
	}
	hmacSecret = []byte(secret)
	return nil
}

func GenerateToken(req TokenRequest) (*TokenResponse, error) {
	if len(hmacSecret) == 0 {
		return nil, fmt.Errorf("jwt secret not initialized")
	}
	if req.EntityID == "" || req.EntityType == "" {
		return nil, fmt.Errorf("entity_id and entity_type are required")
	}

	now := time.Now()
	expiresAt := now.Add(DefaultTokenTTL)

	claims := jwt.MapClaims{
		"sub":        req.EntityID,
		"entityType": req.EntityType,
		"iat":        now.Unix(),
		"exp":        expiresAt.Unix(),
	}

	if req.Channel != "" {
		claims["channel"] = req.Channel
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(hmacSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign token: %w", err)
	}

	return &TokenResponse{
		Token:     tokenString,
		ExpiresAt: expiresAt.Unix(),
	}, nil
}

func ValidateToken(tokenString string) (jwt.MapClaims, error) {
	if len(hmacSecret) == 0 {
		return nil, fmt.Errorf("jwt secret not initialized")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return hmacSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token")
}
