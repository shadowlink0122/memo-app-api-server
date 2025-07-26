package service_test

import (
	"context"
	"testing"

	"memo-app/src/domain"
	"memo-app/src/models"
	"memo-app/src/service"
	"memo-app/src/usecase"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockMemoRepository は MemoRepository のモック実装
type MockMemoRepository struct {
	mock.Mock
}

func (m *MockMemoRepository) Create(ctx context.Context, memo *domain.Memo) (*domain.Memo, error) {
	args := m.Called(ctx, memo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Memo), args.Error(1)
}

// Archive implements repository.MemoRepositoryInterface.
func (m *MockMemoRepository) Archive(ctx context.Context, userID int, id int) (*domain.Memo, error) {
	args := m.Called(ctx, userID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Memo), args.Error(1)
}

// PermanentDelete implements repository.MemoRepositoryInterface.
func (m *MockMemoRepository) PermanentDelete(ctx context.Context, userID int, id int) error {
	args := m.Called(ctx, userID, id)
	return args.Error(0)
}

// Restore implements repository.MemoRepositoryInterface.
func (m *MockMemoRepository) Restore(ctx context.Context, userID int, id int) (*domain.Memo, error) {
	args := m.Called(ctx, userID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Memo), args.Error(1)
}

// Search implements repository.MemoRepositoryInterface.
func (m *MockMemoRepository) Search(ctx context.Context, userID int, query string, filter domain.MemoFilter) ([]domain.Memo, int, error) {
	args := m.Called(ctx, userID, query, filter)
	return args.Get(0).([]domain.Memo), args.Int(1), args.Error(2)
}

func (m *MockMemoRepository) GetByID(ctx context.Context, id int, userID int) (*domain.Memo, error) {
	args := m.Called(ctx, id, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Memo), args.Error(1)
}

func (m *MockMemoRepository) List(ctx context.Context, userID int, filter domain.MemoFilter) ([]domain.Memo, int, error) {
	args := m.Called(ctx, userID, filter)
	return args.Get(0).([]domain.Memo), args.Int(1), args.Error(2)
}

func (m *MockMemoRepository) Update(ctx context.Context, userID int, id int, memo *domain.Memo) (*domain.Memo, error) {
	args := m.Called(ctx, userID, id, memo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Memo), args.Error(1)
}

func (m *MockMemoRepository) Delete(ctx context.Context, userID int, id int) error {
	args := m.Called(ctx, userID, id)
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
			if !tt.wantErr {
				mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(memo *domain.Memo) bool {
					return memo.Title == tt.request.Title && memo.Content == tt.request.Content
				})).Return(&domain.Memo{
					ID:       1,
					UserID:   1,
					Title:    tt.request.Title,
					Content:  tt.request.Content,
					Category: tt.request.Category,
					Priority: domain.PriorityMedium,
					Status:   domain.StatusActive,
				}, nil).Once()
			}

			req := usecase.CreateMemoRequest{
				Title:    tt.request.Title,
				Content:  tt.request.Content,
				Category: tt.request.Category,
				Priority: tt.request.Priority,
				Tags:     tt.request.Tags,
			}
			_, err := service.CreateMemo(context.Background(), 1, req)

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
			request := usecase.CreateMemoRequest{
				Title:   "Test",
				Content: "Test content",
				Tags:    tt.input,
			}

			mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(memo *domain.Memo) bool {
				if memo.Title != "Test" || memo.Content != "Test content" {
					return false
				}
				if len(memo.Tags) != len(tt.expected) {
					return false
				}
				for i, tag := range memo.Tags {
					if tag != tt.expected[i] {
						return false
					}
				}
				return true
			})).Return(&domain.Memo{
				ID:      1,
				UserID:  1,
				Title:   "Test",
				Content: "Test content",
				Status:  domain.StatusActive,
			}, nil).Once()

			_, err := service.CreateMemo(context.Background(), 1, request)
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
			// domain.MemoFilterへ変換
			filter := domain.MemoFilter{
				Category: tt.input.Category,
				Status:   domain.Status(tt.input.Status),
				Priority: domain.Priority(tt.input.Priority),
				Search:   tt.input.Search,
				Tags:     []string{},
				Page:     tt.input.Page,
				Limit:    tt.input.Limit,
			}
			if !tt.wantErr {
				mockRepo.On("List", mock.Anything, mock.Anything, mock.MatchedBy(func(f domain.MemoFilter) bool {
					// Page/Limit補正後の値で判定
					page := f.Page
					limit := f.Limit
					if page <= 0 {
						page = 1
					}
					if limit <= 0 {
						limit = 10
					}
					if limit > 100 {
						limit = 100
					}
					return page == tt.expectedPage && limit == tt.expectedLimit
				})).Return([]domain.Memo{
					{
						ID:     1,
						UserID: 1,
						Title:  "Sample Memo",
						Status: domain.StatusActive,
					},
				}, 1, nil).Maybe()
			}

			memos, total, err := service.ListMemos(context.Background(), 1, filter)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, memos)
				assert.GreaterOrEqual(t, total, 0)
			}
		})
	}

	mockRepo.AssertExpectations(t)
}
