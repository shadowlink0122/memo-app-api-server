package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"memo-app/src/handlers"
	"memo-app/src/logger"
	"memo-app/src/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMain(m *testing.M) {
	// テスト用のロガーを初期化
	if err := logger.InitLogger(); err != nil {
		panic(err)
	}
	defer logger.CloseLogger()

	// テストを実行
	code := m.Run()
	os.Exit(code)
}

// MockAuthService モック認証サービス
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Register(req *models.RegisterRequest, clientIP string) (*models.AuthResponse, error) {
	args := m.Called(req, clientIP)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AuthResponse), args.Error(1)
}

func (m *MockAuthService) Login(req *models.LoginRequest, clientIP string) (*models.AuthResponse, error) {
	args := m.Called(req, clientIP)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AuthResponse), args.Error(1)
}

func (m *MockAuthService) GetGitHubAuthURL(state string) string {
	args := m.Called(state)
	return args.String(0)
}

func (m *MockAuthService) HandleGitHubCallback(code, state, clientIP string) (*models.AuthResponse, error) {
	args := m.Called(code, state, clientIP)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AuthResponse), args.Error(1)
}

func (m *MockAuthService) RefreshToken(refreshToken string) (*models.AuthResponse, error) {
	args := m.Called(refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AuthResponse), args.Error(1)
}

func (m *MockAuthService) ValidateToken(tokenString string) (*models.User, error) {
	args := m.Called(tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthService) CheckIPLimit(clientIP string) error {
	args := m.Called(clientIP)
	return args.Error(0)
}

func (m *MockAuthService) InvalidateToken(tokenString string) error {
	args := m.Called(tokenString)
	return args.Error(0)
}

func TestAuthHandler_Register(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*MockAuthService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "正常な登録",
			requestBody: map[string]string{
				"username": "testuser",
				"email":    "test@example.com",
				"password": "SecurePass123!",
			},
			setupMock: func(m *MockAuthService) {
				m.On("Register", mock.AnythingOfType("*models.RegisterRequest"), mock.AnythingOfType("string")).
					Return(&models.AuthResponse{
						User: &models.PublicUser{
							ID:       1,
							Username: "testuser",
							Email:    "test@example.com",
							IsActive: true,
						},
						AccessToken:  "access-token",
						RefreshToken: "refresh-token",
						ExpiresIn:    86400,
					}, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   "Registration successful",
		},
		{
			name: "不正なリクエスト形式",
			requestBody: map[string]string{
				"username": "testuser",
				// emailとpasswordが不足
			},
			setupMock:      func(m *MockAuthService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request format",
		},
		{
			name: "重複ユーザー名",
			requestBody: map[string]string{
				"username": "existing",
				"email":    "test@example.com",
				"password": "SecurePass123!",
			},
			setupMock: func(m *MockAuthService) {
				m.On("Register", mock.AnythingOfType("*models.RegisterRequest"), mock.AnythingOfType("string")).
					Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Registration failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックをセットアップ
			mockService := &MockAuthService{}
			tt.setupMock(mockService)

			// ハンドラーを作成
			handler := handlers.NewAuthHandler(mockService)

			// リクエストボディを作成
			jsonBody, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			// レスポンスレコーダーを作成
			w := httptest.NewRecorder()

			// Ginコンテキストを作成
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// ハンドラーを実行
			handler.Register(c)

			// アサーション
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)

			// モックの期待値を検証
			mockService.AssertExpectations(t)
		})
	}
}

func TestAuthHandler_Login(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*MockAuthService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "正常なログイン",
			requestBody: map[string]string{
				"email":    "test@example.com",
				"password": "SecurePass123!",
			},
			setupMock: func(m *MockAuthService) {
				m.On("Login", mock.AnythingOfType("*models.LoginRequest"), mock.AnythingOfType("string")).
					Return(&models.AuthResponse{
						User: &models.PublicUser{
							ID:       1,
							Username: "testuser",
							Email:    "test@example.com",
							IsActive: true,
						},
						AccessToken:  "access-token",
						RefreshToken: "refresh-token",
						ExpiresIn:    86400,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Login successful",
		},
		{
			name: "不正なリクエスト形式",
			requestBody: map[string]string{
				"email": "test@example.com",
				// passwordが不足
			},
			setupMock:      func(m *MockAuthService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request format",
		},
		{
			name: "無効な認証情報",
			requestBody: map[string]string{
				"email":    "test@example.com",
				"password": "wrongpassword",
			},
			setupMock: func(m *MockAuthService) {
				m.On("Login", mock.AnythingOfType("*models.LoginRequest"), mock.AnythingOfType("string")).
					Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Login failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックをセットアップ
			mockService := &MockAuthService{}
			tt.setupMock(mockService)

			// ハンドラーを作成
			handler := handlers.NewAuthHandler(mockService)

			// リクエストボディを作成
			jsonBody, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			// レスポンスレコーダーを作成
			w := httptest.NewRecorder()

			// Ginコンテキストを作成
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// ハンドラーを実行
			handler.Login(c)

			// アサーション
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)

			// モックの期待値を検証
			mockService.AssertExpectations(t)
		})
	}
}

func TestAuthHandler_GetGitHubAuthURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := &MockAuthService{}
	mockService.On("GetGitHubAuthURL", mock.AnythingOfType("string")).
		Return("https://github.com/login/oauth/authorize?client_id=test&state=random")

	handler := handlers.NewAuthHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/github/url", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.GetGitHubAuthURL(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "auth_url")
	assert.Contains(t, w.Body.String(), "state")

	mockService.AssertExpectations(t)
}

func TestAuthHandler_RefreshToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*MockAuthService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "正常なトークンリフレッシュ",
			requestBody: map[string]string{
				"refresh_token": "valid-refresh-token",
			},
			setupMock: func(m *MockAuthService) {
				m.On("RefreshToken", "valid-refresh-token").
					Return(&models.AuthResponse{
						User: &models.PublicUser{
							ID:       1,
							Username: "testuser",
							Email:    "test@example.com",
							IsActive: true,
						},
						AccessToken:  "new-access-token",
						RefreshToken: "new-refresh-token",
						ExpiresIn:    86400,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Token refreshed successfully",
		},
		{
			name: "不正なリクエスト形式",
			requestBody: map[string]string{
				"invalid_field": "value",
			},
			setupMock:      func(m *MockAuthService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request format",
		},
		{
			name: "無効なリフレッシュトークン",
			requestBody: map[string]string{
				"refresh_token": "invalid-refresh-token",
			},
			setupMock: func(m *MockAuthService) {
				m.On("RefreshToken", "invalid-refresh-token").
					Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Token refresh failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックをセットアップ
			mockService := &MockAuthService{}
			tt.setupMock(mockService)

			// ハンドラーを作成
			handler := handlers.NewAuthHandler(mockService)

			// リクエストボディを作成
			jsonBody, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			// レスポンスレコーダーを作成
			w := httptest.NewRecorder()

			// Ginコンテキストを作成
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// ハンドラーを実行
			handler.RefreshToken(c)

			// アサーション
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)

			// モックの期待値を検証
			mockService.AssertExpectations(t)
		})
	}
}

func TestAuthHandler_Logout(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		authHeader     string
		setupMock      func(*MockAuthService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:       "正常なログアウト",
			authHeader: "Bearer valid-token",
			setupMock: func(m *MockAuthService) {
				m.On("InvalidateToken", "valid-token").Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Successfully logged out",
		},
		{
			name:           "Authorization ヘッダーなし",
			authHeader:     "",
			setupMock:      func(m *MockAuthService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Authorization header is required",
		},
		{
			name:           "無効なAuthorizationヘッダー形式",
			authHeader:     "invalid-format",
			setupMock:      func(m *MockAuthService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid authorization header format",
		},
		{
			name:       "ログアウト処理失敗",
			authHeader: "Bearer valid-token",
			setupMock: func(m *MockAuthService) {
				m.On("InvalidateToken", "valid-token").Return(assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to logout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックサービスの設定
			mockService := new(MockAuthService)
			tt.setupMock(mockService)

			// ハンドラーの作成
			handler := handlers.NewAuthHandler(mockService)

			// リクエストを作成
			req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			// レスポンスレコーダーを作成
			w := httptest.NewRecorder()

			// Ginコンテキストを作成
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// ハンドラーを実行
			handler.Logout(c)

			// アサーション
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)

			// モックの期待値を検証
			mockService.AssertExpectations(t)
		})
	}
}
