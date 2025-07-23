package service

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"memo-app/src/config"
	"memo-app/src/models"
	"memo-app/src/repository"
)

// AuthService 認証サービスのインターフェース
type AuthService interface {
	// ローカル認証
	Register(req *models.RegisterRequest, clientIP string) (*models.AuthResponse, error)
	Login(req *models.LoginRequest, clientIP string) (*models.AuthResponse, error)

	// GitHub認証
	GetGitHubAuthURL(state string) string
	HandleGitHubCallback(code, state, clientIP string) (*models.AuthResponse, error)

	// トークン管理
	ValidateToken(tokenString string) (*models.User, error)
	RefreshToken(refreshToken string) (*models.AuthResponse, error)

	// IP制限チェック
	CheckIPLimit(clientIP string) error
}

// authService 認証サービスの実装
type authService struct {
	userRepo   repository.UserRepository
	jwtService JWTService
	config     *config.Config
}

// NewAuthService 認証サービスを作成
func NewAuthService(userRepo repository.UserRepository, jwtService JWTService, cfg *config.Config) AuthService {
	return &authService{
		userRepo:   userRepo,
		jwtService: jwtService,
		config:     cfg,
	}
}

// Register 新規ユーザー登録（ローカル認証）
func (s *authService) Register(req *models.RegisterRequest, clientIP string) (*models.AuthResponse, error) {
	// IP制限チェック
	if err := s.CheckIPLimit(clientIP); err != nil {
		return nil, err
	}

	// メールアドレスの重複チェック
	exists, err := s.userRepo.IsEmailExists(req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check email existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("email already exists")
	}

	// ユーザー名の重複チェック
	exists, err = s.userRepo.IsUsernameExists(req.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to check username existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("username already exists")
	}

	// パスワードハッシュ化
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// ユーザー作成
	user := &models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: stringPtr(string(hashedPassword)),
		IsActive:     true,
		CreatedIP:    clientIP,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// IP登録カウントを更新
	if err := s.updateIPRegistration(clientIP); err != nil {
		// ログに記録するが、エラーで失敗させない
		fmt.Printf("Warning: failed to update IP registration: %v\n", err)
	}

	// トークン生成
	return s.generateAuthResponse(user)
}

// Login ユーザーログイン（ローカル認証）
func (s *authService) Login(req *models.LoginRequest, clientIP string) (*models.AuthResponse, error) {
	// ユーザー取得
	user, err := s.userRepo.GetByEmail(req.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// アカウント有効性チェック
	if !user.IsActive {
		return nil, fmt.Errorf("account is deactivated")
	}

	// パスワード認証（ローカル認証の場合のみ）
	if user.PasswordHash == nil {
		return nil, fmt.Errorf("this account uses external authentication")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// 最終ログイン時刻更新
	if err := s.userRepo.UpdateLastLogin(user.ID); err != nil {
		// ログに記録するが、エラーで失敗させない
		fmt.Printf("Warning: failed to update last login: %v\n", err)
	}

	// トークン生成
	return s.generateAuthResponse(user)
}

// GetGitHubAuthURL GitHub認証URLを取得
func (s *authService) GetGitHubAuthURL(state string) string {
	// GitHub OAuth2 URLを手動で構築
	baseURL := "https://github.com/login/oauth/authorize"
	return fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&scope=%s&state=%s",
		baseURL,
		s.config.Auth.GitHubClientID,
		s.config.Auth.GitHubRedirectURL,
		"user:email",
		state,
	)
}

// HandleGitHubCallback GitHubコールバックを処理
func (s *authService) HandleGitHubCallback(code, state, clientIP string) (*models.AuthResponse, error) {
	// 簡易実装：GitHubからアクセストークンを取得
	accessToken, err := s.exchangeCodeForToken(code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// GitHubユーザー情報を取得
	githubUser, err := s.getGitHubUser(accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub user: %w", err)
	}

	// 既存ユーザーをチェック
	existingUser, err := s.userRepo.GetByGitHubID(githubUser.ID)
	if err == nil {
		// 既存ユーザーの場合
		if !existingUser.IsActive {
			return nil, fmt.Errorf("account is deactivated")
		}

		// 最終ログイン時刻更新
		if err := s.userRepo.UpdateLastLogin(existingUser.ID); err != nil {
			fmt.Printf("Warning: failed to update last login: %v\n", err)
		}

		return s.generateAuthResponse(existingUser)
	}

	// 新規ユーザーの場合、IP制限チェック
	if err := s.CheckIPLimit(clientIP); err != nil {
		return nil, err
	}

	// メールアドレスの重複チェック（GitHubのメールアドレスで）
	if githubUser.Email != "" {
		exists, err := s.userRepo.IsEmailExists(githubUser.Email)
		if err != nil {
			return nil, fmt.Errorf("failed to check email existence: %w", err)
		}
		if exists {
			return nil, fmt.Errorf("email already exists with different authentication method")
		}
	}

	// 新規ユーザー作成
	username := githubUser.Login
	// ユーザー名が重複している場合は番号を付ける
	originalUsername := username
	counter := 1
	for {
		exists, err := s.userRepo.IsUsernameExists(username)
		if err != nil {
			return nil, fmt.Errorf("failed to check username existence: %w", err)
		}
		if !exists {
			break
		}
		username = fmt.Sprintf("%s%d", originalUsername, counter)
		counter++
	}

	user := &models.User{
		Username:       username,
		Email:          githubUser.Email,
		GitHubID:       &githubUser.ID,
		GitHubUsername: &githubUser.Login,
		AvatarURL:      stringPtr(githubUser.AvatarURL),
		IsActive:       true,
		CreatedIP:      clientIP,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// IP登録カウントを更新
	if err := s.updateIPRegistration(clientIP); err != nil {
		fmt.Printf("Warning: failed to update IP registration: %v\n", err)
	}

	return s.generateAuthResponse(user)
}

// ValidateToken トークンを検証
func (s *authService) ValidateToken(tokenString string) (*models.User, error) {
	claims, err := s.jwtService.ValidateToken(tokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	user, err := s.userRepo.GetByID(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	if !user.IsActive {
		return nil, fmt.Errorf("account is deactivated")
	}

	return user, nil
}

// RefreshToken リフレッシュトークンで新しいトークンを生成
func (s *authService) RefreshToken(refreshToken string) (*models.AuthResponse, error) {
	claims, err := s.jwtService.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	user, err := s.userRepo.GetByID(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	if !user.IsActive {
		return nil, fmt.Errorf("account is deactivated")
	}

	return s.generateAuthResponse(user)
}

// CheckIPLimit IP制限をチェック
func (s *authService) CheckIPLimit(clientIP string) error {
	// 現在のユーザー数を取得
	currentCount, err := s.userRepo.GetUserCountByIP(clientIP)
	if err != nil {
		return fmt.Errorf("failed to check IP limit: %w", err)
	}

	if currentCount >= s.config.Auth.MaxAccountsPerIP {
		return fmt.Errorf("maximum number of accounts per IP address exceeded")
	}

	return nil
}

// generateAuthResponse 認証レスポンスを生成
func (s *authService) generateAuthResponse(user *models.User) (*models.AuthResponse, error) {
	accessToken, err := s.jwtService.GenerateAccessToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.jwtService.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &models.AuthResponse{
		User:         user.ToPublic(),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.config.Auth.JWTExpiresIn.Seconds()),
	}, nil
}

// updateIPRegistration IP登録情報を更新
func (s *authService) updateIPRegistration(clientIP string) error {
	ipReg, err := s.userRepo.GetIPRegistration(clientIP)
	if err != nil {
		return err
	}

	if ipReg == nil {
		// 新規作成
		ipReg = &models.IPRegistration{
			IPAddress:  clientIP,
			UserCount:  1,
			LastUsedAt: time.Now(),
		}
		return s.userRepo.CreateIPRegistration(ipReg)
	} else {
		// 更新
		ipReg.UserCount++
		ipReg.LastUsedAt = time.Now()
		return s.userRepo.UpdateIPRegistration(ipReg)
	}
}

// getGitHubUser GitHubユーザー情報を取得
func (s *authService) getGitHubUser(accessToken string) (*models.GitHubUser, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "token "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var githubUser models.GitHubUser
	if err := json.Unmarshal(body, &githubUser); err != nil {
		return nil, err
	}

	// メールアドレスが取得できない場合は、別のAPIエンドポイントで取得
	if githubUser.Email == "" {
		emails, err := s.getGitHubUserEmails(accessToken)
		if err == nil && len(emails) > 0 {
			githubUser.Email = emails[0]
		}
	}

	return &githubUser, nil
}

// getGitHubUserEmails GitHubユーザーのメールアドレスを取得
func (s *authService) getGitHubUserEmails(accessToken string) ([]string, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "token "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var emails []struct {
		Email   string `json:"email"`
		Primary bool   `json:"primary"`
	}

	if err := json.Unmarshal(body, &emails); err != nil {
		return nil, err
	}

	var result []string
	// プライマリメールを優先
	for _, email := range emails {
		if email.Primary {
			result = append([]string{email.Email}, result...)
		} else {
			result = append(result, email.Email)
		}
	}

	return result, nil
}

// generateRandomState ランダムなstate文字列を生成
func GenerateRandomState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// exchangeCodeForToken GitHubのコードをアクセストークンに交換
func (s *authService) exchangeCodeForToken(code string) (string, error) {
	tokenURL := "https://github.com/login/oauth/access_token"

	data := url.Values{}
	data.Set("client_id", s.config.Auth.GitHubClientID)
	data.Set("client_secret", s.config.Auth.GitHubClientSecret)
	data.Set("code", code)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tokenResponse struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
		Error       string `json:"error"`
	}

	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return "", err
	}

	if tokenResponse.Error != "" {
		return "", fmt.Errorf("GitHub token exchange error: %s", tokenResponse.Error)
	}

	return tokenResponse.AccessToken, nil
}

// stringPtr 文字列のポインタを生成
func stringPtr(s string) *string {
	return &s
}
