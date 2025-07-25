package usecase_test

import (
	"context"
	"testing"
	"time"

	"memo-app/src/domain"
	"memo-app/src/usecase"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockMemoRepository は domain.MemoRepository のモック実装
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

func (m *MockMemoRepository) GetByID(ctx context.Context, id int, userID int) (*domain.Memo, error) {
	args := m.Called(ctx, id, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Memo), args.Error(1)
}

func (m *MockMemoRepository) List(ctx context.Context, userID int, filter domain.MemoFilter) ([]domain.Memo, int, error) {
	args := m.Called(ctx, userID, filter)
	return args.Get(0).([]domain.Memo), args.Get(1).(int), args.Error(2)
}

func (m *MockMemoRepository) Update(ctx context.Context, id int, userID int, memo *domain.Memo) (*domain.Memo, error) {
	args := m.Called(ctx, id, userID, memo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Memo), args.Error(1)
}

func (m *MockMemoRepository) Delete(ctx context.Context, id int, userID int) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockMemoRepository) Archive(ctx context.Context, id int, userID int) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockMemoRepository) Restore(ctx context.Context, id int, userID int) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockMemoRepository) Search(ctx context.Context, userID int, query string, filter domain.MemoFilter) ([]domain.Memo, int, error) {
	args := m.Called(ctx, userID, query, filter)
	return args.Get(0).([]domain.Memo), args.Get(1).(int), args.Error(2)
}

// PermanentDelete implements domain.MemoRepository.
func (m *MockMemoRepository) PermanentDelete(ctx context.Context, id int, userID int) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func TestMemoUsecase_CreateMemo(t *testing.T) {
	tests := []struct {
		name          string
		request       usecase.CreateMemoRequest
		mockSetup     func(*MockMemoRepository)
		expectedError bool
		errorMsg      string
	}{
		{
			name: "successful creation",
			request: usecase.CreateMemoRequest{
				Title:    "Test Memo",
				Content:  "This is a test memo content",
				Category: "Work",
				Tags:     []string{"test", "work"},
				Priority: "medium",
			},
			mockSetup: func(m *MockMemoRepository) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*domain.Memo")).Return(&domain.Memo{
					ID:        1,
					Title:     "Test Memo",
					Content:   "This is a test memo content",
					Category:  "Work",
					Tags:      []string{"test", "work"},
					Priority:  domain.PriorityMedium,
					Status:    domain.StatusActive,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}, nil)
			},
			expectedError: false,
		},
		{
			name: "invalid title - empty",
			request: usecase.CreateMemoRequest{
				Title:   "",
				Content: "Content",
			},
			mockSetup:     func(m *MockMemoRepository) {},
			expectedError: true,
			errorMsg:      "title is required",
		},
		{
			name: "invalid title - too long",
			request: usecase.CreateMemoRequest{
				Title:   string(make([]byte, 201)), // 201文字のタイトル
				Content: "Content",
			},
			mockSetup:     func(m *MockMemoRepository) {},
			expectedError: true,
			errorMsg:      "title is required and must be less than 200 characters",
		},
		{
			name: "invalid content - empty",
			request: usecase.CreateMemoRequest{
				Title:   "Valid Title",
				Content: "",
			},
			mockSetup:     func(m *MockMemoRepository) {},
			expectedError: true,
			errorMsg:      "content is required",
		},
		{
			name: "invalid priority",
			request: usecase.CreateMemoRequest{
				Title:    "Valid Title",
				Content:  "Valid Content",
				Priority: "invalid",
			},
			mockSetup:     func(m *MockMemoRepository) {},
			expectedError: true,
			errorMsg:      "priority must be low, medium, or high",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockMemoRepository)
			tt.mockSetup(mockRepo)

			uc := usecase.NewMemoUsecase(mockRepo)

			// Act
			result, err := uc.CreateMemo(context.Background(), 1, tt.request)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.request.Title, result.Title)
				assert.Equal(t, tt.request.Content, result.Content)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestMemoUsecase_GetMemo(t *testing.T) {
	tests := []struct {
		name          string
		memoID        int
		mockSetup     func(*MockMemoRepository)
		expectedError bool
	}{
		{
			name:   "successful get",
			memoID: 1,
			mockSetup: func(m *MockMemoRepository) {
				m.On("GetByID", mock.Anything, 1, 1).Return(&domain.Memo{
					ID:      1,
					Title:   "Test Memo",
					Content: "Test Content",
					Status:  domain.StatusActive,
				}, nil)
			},
			expectedError: false,
		},
		{
			name:   "memo not found",
			memoID: 999,
			mockSetup: func(m *MockMemoRepository) {
				m.On("GetByID", mock.Anything, 999, 1).Return(nil, assert.AnError)
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockMemoRepository)
			tt.mockSetup(mockRepo)

			uc := usecase.NewMemoUsecase(mockRepo)

			result, err := uc.GetMemo(context.Background(), 1, tt.memoID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.memoID, result.ID)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestMemoUsecase_ListMemos(t *testing.T) {
	mockRepo := new(MockMemoRepository)

	expectedMemos := []domain.Memo{
		{
			ID:      1,
			Title:   "Memo 1",
			Content: "Content 1",
			Status:  domain.StatusActive,
		},
		{
			ID:      2,
			Title:   "Memo 2",
			Content: "Content 2",
			Status:  domain.StatusActive,
		},
	}

	filter := domain.MemoFilter{
		Page:  1,
		Limit: 10,
	}

	mockRepo.On("List", mock.Anything, 1, filter).Return(expectedMemos, 2, nil)

	uc := usecase.NewMemoUsecase(mockRepo)

	result, total, err := uc.ListMemos(context.Background(), 1, filter)

	assert.NoError(t, err)
	assert.Equal(t, expectedMemos, result)
	assert.Equal(t, 2, total)

	mockRepo.AssertExpectations(t)
}

func TestMemoUsecase_ArchiveMemo(t *testing.T) {
	tests := []struct {
		name          string
		memoID        int
		mockSetup     func(*MockMemoRepository)
		expectedError bool
	}{
		{
			name:   "successful archive",
			memoID: 1,
			mockSetup: func(m *MockMemoRepository) {
				m.On("Archive", mock.Anything, 1, 1).Return(nil)
			},
			expectedError: false,
		},
		{
			name:   "memo not found",
			memoID: 999,
			mockSetup: func(m *MockMemoRepository) {
				m.On("Archive", mock.Anything, 999, 1).Return(assert.AnError)
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockMemoRepository)
			tt.mockSetup(mockRepo)

			uc := usecase.NewMemoUsecase(mockRepo)

			err := uc.ArchiveMemo(context.Background(), 1, tt.memoID)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestMemoUsecase_RestoreMemo(t *testing.T) {
	tests := []struct {
		name          string
		memoID        int
		mockSetup     func(*MockMemoRepository)
		expectedError bool
	}{
		{
			name:   "successful restore",
			memoID: 1,
			mockSetup: func(m *MockMemoRepository) {
				m.On("Restore", mock.Anything, 1, 1).Return(nil)
			},
			expectedError: false,
		},
		{
			name:   "memo not found",
			memoID: 999,
			mockSetup: func(m *MockMemoRepository) {
				m.On("Restore", mock.Anything, 999, 1).Return(assert.AnError)
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockMemoRepository)
			tt.mockSetup(mockRepo)

			uc := usecase.NewMemoUsecase(mockRepo)

			err := uc.RestoreMemo(context.Background(), 1, tt.memoID)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
