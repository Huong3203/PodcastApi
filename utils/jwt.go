package utils

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims định nghĩa payload trong token
type JWTClaims struct {
	UserID string `json:"user_id"` // ID người dùng
	VaiTro string `json:"vai_tro"` // ✅ giữ đúng tên "vai_tro" để khớp controller
	jwt.RegisteredClaims
}

// GenerateToken tạo JWT token khi user đăng nhập thành công
func GenerateToken(userID string, vaiTro string) (string, error) {
	jwtKey := []byte(os.Getenv("JWT_SECRET"))
	if len(jwtKey) == 0 {
		return "", errors.New("JWT_SECRET không được thiết lập")
	}

	claims := JWTClaims{
		UserID: userID,
		VaiTro: vaiTro,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // hết hạn sau 24h
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "podcast-api", // ✅ giúp debug dễ hơn
		},
	}

	// Tạo token ký bằng HS256
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

// VerifyToken xác thực và parse JWT token, trả về claims
func VerifyToken(tokenStr string) (*JWTClaims, error) {
	jwtKey := []byte(os.Getenv("JWT_SECRET"))
	if len(jwtKey) == 0 {
		return nil, errors.New("JWT_SECRET không được thiết lập")
	}

	token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("Token không hợp lệ hoặc đã hết hạn")
}
