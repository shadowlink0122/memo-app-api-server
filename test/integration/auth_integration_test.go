package integration

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"memo-app/src/config"
	"memo-app/src/database"
	"memo-app/src/handlers"
	"memo-app/src/models"
	"memo-app/src/repository"
	"memo-app/src/service"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthenticationIntegration(t *testing.T) {
	// テスト環境のセットアップをスキップ（データベース接続が必要）
	t.Skip("統合テストはデータベース接続が必要です")

	// この統合テストは実際のデータベースとの接続が設定された場合に実行されます
	cfg := setupTestConfig()
	db := setupTestDatabase(t, cfg)
	defer db.Close()

	// リポジトリとサービスのセットアップ
	userRepo := repository.NewUserRepository(db)
	jwtService := service.NewJWTService(cfg)
	authService := service.NewAuthService(userRepo, jwtService, cfg)
	authHandler := handlers.NewAuthHandler(authService)

	// Ginルーターのセットアップ
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/auth/register", authHandler.Register)
	router.POST("/api/auth/login", authHandler.Login)
	router.GET("/api/auth/github/url", authHandler.GetGitHubAuthURL)
	router.POST("/api/auth/refresh", authHandler.RefreshToken)

	t.Run("ユーザー登録とログインのフル実行テスト", func(t *testing.T) {
		// 1. ユーザー登録
		registerReq := map[string]string{
			"username": "integrationtest",
			"email":    "integration@test.com",
			"password": "SecurePass123!",
		}
		registerBody, _ := json.Marshal(registerReq)

		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(registerBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Real-IP", "192.168.1.100")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusCreated, w.Code)

		var registerResp struct {
			Message string               `json:"message"`
			Data    *models.AuthResponse `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &registerResp)
		require.NoError(t, err)
		assert.Equal(t, "Registration successful", registerResp.Message)
		assert.NotEmpty(t, registerResp.Data.AccessToken)
		assert.NotEmpty(t, registerResp.Data.RefreshToken)

		// 2. 同じユーザーでの再登録は失敗するべき
		req2 := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(registerBody))
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("X-Real-IP", "192.168.1.100")

		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusConflict, w2.Code)

		// 3. ログイン
		loginReq := map[string]string{
			"email":    "integration@test.com",
			"password": "SecurePass123!",
		}
		loginBody, _ := json.Marshal(loginReq)

		req3 := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(loginBody))
		req3.Header.Set("Content-Type", "application/json")
		req3.Header.Set("X-Real-IP", "192.168.1.100")

		w3 := httptest.NewRecorder()
		router.ServeHTTP(w3, req3)

		require.Equal(t, http.StatusOK, w3.Code)

		var loginResp struct {
			Message string               `json:"message"`
			Data    *models.AuthResponse `json:"data"`
		}
		err = json.Unmarshal(w3.Body.Bytes(), &loginResp)
		require.NoError(t, err)
		assert.Equal(t, "Login successful", loginResp.Message)
		assert.NotEmpty(t, loginResp.Data.AccessToken)

		// 4. トークンリフレッシュ
		refreshReq := map[string]string{
			"refresh_token": registerResp.Data.RefreshToken,
		}
		refreshBody, _ := json.Marshal(refreshReq)

		req4 := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewBuffer(refreshBody))
		req4.Header.Set("Content-Type", "application/json")

		w4 := httptest.NewRecorder()
		router.ServeHTTP(w4, req4)

		require.Equal(t, http.StatusOK, w4.Code)

		var refreshResp struct {
			Message string               `json:"message"`
			Data    *models.AuthResponse `json:"data"`
		}
		err = json.Unmarshal(w4.Body.Bytes(), &refreshResp)
		require.NoError(t, err)
		assert.Equal(t, "Token refreshed successfully", refreshResp.Message)
		assert.NotEmpty(t, refreshResp.Data.AccessToken)
	})

	t.Run("IP制限テスト", func(t *testing.T) {
		testIP := "192.168.1.200"

		// 最大登録数まで登録
		for i := 0; i < 3; i++ {
			registerReq := map[string]string{
				"username": fmt.Sprintf("limituser%d", i),
				"email":    fmt.Sprintf("limit%d@test.com", i),
				"password": "SecurePass123!",
			}
			registerBody, _ := json.Marshal(registerReq)

			req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(registerBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Real-IP", testIP)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusCreated, w.Code)
		}

		// 4回目の登録は制限に引っかかるべき
		registerReq := map[string]string{
			"username": "limituser4",
			"email":    "limit4@test.com",
			"password": "SecurePass123!",
		}
		registerBody, _ := json.Marshal(registerReq)

		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(registerBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Real-IP", testIP)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusTooManyRequests, w.Code)
	})

	t.Run("GitHub認証URLテスト", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/auth/github/url", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			AuthURL string `json:"auth_url"`
			State   string `json:"state"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp.AuthURL, "github.com/login/oauth/authorize")
		assert.NotEmpty(t, resp.State)
	})
}

func TestPasswordStrengthValidation(t *testing.T) {
	// 実際のバリデーションシステムをテスト
	t.Skip("バリデータのテストは別ファイルで実施")
}

func TestJWTTokenValidation(t *testing.T) {
	// 実際のJWTサービスをテスト
	t.Skip("JWTサービスのテストは別ファイルで実施")
}

// テスト用のヘルパー関数
func setupTestConfig() *config.Config {
	return &config.Config{
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "test_user",
			Password: "test_password",
			DBName:   "test_db",
			SSLMode:  "disable",
		},
		Auth: config.AuthConfig{
			JWTSecret:          "test-jwt-secret-key-for-testing",
			JWTExpiresIn:       24 * time.Hour,
			RefreshExpiresIn:   7 * 24 * time.Hour,
			GitHubClientID:     "test-github-client-id",
			GitHubClientSecret: "test-github-client-secret",
			GitHubRedirectURL:  "http://localhost:8000/api/auth/github/callback",
			MaxAccountsPerIP:   3,
			IPCooldownPeriod:   24 * time.Hour,
		},
	}
}

func setupTestDatabase(t *testing.T, cfg *config.Config) *sql.DB {
	// テスト用データベース接続のセットアップ
	// 実際の実装では、テスト用のデータベースを使用
	db, err := database.NewDB(&database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
		SSLMode:  cfg.Database.SSLMode,
	}, logrus.New())
	require.NoError(t, err)

	// データベーステーブルの初期化
	// マイグレーションまたはテーブル作成SQLを実行

	return db.DB
}
