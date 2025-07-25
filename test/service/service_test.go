package service_test

import (
	"context"
	"testing"

	"memo-app/src/models"
	"memo-app/src/service"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockMemoRepository は MemoRepository のモック実装
type MockMemoRepository struct {
	mock.Mock
}

func (m *MockMemoRepository) Create(ctx context.Context, req *models.CreateMemoRequest) (*models.Memo, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*models.Memo), args.Error(1)
}

func (m *MockMemoRepository) GetByID(ctx context.Context, id int) (*models.Memo, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Memo), args.Error(1)
}

func (m *MockMemoRepository) List(ctx context.Context, filter *models.MemoFilter) (*models.MemoListResponse, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(*models.MemoListResponse), args.Error(1)
}

func (m *MockMemoRepository) Update(ctx context.Context, id int, req *models.UpdateMemoRequest) (*models.Memo, error) {
	args := m.Called(ctx, id, req)
	return args.Get(0).(*models.Memo), args.Error(1)
}

func (m *MockMemoRepository) Delete(ctx context.Context, id int) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestMemoService_ValidateCreateRequest(t *testing.T) {
	logger := logrus.New()
	mockRepo := new(MockMemoRepository)
	service := service.NewMemoService(mockRepo, logger)

	tests := []struct {
		name    string
		request *models.CreateMemoRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			request: &models.CreateMemoRequest{
				Title:    "Test Memo",
				Content:  "This is a test memo",
				Category: "Test",
				Priority: "medium",
			},
			wantErr: false,
		},
		{
			name: "empty title",
			request: &models.CreateMemoRequest{
				Title:   "",
				Content: "This is a test memo",
			},
			wantErr: true,
			errMsg:  "title is required",
		},
		{
			name: "empty content",
			request: &models.CreateMemoRequest{
				Title:   "Test Memo",
				Content: "",
			},
			wantErr: true,
			errMsg:  "content is required",
		},
		{
			name: "title too long",
			request: &models.CreateMemoRequest{
				Title:   "This is a very long title that exceeds the maximum allowed length of 200 characters. Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.",
				Content: "Test content",
			},
			wantErr: true,
			errMsg:  "title must be at most 200 characters",
		},
		{
			name: "invalid priority",
			request: &models.CreateMemoRequest{
				Title:    "Test Memo",
				Content:  "Test content",
				Priority: "invalid",
			},
			wantErr: true,
			errMsg:  "priority must be one of: low, medium, high",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// CreateMemo は内部でバリデーションを実行するので、実際に呼び出してテスト
			if !tt.wantErr {
				// 成功ケースのモック設定
				mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(req *models.CreateMemoRequest) bool {
					return req.Title == tt.request.Title && req.Content == tt.request.Content
				})).Return(&models.Memo{
					ID:       1,
					Title:    tt.request.Title,
					Content:  tt.request.Content,
					Category: tt.request.Category,
					Priority: "medium", // デフォルト値
					Status:   "active",
				}, nil).Once()
			}

			_, err := service.CreateMemo(context.Background(), tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}

	mockRepo.AssertExpectations(t)
}

func TestMemoService_NormalizeTags(t *testing.T) {
	logger := logrus.New()
	mockRepo := new(MockMemoRepository)
	service := service.NewMemoService(mockRepo, logger)

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "remove empty tags",
			input:    []string{"tag1", "", "tag2", "  ", "tag3"},
			expected: []string{"tag1", "tag2", "tag3"},
		},
		{
			name:     "remove duplicate tags",
			input:    []string{"tag1", "tag2", "tag1", "tag3", "tag2"},
			expected: []string{"tag1", "tag2", "tag3"},
		},
		{
			name:     "trim whitespace",
			input:    []string{"  tag1  ", "tag2", "  tag3"},
			expected: []string{"tag1", "tag2", "tag3"},
		},
		{
			name:     "empty input",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "all empty tags",
			input:    []string{"", "  ", ""},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 正規化をテストするため、有効なリクエストを作成してCreateMemoを呼び出す
			request := &models.CreateMemoRequest{
				Title:   "Test",
				Content: "Test content",
				Tags:    tt.input,
			}

			mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(req *models.CreateMemoRequest) bool {
				// 正規化されたタグをチェック
				if len(req.Tags) != len(tt.expected) {
					return false
				}
				for i, tag := range req.Tags {
					if tag != tt.expected[i] {
						return false
					}
				}
				return true
			})).Return(&models.Memo{
				ID:      1,
				Title:   "Test",
				Content: "Test content",
				Status:  "active",
			}, nil).Once()

			_, err := service.CreateMemo(context.Background(), request)
			assert.NoError(t, err)
		})
	}

	mockRepo.AssertExpectations(t)
}

func TestMemoService_ValidateAndNormalizeFilter(t *testing.T) {
	logger := logrus.New()
	mockRepo := new(MockMemoRepository)
	service := service.NewMemoService(mockRepo, logger)

	tests := []struct {
		name           string
		input          *models.MemoFilter
		expectedPage   int
		expectedLimit  int
		expectedStatus string
		wantErr        bool
	}{
		{
			name: "set default page and limit",
			input: &models.MemoFilter{
				Page:  0,
				Limit: 0,
			},
			expectedPage:  1,
			expectedLimit: 10,
			wantErr:       false,
		},
		{
			name: "limit too high",
			input: &models.MemoFilter{
				Page:  1,
				Limit: 200,
			},
			expectedPage:  1,
			expectedLimit: 100,
			wantErr:       false,
		},
		{
			name: "valid status",
			input: &models.MemoFilter{
				Status: "active",
			},
			expectedPage:   1,
			expectedLimit:  10,
			expectedStatus: "active",
			wantErr:        false,
		},
		{
			name: "invalid status",
			input: &models.MemoFilter{
				Status: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.wantErr {
				mockRepo.On("List", mock.Anything, mock.MatchedBy(func(filter *models.MemoFilter) bool {
					return filter.Page == tt.expectedPage &&
						filter.Limit == tt.expectedLimit &&
						(tt.expectedStatus == "" || filter.Status == tt.expectedStatus)
				})).Return(&models.MemoListResponse{
					Memos:      []models.Memo{},
					Total:      0,
					Page:       tt.expectedPage,
					Limit:      tt.expectedLimit,
					TotalPages: 0,
				}, nil).Once()
			}

			_, err := service.ListMemos(context.Background(), tt.input)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedPage, tt.input.Page)
				assert.Equal(t, tt.expectedLimit, tt.input.Limit)
			}
		})
	}

	mockRepo.AssertExpectations(t)
}
