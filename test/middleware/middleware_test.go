package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"memo-app/src/logger"
	"memo-app/src/middleware"
	"memo-app/src/models"
	"memo-app/src/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// MockJWTService は認証ミドルウェアテスト用のモック
type MockJWTService struct{}

func (m *MockJWTService) GenerateAccessToken(userID int) (string, error) {
	return "mock-access-token", nil
}

func (m *MockJWTService) GenerateRefreshToken(userID int) (string, error) {
	return "mock-refresh-token", nil
}

func (m *MockJWTService) ValidateToken(tokenString string) (*service.JWTClaims, error) {
	if tokenString == "valid-token" {
		return &service.JWTClaims{
			UserID: 1,
			Type:   "access",
		}, nil
	}
	return nil, assert.AnError
}

func (m *MockJWTService) ValidateAccessToken(tokenString string) (int, error) {
	if tokenString == "valid-token-123" {
		return 1, nil
	}
	return 0, assert.AnError
}

func (m *MockJWTService) ValidateRefreshToken(tokenString string) (*service.JWTClaims, error) {
	if tokenString == "valid-refresh-token" {
		return &service.JWTClaims{
			UserID: 1,
			Type:   "refresh",
		}, nil
	}
	return nil, assert.AnError
}

func (m *MockJWTService) InvalidateToken(tokenString string) error {
	// モックでは常に成功として扱う
	return nil
}

func (m *MockJWTService) IsTokenInvalidated(tokenString string) bool {
	// テスト用の無効化されたトークン
	return tokenString == "invalidated-token"
}

// MockUserRepository は認証ミドルウェアテスト用のモック
type MockUserRepository struct{}

func (m *MockUserRepository) Create(user *models.User) error {
	return nil
}

func (m *MockUserRepository) GetByID(id int) (*models.User, error) {
	if id == 1 {
		return &models.User{
			ID:       1,
			Username: "testuser",
			Email:    "test@example.com",
			IsActive: true,
		}, nil
	}
	return nil, assert.AnError
}

func (m *MockUserRepository) GetByEmail(email string) (*models.User, error) {
	return &models.User{
		ID:       1,
		Username: "testuser",
		Email:    email,
	}, nil
}

func (m *MockUserRepository) GetByGitHubID(githubID int64) (*models.User, error) {
	return &models.User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		GitHubID: &githubID,
	}, nil
}

func (m *MockUserRepository) GetByUsername(username string) (*models.User, error) {
	return &models.User{
		ID:       1,
		Username: username,
		Email:    "test@example.com",
	}, nil
}

func (m *MockUserRepository) Update(user *models.User) error {
	return nil
}

func (m *MockUserRepository) UpdateLastLogin(userID int) error {
	return nil
}

func (m *MockUserRepository) GetIPRegistration(ipAddress string) (*models.IPRegistration, error) {
	return &models.IPRegistration{
		IPAddress:  ipAddress,
		UserCount:  1,
		LastUsedAt: time.Now(),
	}, nil
}

func (m *MockUserRepository) CreateIPRegistration(ipReg *models.IPRegistration) error {
	return nil
}

func (m *MockUserRepository) UpdateIPRegistration(ipReg *models.IPRegistration) error {
	return nil
}

func (m *MockUserRepository) GetUserCountByIP(ipAddress string) (int, error) {
	return 1, nil
}

func (m *MockUserRepository) IsEmailExists(email string) (bool, error) {
	return false, nil
}

func (m *MockUserRepository) IsUsernameExists(username string) (bool, error) {
	return false, nil
}

func (m *MockUserRepository) IsGitHubIDExists(githubID int64) (bool, error) {
	return false, nil
}

func TestMain(m *testing.M) {
	// テスト前の初期化
	gin.SetMode(gin.TestMode)

	// テスト用ロガーを初期化
	os.Setenv("LOG_LEVEL", "error")          // テスト時はエラーレベルのみ
	os.Setenv("LOG_UPLOAD_ENABLED", "false") // テスト時はアップロード無効

	if err := logger.InitLogger(); err != nil {
		panic(err)
	}

	// テスト実行
	code := m.Run()

	// テスト後のクリーンアップ
	logger.CloseLogger()

	os.Exit(code)
}

func TestLoggerMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(middleware.LoggerMiddleware())

	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("X-Real-IP", "192.168.1.1")

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "test")
}

func TestLoggerMiddlewareWithError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(middleware.LoggerMiddleware())

	r.GET("/error", func(c *gin.Context) {
		c.Error(gin.Error{Err: assert.AnError, Type: gin.ErrorTypePublic})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "test error"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/error", nil)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(middleware.CORSMiddleware())

	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})

	tests := []struct {
		name           string
		method         string
		origin         string
		expectedStatus int
		checkHeaders   bool
	}{
		{
			name:           "通常のGETリクエスト",
			method:         "GET",
			origin:         "http://localhost:3000",
			expectedStatus: http.StatusOK,
			checkHeaders:   true,
		},
		{
			name:           "OPTIONSプリフライトリクエスト",
			method:         "OPTIONS",
			origin:         "https://example.com",
			expectedStatus: http.StatusNoContent,
			checkHeaders:   true,
		},
		{
			name:           "Originヘッダーなし",
			method:         "GET",
			origin:         "",
			expectedStatus: http.StatusOK,
			checkHeaders:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(tt.method, "/test", nil)

			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkHeaders {
				assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
				assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
				assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Authorization")
				assert.Equal(t, "86400", w.Header().Get("Access-Control-Max-Age"))
			}
		})
	}
}

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "認証ヘッダーなし",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Authorization header required",
		},
		{
			name:           "不正な認証形式",
			authHeader:     "Basic dGVzdA==",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Invalid authorization format",
		},
		{
			name:           "空のtoken",
			authHeader:     "Bearer ",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Token is empty",
		},
		{
			name:           "無効なtoken",
			authHeader:     "Bearer invalid-token",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Invalid token",
		},
		{
			name:           "無効化されたtoken",
			authHeader:     "Bearer invalidated-token",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Token has been invalidated",
		},
		{
			name:           "有効なtoken",
			authHeader:     "Bearer valid-token-123",
			expectedStatus: http.StatusOK,
			expectedBody:   "protected resource",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			// モックサービスを作成してAuthMiddlewareに渡す
			mockJWTService := &MockJWTService{}
			mockUserRepo := &MockUserRepository{}
			r.Use(middleware.AuthMiddleware(mockJWTService, mockUserRepo))

			r.GET("/protected", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "protected resource"})
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/protected", nil)

			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
		})
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(middleware.RateLimitMiddleware())

	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 複数回リクエストしてレート制限が動作することを確認
	// 現在は空実装なので、すべてのリクエストが通る
	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Real-IP", "192.168.1.1")

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	}
}

func TestMiddlewareChain(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(middleware.LoggerMiddleware())
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.RateLimitMiddleware())
	// モックサービスを作成してAuthMiddlewareに渡す
	mockJWTService := &MockJWTService{}
	mockUserRepo := &MockUserRepository{}
	r.Use(middleware.AuthMiddleware(mockJWTService, mockUserRepo))

	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "all middleware applied"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Authorization", "Bearer valid-token-123") // 有効なトークンを追加

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "all middleware applied")

	// CORSヘッダーが設定されていることを確認
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}
