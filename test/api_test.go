package main

import (
	"encoding/json"
	"fmt"
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

	// プライベートルート（実際のサーバーと同じ構成）
	private := r.Group("/api")
	private.Use(middleware.AuthMiddleware())
	{
		private.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "これは認証が必要なエンドポイントです",
				"user":    "認証されたユーザー",
			})
		})

		// メモAPIエンドポイント（テスト用のスタブ）
		private.GET("/memos", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"memos": []gin.H{},
				"total": 0,
			})
		})

		private.POST("/memos", func(c *gin.Context) {
			c.JSON(http.StatusCreated, gin.H{
				"id":      1,
				"message": "メモが作成されました",
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

	// 3. 認証が必要なエンドポイント（認証なし）
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/protected", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// 4. 認証が必要なエンドポイント（有効な認証あり）
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-token-123")
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

	// 認証なしでアクセス - 401 Unauthorized が期待される
	t.Run("Without Authentication", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/protected", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Authorization header required", response["error"])
	})

	// 無効なトークンでアクセス - 401 Unauthorized が期待される
	t.Run("With Invalid Token", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Invalid token", response["error"])
	})

	// 有効なトークンでアクセス - 200 OK が期待される
	t.Run("With Valid Token", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/protected", nil)
		req.Header.Set("Authorization", "Bearer valid-token-123")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "これは認証が必要なエンドポイントです", response["message"])
		assert.Equal(t, "認証されたユーザー", response["user"])
	})
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

// === 認証テスト ===

func TestAuthenticationRequired(t *testing.T) {
	router := setupTestRouter()

	// 認証が必要なエンドポイントの一覧
	endpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/api/protected"},
		{"GET", "/api/memos"},
		{"POST", "/api/memos"},
	}

	for _, endpoint := range endpoints {
		t.Run(fmt.Sprintf("%s %s without auth", endpoint.method, endpoint.path), func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(endpoint.method, endpoint.path, nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

func TestInvalidAuthorizationFormat(t *testing.T) {
	router := setupTestRouter()

	// 無効なAuthorization形式のテスト
	testCases := []struct {
		name   string
		header string
	}{
		{"No Bearer prefix", "invalid-token"},
		{"Empty Bearer token", "Bearer "},
		{"Invalid prefix", "Basic dGVzdA=="},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/protected", nil)
			req.Header.Set("Authorization", tc.header)
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

// === メモAPIテスト ===

func TestMemoEndpointsWithValidAuth(t *testing.T) {
	router := setupTestRouter()

	// GET /api/memos
	t.Run("List Memos with Valid Token", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/memos", nil)
		req.Header.Set("Authorization", "Bearer valid-token-123")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "memos")
		assert.Contains(t, response, "total")
	})

	// POST /api/memos
	t.Run("Create Memo with Valid Token", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/memos", nil)
		req.Header.Set("Authorization", "Bearer valid-token-123")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "id")
		assert.Contains(t, response, "message")
	})
}
