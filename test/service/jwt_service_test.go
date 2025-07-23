package service

import (
	"testing"
	"time"

	"memo-app/src/config"
	"memo-app/src/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTService_GenerateAndValidateAccessToken(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret:        "test-secret-key-for-testing",
			JWTExpiresIn:     24 * time.Hour,
			RefreshExpiresIn: 7 * 24 * time.Hour,
		},
	}

	jwtService := service.NewJWTService(cfg)

	tests := []struct {
		name   string
		userID int
	}{
		{
			name:   "有効なユーザーID",
			userID: 1,
		},
		{
			name:   "大きなユーザーID",
			userID: 999999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// アクセストークンを生成
			token, err := jwtService.GenerateAccessToken(tt.userID)
			require.NoError(t, err)
			assert.NotEmpty(t, token)

			// 生成されたトークンを検証
			userID, err := jwtService.ValidateAccessToken(token)
			require.NoError(t, err)
			assert.Equal(t, tt.userID, userID)
		})
	}
}

func TestJWTService_GenerateAndValidateRefreshToken(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret:        "test-secret-key-for-testing",
			JWTExpiresIn:     24 * time.Hour,
			RefreshExpiresIn: 7 * 24 * time.Hour,
		},
	}

	jwtService := service.NewJWTService(cfg)

	tests := []struct {
		name   string
		userID int
	}{
		{
			name:   "有効なユーザーID",
			userID: 1,
		},
		{
			name:   "大きなユーザーID",
			userID: 999999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// リフレッシュトークンを生成
			token, err := jwtService.GenerateRefreshToken(tt.userID)
			require.NoError(t, err)
			assert.NotEmpty(t, token)

			// 生成されたトークンを検証
			claims, err := jwtService.ValidateRefreshToken(token)
			require.NoError(t, err)
			assert.Equal(t, tt.userID, claims.UserID)
			assert.Equal(t, "refresh", claims.Type)
		})
	}
}

func TestJWTService_ValidateToken_GeneralValidation(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret:        "test-secret-key-for-testing",
			JWTExpiresIn:     24 * time.Hour,
			RefreshExpiresIn: 7 * 24 * time.Hour,
		},
	}

	jwtService := service.NewJWTService(cfg)
	userID := 123

	// アクセストークンを生成
	accessToken, err := jwtService.GenerateAccessToken(userID)
	require.NoError(t, err)

	// リフレッシュトークンを生成
	refreshToken, err := jwtService.GenerateRefreshToken(userID)
	require.NoError(t, err)

	tests := []struct {
		name        string
		token       string
		expectError bool
	}{
		{
			name:        "有効なアクセストークン",
			token:       accessToken,
			expectError: false,
		},
		{
			name:        "リフレッシュトークン（無効なタイプ）",
			token:       refreshToken,
			expectError: true, // ValidateTokenはアクセストークンのみ許可
		},
		{
			name:        "無効なトークン",
			token:       "invalid.token.here",
			expectError: true,
		},
		{
			name:        "空のトークン",
			token:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := jwtService.ValidateToken(tt.token)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, claims)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, claims)
				if claims != nil {
					assert.Equal(t, userID, claims.UserID)
				}
			}
		})
	}
}

func TestJWTService_InvalidSecret(t *testing.T) {
	// 異なるシークレットで生成されたJWTサービス
	cfg1 := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret:        "secret-1",
			JWTExpiresIn:     24 * time.Hour,
			RefreshExpiresIn: 7 * 24 * time.Hour,
		},
	}

	cfg2 := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret:        "secret-2",
			JWTExpiresIn:     24 * time.Hour,
			RefreshExpiresIn: 7 * 24 * time.Hour,
		},
	}

	jwtService1 := service.NewJWTService(cfg1)
	jwtService2 := service.NewJWTService(cfg2)

	userID := 123

	// service1でトークンを生成
	token, err := jwtService1.GenerateAccessToken(userID)
	require.NoError(t, err)

	// service2で検証（異なるシークレット）
	_, err = jwtService2.ValidateAccessToken(token)
	assert.Error(t, err, "異なるシークレットで生成されたトークンは検証に失敗するべき")
}

func TestJWTService_TokenTypes(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret:        "test-secret-key-for-testing",
			JWTExpiresIn:     24 * time.Hour,
			RefreshExpiresIn: 7 * 24 * time.Hour,
		},
	}

	jwtService := service.NewJWTService(cfg)
	userID := 123

	// アクセストークンを生成
	accessToken, err := jwtService.GenerateAccessToken(userID)
	require.NoError(t, err)

	// リフレッシュトークンを生成
	refreshToken, err := jwtService.GenerateRefreshToken(userID)
	require.NoError(t, err)

	// アクセストークンをリフレッシュトークンとして検証（失敗すべき）
	_, err = jwtService.ValidateRefreshToken(accessToken)
	assert.Error(t, err, "アクセストークンはリフレッシュトークンとして検証されるべきではない")

	// リフレッシュトークンをアクセストークンとして検証（失敗すべき）
	_, err = jwtService.ValidateAccessToken(refreshToken)
	assert.Error(t, err, "リフレッシュトークンはアクセストークンとして検証されるべきではない")
}
