package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"memo-app/src/domain"
	"memo-app/src/interface/handler"
	"memo-app/src/logger"
	"memo-app/src/middleware"
	"memo-app/src/models"
	"memo-app/src/service"
	"memo-app/src/usecase"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockJWTService APIテスト用のモック
type MockJWTService struct{}

func (m *MockJWTService) GenerateAccessToken(userID int) (string, error) {
	return "mock-access-token", nil
}

func (m *MockJWTService) GenerateRefreshToken(userID int) (string, error) {
	return "mock-refresh-token", nil
}

func (m *MockJWTService) ValidateToken(tokenString string) (*service.JWTClaims, error) {
	if tokenString == "valid-token-123" {
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

// MockUserRepository APIテスト用のモック
type MockUserRepository struct{}

func (m *MockUserRepository) Create(user *models.User) error { return nil }
func (m *MockUserRepository) GetByID(id int) (*models.User, error) {
	if id == 1 {
		return &models.User{
			ID:       1,
			Username: "testuser",
			Email:    "test@example.com",
			IsActive: true, // アクティブなユーザーとして設定
		}, nil
	}
	return nil, assert.AnError
}
func (m *MockUserRepository) GetByEmail(email string) (*models.User, error)       { return nil, nil }
func (m *MockUserRepository) GetByGitHubID(githubID int64) (*models.User, error)  { return nil, nil }
func (m *MockUserRepository) GetByUsername(username string) (*models.User, error) { return nil, nil }
func (m *MockUserRepository) Update(user *models.User) error                      { return nil }
func (m *MockUserRepository) UpdateLastLogin(userID int) error                    { return nil }
func (m *MockUserRepository) GetIPRegistration(ipAddress string) (*models.IPRegistration, error) {
	return nil, nil
}
func (m *MockUserRepository) CreateIPRegistration(ipReg *models.IPRegistration) error { return nil }
func (m *MockUserRepository) UpdateIPRegistration(ipReg *models.IPRegistration) error { return nil }
func (m *MockUserRepository) GetUserCountByIP(ipAddress string) (int, error)          { return 0, nil }
func (m *MockUserRepository) IsEmailExists(email string) (bool, error)                { return false, nil }
func (m *MockUserRepository) IsUsernameExists(username string) (bool, error)          { return false, nil }
func (m *MockUserRepository) IsGitHubIDExists(githubID int64) (bool, error)           { return false, nil }

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

// MockMemoUsecase for API testing
type MockMemoUsecase struct {
	mock.Mock
}

// PermanentDeleteMemo implements usecase.MemoUsecase.
func (m *MockMemoUsecase) PermanentDeleteMemo(ctx context.Context, id int) error {
	panic("unimplemented")
}

func (m *MockMemoUsecase) CreateMemo(ctx context.Context, req usecase.CreateMemoRequest) (*domain.Memo, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Memo), args.Error(1)
}

func (m *MockMemoUsecase) GetMemo(ctx context.Context, id int) (*domain.Memo, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Memo), args.Error(1)
}

func (m *MockMemoUsecase) ListMemos(ctx context.Context, filter domain.MemoFilter) ([]domain.Memo, int, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]domain.Memo), args.Get(1).(int), args.Error(2)
}

func (m *MockMemoUsecase) UpdateMemo(ctx context.Context, id int, req usecase.UpdateMemoRequest) (*domain.Memo, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Memo), args.Error(1)
}

func (m *MockMemoUsecase) DeleteMemo(ctx context.Context, id int) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockMemoUsecase) ArchiveMemo(ctx context.Context, id int) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockMemoUsecase) RestoreMemo(ctx context.Context, id int) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockMemoUsecase) SearchMemos(ctx context.Context, query string, filter domain.MemoFilter) ([]domain.Memo, int, error) {
	args := m.Called(ctx, query, filter)
	return args.Get(0).([]domain.Memo), args.Get(1).(int), args.Error(2)
}

func setupTestRouter(mockUsecase *MockMemoUsecase) *gin.Engine {
	r := gin.New()

	// テスト用ミドルウェアを適用
	r.Use(middleware.LoggerMiddleware())
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.RateLimitMiddleware())

	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel) // テスト時はWARN以上のみ

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

	// プライベートルート（認証が必要）
	private := r.Group("/api")
	mockJWTService := &MockJWTService{}
	mockUserRepo := &MockUserRepository{}
	private.Use(middleware.AuthMiddleware(mockJWTService, mockUserRepo))
	{
		private.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "これは認証が必要なエンドポイントです",
				"user":    "認証されたユーザー",
			})
		})

		// 実際のメモAPIエンドポイント（Mockを使用）
		if mockUsecase != nil {
			memoHandler := handler.NewMemoHandler(mockUsecase, logger)
			memos := private.Group("/memos")
			{
				memos.POST("", memoHandler.CreateMemo)
				memos.GET("", memoHandler.ListMemos)
				memos.GET("/:id", memoHandler.GetMemo)
				memos.PUT("/:id", memoHandler.UpdateMemo)
				memos.DELETE("/:id", memoHandler.DeleteMemo)
				memos.GET("/search", memoHandler.SearchMemos)
			}
		} else {
			// Mock無しの場合のスタブエンドポイント
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
	}

	return r
}

// === API エンドポイントのテスト ===

func BenchmarkHealthEndpoint(b *testing.B) {
	router := setupTestRouter(nil)

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health", nil)
		router.ServeHTTP(w, req)
	}
}

func BenchmarkRootEndpoint(b *testing.B) {
	router := setupTestRouter(nil)

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/", nil)
		router.ServeHTTP(w, req)
	}
}

// === 統合テスト ===

func TestFullRequestFlow(t *testing.T) {
	router := setupTestRouter(nil)

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
	router := setupTestRouter(nil)

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
	router := setupTestRouter(nil)

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
	router := setupTestRouter(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/hello", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Hello World!", w.Body.String())
}

func TestProtectedEndpoint(t *testing.T) {
	router := setupTestRouter(nil)

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
	router := setupTestRouter(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
}

func TestCORSPreflight(t *testing.T) {
	router := setupTestRouter(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

// === エラーハンドリングテスト ===

func TestNotFoundEndpoint(t *testing.T) {
	router := setupTestRouter(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/nonexistent", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// === 認証テスト ===

func TestAuthenticationRequired(t *testing.T) {
	router := setupTestRouter(nil)

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
	router := setupTestRouter(nil)

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
	router := setupTestRouter(nil)

	// GET /api/memos (スタブ版)
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

	// POST /api/memos (スタブ版)
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

// === Mock を使った実際のメモAPIテスト ===

func TestMemoAPIWithMocks(t *testing.T) {
	mockUsecase := new(MockMemoUsecase)
	router := setupTestRouter(mockUsecase)

	t.Run("Create Memo with Mock", func(t *testing.T) {
		// Mockの設定
		mockUsecase.On("CreateMemo", mock.Anything, mock.AnythingOfType("usecase.CreateMemoRequest")).Return(&domain.Memo{
			ID:       1,
			Title:    "Test Memo",
			Content:  "Test Content",
			Category: "Test",
			Priority: domain.PriorityMedium,
			Status:   domain.StatusActive,
		}, nil)

		// リクエストボディの作成
		requestBody := usecase.CreateMemoRequest{
			Title:    "Test Memo",
			Content:  "Test Content",
			Category: "Test",
			Priority: "medium",
		}
		body, _ := json.Marshal(requestBody)

		// リクエストの実行
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/memos", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer valid-token-123")
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		// レスポンスの検証
		assert.Equal(t, http.StatusCreated, w.Code)

		var response domain.Memo
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Test Memo", response.Title)
		assert.Equal(t, "Test Content", response.Content)

		mockUsecase.AssertExpectations(t)
	})

	t.Run("Get Memo with Mock", func(t *testing.T) {
		// 新しいMockインスタンスを作成
		mockUsecase := new(MockMemoUsecase)
		router := setupTestRouter(mockUsecase)

		// Mockの設定
		mockUsecase.On("GetMemo", mock.Anything, 1).Return(&domain.Memo{
			ID:      1,
			Title:   "Test Memo",
			Content: "Test Content",
			Status:  domain.StatusActive,
		}, nil)

		// リクエストの実行
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/memos/1", nil)
		req.Header.Set("Authorization", "Bearer valid-token-123")
		router.ServeHTTP(w, req)

		// レスポンスの検証
		assert.Equal(t, http.StatusOK, w.Code)

		var response domain.Memo
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Test Memo", response.Title)

		mockUsecase.AssertExpectations(t)
	})

	t.Run("Get Memo Not Found", func(t *testing.T) {
		// 新しいMockインスタンスを作成
		mockUsecase := new(MockMemoUsecase)
		router := setupTestRouter(mockUsecase)

		// Mockの設定 - メモが見つからない場合（適切なエラータイプを使用）
		mockUsecase.On("GetMemo", mock.Anything, 999).Return(nil, usecase.ErrMemoNotFound)

		// リクエストの実行
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/memos/999", nil)
		req.Header.Set("Authorization", "Bearer valid-token-123")
		router.ServeHTTP(w, req)

		// レスポンスの検証 - 正しく404が返されることを期待
		assert.Equal(t, http.StatusNotFound, w.Code)

		mockUsecase.AssertExpectations(t)
	})

	t.Run("List Memos with Mock", func(t *testing.T) {
		// 新しいMockインスタンスを作成
		mockUsecase := new(MockMemoUsecase)
		router := setupTestRouter(mockUsecase)

		// Mockの設定
		mockUsecase.On("ListMemos", mock.Anything, mock.AnythingOfType("domain.MemoFilter")).Return([]domain.Memo{
			{ID: 1, Title: "Memo 1", Content: "Content 1", Status: domain.StatusActive},
			{ID: 2, Title: "Memo 2", Content: "Content 2", Status: domain.StatusActive},
		}, 2, nil)

		// リクエストの実行
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/memos", nil)
		req.Header.Set("Authorization", "Bearer valid-token-123")
		router.ServeHTTP(w, req)

		// レスポンスの検証
		assert.Equal(t, http.StatusOK, w.Code)

		mockUsecase.AssertExpectations(t)
	})

	t.Run("Search Memos with Mock", func(t *testing.T) {
		// 新しいMockインスタンスを作成
		mockUsecase := new(MockMemoUsecase)
		router := setupTestRouter(mockUsecase)

		// Mockの設定 - クエリパラメータに"test"が含まれる場合
		mockUsecase.On("SearchMemos", mock.Anything, "test", mock.AnythingOfType("domain.MemoFilter")).Return([]domain.Memo{
			{ID: 1, Title: "Test Memo", Content: "Test Content", Status: domain.StatusActive},
		}, 1, nil)

		// リクエストの実行 - 正しいパラメータ名'search'を使用
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/memos/search?search=test&limit=10&page=1", nil)
		req.Header.Set("Authorization", "Bearer valid-token-123")
		router.ServeHTTP(w, req)

		// レスポンスの検証
		assert.Equal(t, http.StatusOK, w.Code)

		mockUsecase.AssertExpectations(t)
	})
}
