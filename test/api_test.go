package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"memo-app/src/logger"
	"memo-app/src/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func setupTestRouter() *gin.Engine {
	r := gin.New()

	// テスト用ミドルウェアを適用
	r.Use(middleware.LoggerMiddleware())
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.RateLimitMiddleware())

	// パブリックルート
	public := r.Group("/")
	{
		public.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "Hello World",
				"version": "2.0",
				"service": "memo-app-api-server",
			})
		})

		public.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status": "OK",
				"uptime": "running",
			})
		})

		public.GET("/hello", func(c *gin.Context) {
			c.String(http.StatusOK, "Hello World!")
		})
	}

	// プライベートルート
	private := r.Group("/api")
	private.Use(middleware.AuthMiddleware())
	{
		private.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "これは認証が必要なエンドポイントです",
				"user":    "認証されたユーザー",
			})
		})
	}

	return r
}

// === API エンドポイントのテスト ===

func BenchmarkHealthEndpoint(b *testing.B) {
	router := setupTestRouter()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health", nil)
		router.ServeHTTP(w, req)
	}
}

func BenchmarkRootEndpoint(b *testing.B) {
	router := setupTestRouter()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/", nil)
		router.ServeHTTP(w, req)
	}
}

// === 統合テスト ===

func TestFullRequestFlow(t *testing.T) {
	router := setupTestRouter()

	// 1. ヘルスチェック
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 2. メインエンドポイント
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 3. 認証が必要なエンドポイント
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/protected", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHealthEndpoint(t *testing.T) {
	router := setupTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "OK", response["status"])
	assert.Equal(t, "running", response["uptime"])
}

func TestRootEndpoint(t *testing.T) {
	router := setupTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Hello World", response["message"])
	assert.Equal(t, "2.0", response["version"])
	assert.Equal(t, "memo-app-api-server", response["service"])
}

func TestHelloEndpoint(t *testing.T) {
	router := setupTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/hello", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Hello World!", w.Body.String())
}

func TestProtectedEndpoint(t *testing.T) {
	router := setupTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/protected", nil)
	router.ServeHTTP(w, req)

	// 認証ミドルウェアは現在空実装なので、200が返される
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "これは認証が必要なエンドポイントです", response["message"])
	assert.Equal(t, "認証されたユーザー", response["user"])
}

// === CORS テスト ===

func TestCORSHeaders(t *testing.T) {
	router := setupTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
}

func TestCORSPreflight(t *testing.T) {
	router := setupTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

// === エラーハンドリングテスト ===

func TestNotFoundEndpoint(t *testing.T) {
	router := setupTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/nonexistent", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
