package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"memo-app/src/logger"
	"memo-app/src/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

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
			name:           "有効なtoken",
			authHeader:     "Bearer valid-token-123",
			expectedStatus: http.StatusOK,
			expectedBody:   "protected resource",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(middleware.AuthMiddleware())

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
	r.Use(middleware.AuthMiddleware())

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
