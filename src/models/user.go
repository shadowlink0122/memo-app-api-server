package models

import (
	"time"
)

// User ユーザーモデル
type User struct {
	ID             int        `json:"id" db:"id"`
	Username       string     `json:"username" db:"username"`
	Email          string     `json:"email" db:"email"`
	PasswordHash   *string    `json:"-" db:"password_hash"`     // ローカル認証用（JSON出力しない）
	GitHubID       *int64     `json:"github_id" db:"github_id"` // GitHub認証用
	GitHubUsername *string    `json:"github_username" db:"github_username"`
	AvatarURL      *string    `json:"avatar_url" db:"avatar_url"`
	IsActive       bool       `json:"is_active" db:"is_active"`
	LastLoginAt    *time.Time `json:"last_login_at" db:"last_login_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
	CreatedIP      string     `json:"created_ip" db:"created_ip"` // 作成時のIPアドレス
}

// PublicUser 公開用ユーザー情報（センシティブな情報を除外）
type PublicUser struct {
	ID             int       `json:"id"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	GitHubUsername *string   `json:"github_username,omitempty"`
	AvatarURL      *string   `json:"avatar_url,omitempty"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
}

// ToPublic センシティブな情報を除外したPublicUserを返す
func (u *User) ToPublic() *PublicUser {
	return &PublicUser{
		ID:             u.ID,
		Username:       u.Username,
		Email:          u.Email,
		GitHubUsername: u.GitHubUsername,
		AvatarURL:      u.AvatarURL,
		IsActive:       u.IsActive,
		CreatedAt:      u.CreatedAt,
	}
}

// AuthProvider 認証プロバイダーの種類
type AuthProvider string

const (
	AuthProviderLocal  AuthProvider = "local"
	AuthProviderGitHub AuthProvider = "github"
)

// GetAuthProvider ユーザーの認証プロバイダーを取得
func (u *User) GetAuthProvider() AuthProvider {
	if u.GitHubID != nil {
		return AuthProviderGitHub
	}
	return AuthProviderLocal
}

// IPRegistration IP制限用のモデル
type IPRegistration struct {
	ID         int       `json:"id" db:"id"`
	IPAddress  string    `json:"ip_address" db:"ip_address"`
	UserCount  int       `json:"user_count" db:"user_count"`
	LastUsedAt time.Time `json:"last_used_at" db:"last_used_at"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// LoginRequest ログインリクエスト
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" validate:"required,email"`
	Password string `json:"password" binding:"required,min=8" validate:"required,min=8"`
}

// RegisterRequest 新規登録リクエスト（ローカル認証）
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50" validate:"required,min=3,max=50,safe_text"`
	Email    string `json:"email" binding:"required,email" validate:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=128" validate:"required,min=8,max=128,password_strength"`
}

// GitHubAuthRequest GitHub認証リクエスト
type GitHubAuthRequest struct {
	Code  string `json:"code" binding:"required" validate:"required"`
	State string `json:"state" binding:"required" validate:"required"`
}

// AuthResponse 認証レスポンス
type AuthResponse struct {
	User         *PublicUser `json:"user"`
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	ExpiresIn    int64       `json:"expires_in"` // 秒単位
}

// RefreshTokenRequest リフレッシュトークンリクエスト
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required" validate:"required"`
}

// GitHubUser GitHub APIから取得するユーザー情報
type GitHubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}
