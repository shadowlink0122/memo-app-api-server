package repository

import (
	"database/sql"
	"fmt"
	"time"

	"memo-app/src/models"
)

// UserRepository ユーザーデータアクセス層のインターフェース
type UserRepository interface {
	// ユーザー管理
	Create(user *models.User) error
	GetByID(id int) (*models.User, error)
	GetByEmail(email string) (*models.User, error)
	GetByGitHubID(githubID int64) (*models.User, error)
	GetByUsername(username string) (*models.User, error)
	Update(user *models.User) error
	UpdateLastLogin(userID int) error

	// IP制限管理
	GetIPRegistration(ipAddress string) (*models.IPRegistration, error)
	CreateIPRegistration(ipReg *models.IPRegistration) error
	UpdateIPRegistration(ipReg *models.IPRegistration) error
	GetUserCountByIP(ipAddress string) (int, error)

	// セキュリティ
	IsEmailExists(email string) (bool, error)
	IsUsernameExists(username string) (bool, error)
	IsGitHubIDExists(githubID int64) (bool, error)
}

// userRepository ユーザーリポジトリの実装
type userRepository struct {
	db *sql.DB
}

// NewUserRepository ユーザーリポジトリを作成
func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

// Create ユーザーを作成
func (r *userRepository) Create(user *models.User) error {
	query := `
		INSERT INTO users (username, email, password_hash, github_id, github_username, avatar_url, is_active, created_ip, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRow(
		query,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.GitHubID,
		user.GitHubUsername,
		user.AvatarURL,
		user.IsActive,
		user.CreatedIP,
		time.Now(),
		time.Now(),
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetByID IDでユーザーを取得
func (r *userRepository) GetByID(id int) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, username, email, password_hash, github_id, github_username, avatar_url, 
		       is_active, last_login_at, created_at, updated_at, created_ip
		FROM users WHERE id = $1`

	err := r.db.QueryRow(query, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.GitHubID, &user.GitHubUsername, &user.AvatarURL,
		&user.IsActive, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt, &user.CreatedIP,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetByEmail メールアドレスでユーザーを取得
func (r *userRepository) GetByEmail(email string) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, username, email, password_hash, github_id, github_username, avatar_url, 
		       is_active, last_login_at, created_at, updated_at, created_ip
		FROM users WHERE email = $1`

	err := r.db.QueryRow(query, email).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.GitHubID, &user.GitHubUsername, &user.AvatarURL,
		&user.IsActive, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt, &user.CreatedIP,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetByGitHubID GitHub IDでユーザーを取得
func (r *userRepository) GetByGitHubID(githubID int64) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, username, email, password_hash, github_id, github_username, avatar_url, 
		       is_active, last_login_at, created_at, updated_at, created_ip
		FROM users WHERE github_id = $1`

	err := r.db.QueryRow(query, githubID).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.GitHubID, &user.GitHubUsername, &user.AvatarURL,
		&user.IsActive, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt, &user.CreatedIP,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetByUsername ユーザー名でユーザーを取得
func (r *userRepository) GetByUsername(username string) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, username, email, password_hash, github_id, github_username, avatar_url, 
		       is_active, last_login_at, created_at, updated_at, created_ip
		FROM users WHERE username = $1`

	err := r.db.QueryRow(query, username).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.GitHubID, &user.GitHubUsername, &user.AvatarURL,
		&user.IsActive, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt, &user.CreatedIP,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// Update ユーザー情報を更新
func (r *userRepository) Update(user *models.User) error {
	query := `
		UPDATE users 
		SET username = $2, email = $3, password_hash = $4, github_id = $5, 
		    github_username = $6, avatar_url = $7, is_active = $8, updated_at = $9
		WHERE id = $1`

	_, err := r.db.Exec(
		query,
		user.ID, user.Username, user.Email, user.PasswordHash,
		user.GitHubID, user.GitHubUsername, user.AvatarURL,
		user.IsActive, time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// UpdateLastLogin 最終ログイン時刻を更新
func (r *userRepository) UpdateLastLogin(userID int) error {
	query := `UPDATE users SET last_login_at = $1 WHERE id = $2`
	_, err := r.db.Exec(query, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}
	return nil
}

// GetIPRegistration IP登録情報を取得
func (r *userRepository) GetIPRegistration(ipAddress string) (*models.IPRegistration, error) {
	ipReg := &models.IPRegistration{}
	query := `
		SELECT id, ip_address, user_count, last_used_at, created_at, updated_at
		FROM ip_registrations WHERE ip_address = $1`

	err := r.db.QueryRow(query, ipAddress).Scan(
		&ipReg.ID, &ipReg.IPAddress, &ipReg.UserCount,
		&ipReg.LastUsedAt, &ipReg.CreatedAt, &ipReg.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // レコードが存在しない場合はnilを返す
		}
		return nil, fmt.Errorf("failed to get IP registration: %w", err)
	}

	return ipReg, nil
}

// CreateIPRegistration IP登録情報を作成
func (r *userRepository) CreateIPRegistration(ipReg *models.IPRegistration) error {
	query := `
		INSERT INTO ip_registrations (ip_address, user_count, last_used_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`

	err := r.db.QueryRow(
		query,
		ipReg.IPAddress,
		ipReg.UserCount,
		ipReg.LastUsedAt,
		time.Now(),
		time.Now(),
	).Scan(&ipReg.ID)

	if err != nil {
		return fmt.Errorf("failed to create IP registration: %w", err)
	}

	return nil
}

// UpdateIPRegistration IP登録情報を更新
func (r *userRepository) UpdateIPRegistration(ipReg *models.IPRegistration) error {
	query := `
		UPDATE ip_registrations 
		SET user_count = $2, last_used_at = $3, updated_at = $4
		WHERE id = $1`

	_, err := r.db.Exec(
		query,
		ipReg.ID,
		ipReg.UserCount,
		ipReg.LastUsedAt,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to update IP registration: %w", err)
	}

	return nil
}

// GetUserCountByIP 指定IPアドレスのユーザー数を取得
func (r *userRepository) GetUserCountByIP(ipAddress string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM users WHERE created_ip = $1`

	err := r.db.QueryRow(query, ipAddress).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get user count by IP: %w", err)
	}

	return count, nil
}

// IsEmailExists メールアドレスが既に存在するかチェック
func (r *userRepository) IsEmailExists(email string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM users WHERE email = $1`

	err := r.db.QueryRow(query, email).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}

	return count > 0, nil
}

// IsUsernameExists ユーザー名が既に存在するかチェック
func (r *userRepository) IsUsernameExists(username string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM users WHERE username = $1`

	err := r.db.QueryRow(query, username).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check username existence: %w", err)
	}

	return count > 0, nil
}

// IsGitHubIDExists GitHub IDが既に存在するかチェック
func (r *userRepository) IsGitHubIDExists(githubID int64) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM users WHERE github_id = $1`

	err := r.db.QueryRow(query, githubID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check GitHub ID existence: %w", err)
	}

	return count > 0, nil
}
