package utils

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTClaims struct {
	UserID   string `json:"user_id"`
	Role     string `json:"role"`
	Provider string `json:"provider"`
	jwt.RegisteredClaims
}

func GenerateToken(userID, role, provider string) (string, error) {
	key := []byte(os.Getenv("JWT_SECRET"))
	if len(key) == 0 {
		return "", errors.New("JWT_SECRET không được thiết lập")
	}

	claims := JWTClaims{
		UserID:   userID,
		Role:     role,
		Provider: provider,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(key)
}
