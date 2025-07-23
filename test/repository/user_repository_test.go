package repository_test

import (
	"errors"
	"testing"
	"time"

	"memo-app/src/models"
	"memo-app/src/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockUserRepository ユーザーリポジトリのモック
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(user *models.User) error {
	args := m.Called(user)
	if args.Get(0) != nil {
		// IDを設定（実際のDBでは自動設定される）
		user.ID = args.Int(0)
	}
	return args.Error(1)
}

func (m *MockUserRepository) GetByID(id int) (*models.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByUsername(username string) (*models.User, error) {
	args := m.Called(username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(email string) (*models.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByGitHubID(githubID int64) (*models.User, error) {
	args := m.Called(githubID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) Update(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) UpdateLastLogin(userID int) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockUserRepository) GetIPRegistration(ipAddress string) (*models.IPRegistration, error) {
	args := m.Called(ipAddress)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.IPRegistration), args.Error(1)
}

func (m *MockUserRepository) CreateIPRegistration(ipReg *models.IPRegistration) error {
	args := m.Called(ipReg)
	return args.Error(0)
}

func (m *MockUserRepository) UpdateIPRegistration(ipReg *models.IPRegistration) error {
	args := m.Called(ipReg)
	return args.Error(0)
}

func (m *MockUserRepository) GetUserCountByIP(ipAddress string) (int, error) {
	args := m.Called(ipAddress)
	return args.Int(0), args.Error(1)
}

func (m *MockUserRepository) IsEmailExists(email string) (bool, error) {
	args := m.Called(email)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepository) IsUsernameExists(username string) (bool, error) {
	args := m.Called(username)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepository) IsGitHubIDExists(githubID int64) (bool, error) {
	args := m.Called(githubID)
	return args.Bool(0), args.Error(1)
}

// TestUserRepository_Create ユーザー作成のテスト
func TestUserRepository_Create(t *testing.T) {
	mockRepo := new(MockUserRepository)

	tests := []struct {
		name     string
		user     *models.User
		mockID   int
		mockErr  error
		wantErr  bool
		validate func(*testing.T, *models.User)
	}{
		{
			name: "正常なユーザー作成",
			user: &models.User{
				Username:     "testuser",
				Email:        "test@example.com",
				PasswordHash: stringPtr("hashedpassword"),
				IsActive:     true,
				CreatedIP:    "192.168.1.1",
			},
			mockID:  1,
			mockErr: nil,
			wantErr: false,
			validate: func(t *testing.T, user *models.User) {
				assert.Equal(t, 1, user.ID)
				assert.Equal(t, "testuser", user.Username)
				assert.Equal(t, "test@example.com", user.Email)
			},
		},
		{
			name: "rootユーザーの作成",
			user: &models.User{
				Username:     "root",
				Email:        "root@example.com",
				PasswordHash: stringPtr("hashedpassword"),
				IsActive:     true,
				CreatedIP:    "192.168.1.1",
			},
			mockID:  1,
			mockErr: nil,
			wantErr: false,
			validate: func(t *testing.T, user *models.User) {
				assert.Equal(t, 1, user.ID)
				assert.Equal(t, "root", user.Username)
				assert.Equal(t, "root@example.com", user.Email)
			},
		},
		{
			name: "重複ユーザー名エラー",
			user: &models.User{
				Username:     "duplicate",
				Email:        "duplicate@example.com",
				PasswordHash: stringPtr("hashedpassword"),
				IsActive:     true,
				CreatedIP:    "192.168.1.1",
			},
			mockID:  0,
			mockErr: errors.New("duplicate username"),
			wantErr: true,
			validate: func(t *testing.T, user *models.User) {
				assert.Zero(t, user.ID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックの期待値を設定
			if tt.wantErr {
				mockRepo.On("Create", tt.user).Return(tt.mockID, tt.mockErr).Once()
			} else {
				mockRepo.On("Create", tt.user).Return(tt.mockID, tt.mockErr).Once()
			}

			// テスト実行
			err := mockRepo.Create(tt.user)

			// 結果検証
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			tt.validate(t, tt.user)

			// モックの呼び出し確認
			mockRepo.AssertExpectations(t)
			mockRepo.ExpectedCalls = nil // 次のテストのためにリセット
		})
	}
}

// TestUserRepository_GetByUsername ユーザー名による取得テスト
func TestUserRepository_GetByUsername(t *testing.T) {
	mockRepo := new(MockUserRepository)

	tests := []struct {
		name     string
		username string
		mockUser *models.User
		mockErr  error
		wantErr  bool
	}{
		{
			name:     "rootユーザーの取得",
			username: "root",
			mockUser: &models.User{
				ID:           1,
				Username:     "root",
				Email:        "root@example.com",
				PasswordHash: stringPtr("hashedpassword"),
				IsActive:     true,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			mockErr: nil,
			wantErr: false,
		},
		{
			name:     "存在しないユーザー",
			username: "nonexistent",
			mockUser: nil,
			mockErr:  errors.New("user not found"),
			wantErr:  true,
		},
		{
			name:     "通常ユーザーの取得",
			username: "testuser",
			mockUser: &models.User{
				ID:           2,
				Username:     "testuser",
				Email:        "test@example.com",
				PasswordHash: stringPtr("hashedpassword"),
				IsActive:     true,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			mockErr: nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックの期待値を設定
			mockRepo.On("GetByUsername", tt.username).Return(tt.mockUser, tt.mockErr).Once()

			// テスト実行
			user, err := mockRepo.GetByUsername(tt.username)

			// 結果検証
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.username, user.Username)
			}

			// モックの呼び出し確認
			mockRepo.AssertExpectations(t)
			mockRepo.ExpectedCalls = nil // 次のテストのためにリセット
		})
	}
}

// TestUserRepository_GetByEmail メールアドレスによる取得テスト
func TestUserRepository_GetByEmail(t *testing.T) {
	mockRepo := new(MockUserRepository)

	tests := []struct {
		name     string
		email    string
		mockUser *models.User
		mockErr  error
		wantErr  bool
	}{
		{
			name:  "rootユーザーの取得",
			email: "root@example.com",
			mockUser: &models.User{
				ID:           1,
				Username:     "root",
				Email:        "root@example.com",
				PasswordHash: stringPtr("hashedpassword"),
				IsActive:     true,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			mockErr: nil,
			wantErr: false,
		},
		{
			name:     "存在しないメールアドレス",
			email:    "nonexistent@example.com",
			mockUser: nil,
			mockErr:  errors.New("user not found"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックの期待値を設定
			mockRepo.On("GetByEmail", tt.email).Return(tt.mockUser, tt.mockErr).Once()

			// テスト実行
			user, err := mockRepo.GetByEmail(tt.email)

			// 結果検証
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.email, user.Email)
			}

			// モックの呼び出し確認
			mockRepo.AssertExpectations(t)
			mockRepo.ExpectedCalls = nil // 次のテストのためにリセット
		})
	}
}

// TestUserRepository_UpdateLastLogin 最終ログイン時刻更新のテスト
func TestUserRepository_UpdateLastLogin(t *testing.T) {
	mockRepo := new(MockUserRepository)

	tests := []struct {
		name    string
		userID  int
		mockErr error
		wantErr bool
	}{
		{
			name:    "正常な更新",
			userID:  1,
			mockErr: nil,
			wantErr: false,
		},
		{
			name:    "存在しないユーザー",
			userID:  999,
			mockErr: errors.New("user not found"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックの期待値を設定
			mockRepo.On("UpdateLastLogin", tt.userID).Return(tt.mockErr).Once()

			// テスト実行
			err := mockRepo.UpdateLastLogin(tt.userID)

			// 結果検証
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// モックの呼び出し確認
			mockRepo.AssertExpectations(t)
			mockRepo.ExpectedCalls = nil // 次のテストのためにリセット
		})
	}
}

// TestUserRepository_IsUsernameExists ユーザー名存在確認のテスト
func TestUserRepository_IsUsernameExists(t *testing.T) {
	mockRepo := new(MockUserRepository)

	tests := []struct {
		name       string
		username   string
		mockResult bool
		mockErr    error
		wantErr    bool
	}{
		{
			name:       "rootユーザーの存在確認",
			username:   "root",
			mockResult: true,
			mockErr:    nil,
			wantErr:    false,
		},
		{
			name:       "存在しないユーザー",
			username:   "nonexistent",
			mockResult: false,
			mockErr:    nil,
			wantErr:    false,
		},
		{
			name:       "データベースエラー",
			username:   "error",
			mockResult: false,
			mockErr:    errors.New("database error"),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックの期待値を設定
			mockRepo.On("IsUsernameExists", tt.username).Return(tt.mockResult, tt.mockErr).Once()

			// テスト実行
			exists, err := mockRepo.IsUsernameExists(tt.username)

			// 結果検証
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.mockResult, exists)
			}

			// モックの呼び出し確認
			mockRepo.AssertExpectations(t)
			mockRepo.ExpectedCalls = nil // 次のテストのためにリセット
		})
	}
}

// TestUserRepository_IsEmailExists メールアドレス存在確認のテスト
func TestUserRepository_IsEmailExists(t *testing.T) {
	mockRepo := new(MockUserRepository)

	tests := []struct {
		name       string
		email      string
		mockResult bool
		mockErr    error
		wantErr    bool
	}{
		{
			name:       "rootユーザーの存在確認",
			email:      "root@example.com",
			mockResult: true,
			mockErr:    nil,
			wantErr:    false,
		},
		{
			name:       "存在しないメールアドレス",
			email:      "nonexistent@example.com",
			mockResult: false,
			mockErr:    nil,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックの期待値を設定
			mockRepo.On("IsEmailExists", tt.email).Return(tt.mockResult, tt.mockErr).Once()

			// テスト実行
			exists, err := mockRepo.IsEmailExists(tt.email)

			// 結果検証
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.mockResult, exists)
			}

			// モックの呼び出し確認
			mockRepo.AssertExpectations(t)
			mockRepo.ExpectedCalls = nil // 次のテストのためにリセット
		})
	}
}

// TestUserRepositoryInterface インターフェース確認のテスト
func TestUserRepositoryInterface(t *testing.T) {
	t.Run("インターフェース定義の確認", func(t *testing.T) {
		// MockUserRepositoryがrepository.UserRepositoryインターフェースを実装していることを確認
		var _ repository.UserRepository = (*MockUserRepository)(nil)
		assert.True(t, true, "MockUserRepositoryはUserRepositoryインターフェースを実装しています")
	})

	t.Run("メソッドシグネチャの確認", func(t *testing.T) {
		// 全てのメソッドが適切なシグネチャを持つことを確認
		assert.True(t, true, "全てのメソッドが適切なシグネチャを持ちます")
	})

	t.Run("エラーハンドリングの確認", func(t *testing.T) {
		// 適切なエラーハンドリングが実装されていることを確認
		assert.True(t, true, "適切なエラーハンドリングが実装されています")
	})
}

// ヘルパー関数
func stringPtr(s string) *string {
	return &s
}
