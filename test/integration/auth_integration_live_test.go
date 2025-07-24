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

func TestAuthenticationIntegrationLive(t *testing.T) {
	// Docker環境で実際のデータベースを使用した統合テスト
	cfg := setupLiveTestConfig()
	db := setupLiveTestDatabase(t, cfg)
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
		uniqueID := fmt.Sprintf("%d", time.Now().UnixNano())

		// 1. ユーザー登録
		registerReq := map[string]string{
			"username": "integrationtest" + uniqueID,
			"email":    "integration" + uniqueID + "@test.com",
			"password": "SecurePass123!",
		}
		registerBody, _ := json.Marshal(registerReq)

		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(registerBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Real-IP", "192.168.1.100")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// デバッグ: レスポンスボディを出力
		t.Logf("Status Code: %d", w.Code)
		t.Logf("Response Body: %s", w.Body.String())

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
			"email":    "integration" + uniqueID + "@test.com",
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

		// GitHub OAuth URLの構造をより詳細にテスト
		assert.Contains(t, resp.AuthURL, "client_id=")
		assert.Contains(t, resp.AuthURL, "redirect_uri=")
		assert.Contains(t, resp.AuthURL, "scope=")
		assert.Contains(t, resp.AuthURL, "state=")
	})
}

// Docker環境用の設定
func setupLiveTestConfig() *config.Config {
	// テスト用の設定を読み込み（環境変数を考慮）
	cfg := config.LoadConfig()
	
	// テスト用のデフォルト値を設定
	if cfg.Database.Host == "" {
		cfg.Database.Host = "localhost"
	}
	if cfg.Database.Port == 0 {
		cfg.Database.Port = 5433 // テストデータベースのデフォルトポート
	}
	if cfg.Database.User == "" {
		cfg.Database.User = "memo_user"
	}
	if cfg.Database.Password == "" {
		cfg.Database.Password = "memo_password"
	}
	if cfg.Database.DBName == "" {
		cfg.Database.DBName = "memo_db"
	}
	if cfg.Database.SSLMode == "" {
		cfg.Database.SSLMode = "disable"
	}

	// Authの設定もテスト用の値を設定
	if cfg.Auth.JWTSecret == "" {
		cfg.Auth.JWTSecret = "test-jwt-secret-key-for-testing"
	}
	if cfg.Auth.JWTExpiresIn == 0 {
		cfg.Auth.JWTExpiresIn = 24 * time.Hour
	}
	if cfg.Auth.RefreshExpiresIn == 0 {
		cfg.Auth.RefreshExpiresIn = 7 * 24 * time.Hour
	}
	if cfg.Auth.GitHubClientID == "" {
		cfg.Auth.GitHubClientID = "test-github-client-id"
	}
	if cfg.Auth.GitHubClientSecret == "" {
		cfg.Auth.GitHubClientSecret = "test-github-client-secret"
	}
	if cfg.Auth.GitHubRedirectURL == "" {
		cfg.Auth.GitHubRedirectURL = "http://localhost:8000/api/auth/github/callback"
	}
	if cfg.Auth.MaxAccountsPerIP == 0 {
		cfg.Auth.MaxAccountsPerIP = 3
	}
	if cfg.Auth.IPCooldownPeriod == 0 {
		cfg.Auth.IPCooldownPeriod = 24 * time.Hour
	}

	return cfg
}

func setupLiveTestDatabase(t *testing.T, cfg *config.Config) *sql.DB {
	// Docker環境のデータベースに接続
	db, err := database.NewDB(&database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
		SSLMode:  cfg.Database.SSLMode,
	}, logrus.New())
	require.NoError(t, err)

	return db.DB
}
