package utils

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims là cấu trúc lưu thông tin token
type JWTClaims struct {
	UserID string `json:"user_id"` // ID người dùng
	Role   string `json:"role"`    // Vai trò: admin / user
	jwt.RegisteredClaims
}

// GenerateToken tạo JWT token từ userID và role
// Thường dùng khi user login thành công
func GenerateToken(userID string, role string) (string, error) {
	jwtKey := []byte(os.Getenv("JWT_SECRET")) // Đọc biến môi trường JWT_SECRET
	if len(jwtKey) == 0 {
		return "", errors.New("JWT_SECRET không được thiết lập")
	}

	claims := JWTClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // Token hết hạn sau 24h
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// Tạo token với phương thức HS256
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

// VerifyToken xác minh và parse JWT token
// Trả về thông tin claims nếu token hợp lệ
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

	// Nếu token hợp lệ, trả về claims
	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("Token không hợp lệ")
}
