package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtKey []byte

type Claims struct {
	UserID int `json:"user_id"`
	jwt.RegisteredClaims
}

func ConfigureJWT(secret string) error {
	if len(secret) < 16 {
		return errors.New("JWT secret must be at least 16 characters")
	}
	jwtKey = []byte(secret)
	return nil
}

func GenerateToken(userID int) (string, error) {
	if len(jwtKey) == 0 {
		return "", errors.New("jwt secret is not configured")
	}

	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

func ValidateToken(tokenStr string) (*Claims, error) {
	if len(jwtKey) == 0 {
		return nil, errors.New("jwt secret is not configured")
	}

	claims := &Claims{}

	tkn, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrSignatureInvalid) {
			return nil, fmt.Errorf("invalid token signature")
		}
		return nil, fmt.Errorf("invalid token")
	}
	if !tkn.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}
