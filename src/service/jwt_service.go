package service

import (
	"fmt"
	"time"

	"memo-app/src/config"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims JWT内のカスタムクレーム
type JWTClaims struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	Type   string `json:"type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

// JWTService JWT管理サービスのインターフェース
type JWTService interface {
	GenerateAccessToken(userID int) (string, error)
	GenerateRefreshToken(userID int) (string, error)
	ValidateToken(tokenString string) (*JWTClaims, error)
	ValidateAccessToken(tokenString string) (int, error)
	ValidateRefreshToken(tokenString string) (*JWTClaims, error)
}

// jwtService JWT管理サービスの実装
type jwtService struct {
	config *config.Config
}

// NewJWTService JWT管理サービスを作成
func NewJWTService(cfg *config.Config) JWTService {
	return &jwtService{config: cfg}
}

// GenerateAccessToken アクセストークンを生成
func (s *jwtService) GenerateAccessToken(userID int) (string, error) {
	claims := &JWTClaims{
		UserID: userID,
		Type:   "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.config.Auth.JWTExpiresIn)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "memo-app",
			Subject:   fmt.Sprintf("user:%d", userID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.Auth.JWTSecret))
}

// GenerateRefreshToken リフレッシュトークンを生成
func (s *jwtService) GenerateRefreshToken(userID int) (string, error) {
	claims := &JWTClaims{
		UserID: userID,
		Type:   "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.config.Auth.RefreshExpiresIn)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "memo-app",
			Subject:   fmt.Sprintf("user:%d", userID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.Auth.JWTSecret))
}

// ValidateToken アクセストークンを検証
func (s *jwtService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.Auth.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		if claims.Type != "access" {
			return nil, fmt.Errorf("invalid token type")
		}
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// ValidateRefreshToken リフレッシュトークンを検証
func (s *jwtService) ValidateRefreshToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.Auth.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		if claims.Type != "refresh" {
			return nil, fmt.Errorf("invalid token type")
		}
		return claims, nil
	}

	return nil, fmt.Errorf("invalid refresh token")
}

// ValidateAccessToken アクセストークンを検証してユーザーIDを返す
func (s *jwtService) ValidateAccessToken(tokenString string) (int, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.Auth.JWTSecret), nil
	})

	if err != nil {
		return 0, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		if claims.Type != "access" {
			return 0, fmt.Errorf("invalid token type")
		}
		return claims.UserID, nil
	}

	return 0, fmt.Errorf("invalid access token")
}
