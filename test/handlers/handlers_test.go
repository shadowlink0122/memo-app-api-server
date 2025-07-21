package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"memo-app/src/domain"
	"memo-app/src/interface/handler"
	"memo-app/src/usecase"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockMemoUsecase は MemoUsecase のモック実装
type MockMemoUsecase struct {
	mock.Mock
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
	gin.SetMode(gin.TestMode)
	r := gin.New()

	logger := logrus.New()
	memoHandler := handler.NewMemoHandler(mockUsecase, logger)

	// ルートの設定
	api := r.Group("/api/memos")
	{
		api.POST("", memoHandler.CreateMemo)
		api.GET("", memoHandler.ListMemos)
		api.GET("/:id", memoHandler.GetMemo)
		api.PUT("/:id", memoHandler.UpdateMemo)
		api.DELETE("/:id", memoHandler.DeleteMemo)
		api.PATCH("/:id/archive", memoHandler.ArchiveMemo)
		api.PATCH("/:id/restore", memoHandler.RestoreMemo)
		api.GET("/search", memoHandler.SearchMemos)
	}

	return r
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
				m.On("CreateMemo", mock.Anything, mock.AnythingOfType("usecase.CreateMemoRequest")).Return(&domain.Memo{
					ID:       1,
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
				m.On("GetMemo", mock.Anything, 1).Return(&domain.Memo{
					ID:      1,
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
				m.On("GetMemo", mock.Anything, 999).Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
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

	mockUsecase.On("ListMemos", mock.Anything, mock.AnythingOfType("domain.MemoFilter")).Return([]domain.Memo{
		{
			ID:      1,
			Title:   "Test Memo 1",
			Content: "Content 1",
			Status:  domain.StatusActive,
		},
		{
			ID:      2,
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
