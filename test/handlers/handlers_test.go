package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"memo-app/src/domain"
	"memo-app/src/interface/handler"
	"memo-app/src/usecase"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}

// MockMemoUsecase は MemoUsecase のモック実装
type MockMemoUsecase struct {
	mock.Mock
}

// PermanentDeleteMemo implements usecase.MemoUsecase.
func (m *MockMemoUsecase) PermanentDeleteMemo(ctx context.Context, userID int, id int) error {
	args := m.Called(ctx, userID, id)
	return args.Error(0)
}

func (m *MockMemoUsecase) CreateMemo(ctx context.Context, userID int, req usecase.CreateMemoRequest) (*domain.Memo, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Memo), args.Error(1)
}

func (m *MockMemoUsecase) GetMemo(ctx context.Context, userID int, id int) (*domain.Memo, error) {
	args := m.Called(ctx, userID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Memo), args.Error(1)
}

func (m *MockMemoUsecase) ListMemos(ctx context.Context, userID int, filter domain.MemoFilter) ([]domain.Memo, int, error) {
	args := m.Called(ctx, userID, filter)
	return args.Get(0).([]domain.Memo), args.Get(1).(int), args.Error(2)
}

func (m *MockMemoUsecase) UpdateMemo(ctx context.Context, userID int, id int, req usecase.UpdateMemoRequest) (*domain.Memo, error) {
	args := m.Called(ctx, userID, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Memo), args.Error(1)
}

func (m *MockMemoUsecase) DeleteMemo(ctx context.Context, userID int, id int) error {
	args := m.Called(ctx, userID, id)
	return args.Error(0)
}

func (m *MockMemoUsecase) ArchiveMemo(ctx context.Context, userID int, id int) error {
	args := m.Called(ctx, userID, id)
	return args.Error(0)
}

func (m *MockMemoUsecase) RestoreMemo(ctx context.Context, userID int, id int) error {
	args := m.Called(ctx, userID, id)
	return args.Error(0)
}

func (m *MockMemoUsecase) SearchMemos(ctx context.Context, userID int, query string, filter domain.MemoFilter) ([]domain.Memo, int, error) {
	args := m.Called(ctx, userID, query, filter)
	return args.Get(0).([]domain.Memo), args.Get(1).(int), args.Error(2)
}

func setupTestRouter(mockUsecase *MockMemoUsecase) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	logger := logrus.New()
	memoHandler := handler.NewMemoHandler(mockUsecase, logger)

	// ルートの設定
	api := r.Group("/api/memos")
	// モック認証ミドルウェアを追加
	api.Use(mockAuthMiddleware())
	{
		api.POST("", memoHandler.CreateMemo)
		api.GET("", memoHandler.ListMemos)
		api.GET("/:id", memoHandler.GetMemo)
		api.PUT("/:id", memoHandler.UpdateMemo)
		api.DELETE("/:id", memoHandler.DeleteMemo)
		api.DELETE("/:id/permanent", memoHandler.PermanentDeleteMemo)
		api.PATCH("/:id/archive", memoHandler.ArchiveMemo)
		api.PATCH("/:id/restore", memoHandler.RestoreMemo)
		api.GET("/search", memoHandler.SearchMemos)
	}

	return r
}

// モック認証ミドルウェア - テスト用
func mockAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// テスト用の固定ユーザーID
		c.Set("user_id", 1)
		c.Next()
	}
}

func TestMemoHandler_CreateMemo(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*MockMemoUsecase)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "successful creation",
			requestBody: usecase.CreateMemoRequest{
				Title:    "Test Memo",
				Content:  "This is a test memo",
				Category: "Test",
				Priority: "medium",
			},
			mockSetup: func(m *MockMemoUsecase) {
				m.On("CreateMemo", mock.Anything, 1, mock.AnythingOfType("usecase.CreateMemoRequest")).Return(&domain.Memo{
					ID:       1,
					UserID:   1,
					Title:    "Test Memo",
					Content:  "This is a test memo",
					Category: "Test",
					Priority: domain.PriorityMedium,
					Status:   domain.StatusActive,
				}, nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "invalid request body",
			requestBody:    "invalid json",
			mockSetup:      func(m *MockMemoUsecase) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUsecase := new(MockMemoUsecase)
			tt.mockSetup(mockUsecase)

			router := setupTestRouter(mockUsecase)

			var body []byte
			var err error
			if _, ok := tt.requestBody.(string); ok {
				body = []byte(tt.requestBody.(string))
			} else {
				body, err = json.Marshal(tt.requestBody)
				assert.NoError(t, err)
			}

			req, _ := http.NewRequest("POST", "/api/memos", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusCreated {
				var response domain.Memo
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, "Test Memo", response.Title)
			}

			mockUsecase.AssertExpectations(t)
		})
	}
}

func TestMemoHandler_GetMemo(t *testing.T) {
	tests := []struct {
		name           string
		memoID         string
		mockSetup      func(*MockMemoUsecase)
		expectedStatus int
	}{
		{
			name:   "successful get",
			memoID: "1",
			mockSetup: func(m *MockMemoUsecase) {
				m.On("GetMemo", mock.Anything, 1, 1).Return(&domain.Memo{
					ID:      1,
					UserID:  1,
					Title:   "Test Memo",
					Content: "This is a test memo",
					Status:  domain.StatusActive,
				}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid memo ID",
			memoID:         "invalid",
			mockSetup:      func(m *MockMemoUsecase) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "memo not found",
			memoID: "999",
			mockSetup: func(m *MockMemoUsecase) {
				m.On("GetMemo", mock.Anything, 1, 999).Return(nil, usecase.ErrMemoNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUsecase := new(MockMemoUsecase)
			tt.mockSetup(mockUsecase)

			router := setupTestRouter(mockUsecase)

			req, _ := http.NewRequest("GET", "/api/memos/"+tt.memoID, nil)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response domain.Memo
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, "Test Memo", response.Title)
			}

			mockUsecase.AssertExpectations(t)
		})
	}
}

func TestMemoHandler_ListMemos(t *testing.T) {
	mockUsecase := new(MockMemoUsecase)

	mockUsecase.On("ListMemos", mock.Anything, 1, mock.AnythingOfType("domain.MemoFilter")).Return([]domain.Memo{
		{
			ID:      1,
			UserID:  1,
			Title:   "Test Memo 1",
			Content: "Content 1",
			Status:  domain.StatusActive,
		},
		{
			ID:      2,
			UserID:  1,
			Title:   "Test Memo 2",
			Content: "Content 2",
			Status:  domain.StatusActive,
		},
	}, 2, nil)

	router := setupTestRouter(mockUsecase)

	req, _ := http.NewRequest("GET", "/api/memos", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// レスポンス形式をチェック（実際のレスポンス構造に合わせて調整）
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	mockUsecase.AssertExpectations(t)
}

func TestMemoHandler_UpdateMemo(t *testing.T) {
	tests := []struct {
		name           string
		memoID         string
		requestBody    interface{}
		mockSetup      func(*MockMemoUsecase)
		expectedStatus int
	}{
		{
			name:   "successful update",
			memoID: "1",
			requestBody: usecase.UpdateMemoRequest{
				Title:   stringPtr("Updated Title"),
				Content: stringPtr("Updated Content"),
			},
			mockSetup: func(m *MockMemoUsecase) {
				m.On("UpdateMemo", mock.Anything, 1, 1, mock.AnythingOfType("usecase.UpdateMemoRequest")).Return(&domain.Memo{
					ID:        1,
					UserID:    1,
					Title:     "Updated Title",
					Content:   "Updated Content",
					Status:    domain.StatusActive,
					UpdatedAt: time.Now(),
				}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid memo ID",
			memoID:         "invalid",
			requestBody:    map[string]string{"title": "test"},
			mockSetup:      func(m *MockMemoUsecase) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid request body",
			memoID:         "1",
			requestBody:    "invalid json",
			mockSetup:      func(m *MockMemoUsecase) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "memo not found",
			memoID: "999",
			requestBody: usecase.UpdateMemoRequest{
				Title: stringPtr("Updated Title"),
			},
			mockSetup: func(m *MockMemoUsecase) {
				m.On("UpdateMemo", mock.Anything, 1, 999, mock.AnythingOfType("usecase.UpdateMemoRequest")).Return(nil, usecase.ErrMemoNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUsecase := new(MockMemoUsecase)
			tt.mockSetup(mockUsecase)

			router := setupTestRouter(mockUsecase)

			var body []byte
			var err error
			if _, ok := tt.requestBody.(string); ok {
				body = []byte(tt.requestBody.(string))
			} else {
				body, err = json.Marshal(tt.requestBody)
				assert.NoError(t, err)
			}

			req, _ := http.NewRequest("PUT", "/api/memos/"+tt.memoID, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response domain.Memo
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, "Updated Title", response.Title)
			}

			mockUsecase.AssertExpectations(t)
		})
	}
}

func TestMemoHandler_DeleteMemo(t *testing.T) {
	tests := []struct {
		name           string
		memoID         string
		mockSetup      func(*MockMemoUsecase)
		expectedStatus int
	}{
		{
			name:   "successful delete",
			memoID: "1",
			mockSetup: func(m *MockMemoUsecase) {
				m.On("DeleteMemo", mock.Anything, 1, 1).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "invalid memo ID",
			memoID:         "invalid",
			mockSetup:      func(m *MockMemoUsecase) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "memo not found",
			memoID: "999",
			mockSetup: func(m *MockMemoUsecase) {
				m.On("DeleteMemo", mock.Anything, 1, 999).Return(usecase.ErrMemoNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUsecase := new(MockMemoUsecase)
			tt.mockSetup(mockUsecase)

			router := setupTestRouter(mockUsecase)

			req, _ := http.NewRequest("DELETE", "/api/memos/"+tt.memoID, nil)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			mockUsecase.AssertExpectations(t)
		})
	}
}

func TestMemoHandler_SearchMemos(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		mockSetup      func(*MockMemoUsecase)
		expectedStatus int
	}{
		{
			name:        "successful search",
			queryParams: "?search=test&limit=10&page=1",
			mockSetup: func(m *MockMemoUsecase) {
				m.On("SearchMemos", mock.Anything, 1, "test", mock.AnythingOfType("domain.MemoFilter")).Return([]domain.Memo{
					{
						ID:      1,
						UserID:  1,
						Title:   "Test Memo",
						Content: "Test content",
						Status:  domain.StatusActive,
					},
				}, 1, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "empty search query",
			queryParams: "?search=",
			mockSetup: func(m *MockMemoUsecase) {
				m.On("SearchMemos", mock.Anything, 1, "", mock.AnythingOfType("domain.MemoFilter")).Return([]domain.Memo{}, 0, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "search with invalid limit",
			queryParams:    "?search=test&limit=invalid",
			mockSetup:      func(m *MockMemoUsecase) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUsecase := new(MockMemoUsecase)
			tt.mockSetup(mockUsecase)

			router := setupTestRouter(mockUsecase)

			req, _ := http.NewRequest("GET", "/api/memos/search"+tt.queryParams, nil)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			mockUsecase.AssertExpectations(t)
		})
	}
}

func TestMemoHandler_PermanentDeleteMemo(t *testing.T) {
	tests := []struct {
		name           string
		memoID         string
		mockSetup      func(*MockMemoUsecase)
		expectedStatus int
	}{
		{
			name:   "successful permanent delete",
			memoID: "1",
			mockSetup: func(m *MockMemoUsecase) {
				m.On("PermanentDeleteMemo", mock.Anything, 1, 1).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "invalid memo ID",
			memoID:         "invalid",
			mockSetup:      func(m *MockMemoUsecase) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "memo not found",
			memoID: "999",
			mockSetup: func(m *MockMemoUsecase) {
				m.On("PermanentDeleteMemo", mock.Anything, 1, 999).Return(usecase.ErrMemoNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUsecase := new(MockMemoUsecase)
			tt.mockSetup(mockUsecase)

			router := setupTestRouter(mockUsecase)

			req, _ := http.NewRequest("DELETE", "/api/memos/"+tt.memoID+"/permanent", nil)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			mockUsecase.AssertExpectations(t)
		})
	}
}
