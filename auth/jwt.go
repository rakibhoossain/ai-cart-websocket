package auth

import (
	"crypto/rsa"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

var verifyKey *rsa.PublicKey

func LoadPublicKey(keyBytes []byte) error {
	var err error
	verifyKey, err = jwt.ParseRSAPublicKeyFromPEM(keyBytes)
	return err
}

func ValidateToken(tokenString string) (jwt.MapClaims, error) {
	if verifyKey == nil {
		return nil, fmt.Errorf("public key not loaded")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return verifyKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	} else {
		return nil, fmt.Errorf("invalid token")
	}
}
