package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"

	"memo-app/src/logger"
	"memo-app/src/models"
	"memo-app/src/service"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AuthHandler 認証ハンドラー
type AuthHandler struct {
	authService service.AuthService
}

// NewAuthHandler 認証ハンドラーのコンストラクタ
func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// RegisterRequest 新規登録リクエスト
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginRequest ログインリクエスト
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Register 新規登録
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// クライアントIPを取得
	clientIP := getClientIP(c)

	// リクエストをモデル形式に変換
	registerReq := &models.RegisterRequest{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	}

	// 新規登録処理
	authResponse, err := h.authService.Register(registerReq, clientIP)
	if err != nil {
		if strings.Contains(err.Error(), "username already exists") {
			c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
			return
		}
		if strings.Contains(err.Error(), "email already exists") {
			c.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
			return
		}
		if strings.Contains(err.Error(), "IP limit exceeded") {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many registrations from this IP address"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Registration failed"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Registration successful",
		"data":    authResponse,
	})
}

// Login ログイン
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// リクエストをモデル形式に変換
	loginReq := &models.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	}

	// ログイン処理
	authResponse, err := h.authService.Login(loginReq, getClientIP(c))
	if err != nil {
		if strings.Contains(err.Error(), "invalid credentials") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
			return
		}
		if strings.Contains(err.Error(), "account is deactivated") {
			c.JSON(http.StatusForbidden, gin.H{"error": "Account is deactivated"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Login failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"data":    authResponse,
	})
}

// GetGitHubAuthURL GitHub認証URLを取得
func (h *AuthHandler) GetGitHubAuthURL(c *gin.Context) {
	// CSRF防止のためのstateを生成
	state := generateRandomString(32)

	// セッションにstateを保存（本実装では簡略化）
	c.SetCookie("github_oauth_state", state, 600, "/", "", false, true)

	authURL := h.authService.GetGitHubAuthURL(state)

	c.JSON(http.StatusOK, gin.H{
		"auth_url": authURL,
		"state":    state,
	})
}

// GitHubCallback GitHub OAuth コールバック
func (h *AuthHandler) GitHubCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Authorization code is required"})
		return
	}

	// stateの検証（本実装では簡略化）
	storedState, err := c.Cookie("github_oauth_state")
	if err != nil || storedState != state {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid state parameter"})
		return
	}

	// stateクッキーを削除
	c.SetCookie("github_oauth_state", "", -1, "/", "", false, true)

	// クライアントIPを取得
	clientIP := getClientIP(c)

	// GitHub認証処理
	authResponse, err := h.authService.HandleGitHubCallback(code, state, clientIP)
	if err != nil {
		if strings.Contains(err.Error(), "IP limit exceeded") {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many registrations from this IP address"})
			return
		}
		if strings.Contains(err.Error(), "email already exists") {
			c.JSON(http.StatusConflict, gin.H{"error": "Email already exists with different authentication method"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "GitHub authentication failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "GitHub authentication successful",
		"data":    authResponse,
	})
}

// RefreshToken トークンリフレッシュ
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	type RefreshRequest struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	authResponse, err := h.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		if strings.Contains(err.Error(), "invalid refresh token") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token refresh failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Token refreshed successfully",
		"data":    authResponse,
	})
}

// GetProfile 現在のユーザープロフィールを取得
func (h *AuthHandler) GetProfile(c *gin.Context) {
	// ミドルウェアから認証されたユーザーを取得
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	user, ok := userInterface.(*models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": user.ToPublic(),
	})
}

// Logout ユーザーをログアウト
func (h *AuthHandler) Logout(c *gin.Context) {
	// Authorization ヘッダーからトークンを取得
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Authorization header is required"})
		return
	}

	// Bearer トークンの形式をチェック
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid authorization header format"})
		return
	}

	token := parts[1]

	// トークンをブラックリストに追加（トークンを無効化）
	err := h.authService.InvalidateToken(token)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
			"token": token[:10] + "...", // セキュリティのため一部のみログ出力
		}).Error("トークンの無効化に失敗")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to logout"})
		return
	}

	logger.WithFields(logrus.Fields{
		"token": token[:10] + "...", // セキュリティのため一部のみログ出力
	}).Info("トークンを正常に無効化しました")

	c.JSON(http.StatusOK, gin.H{
		"message": "Successfully logged out",
	})
}

// getClientIP クライアントのIPアドレスを取得
func getClientIP(c *gin.Context) string {
	// X-Forwarded-For ヘッダーをチェック
	xForwardedFor := c.GetHeader("X-Forwarded-For")
	if xForwardedFor != "" {
		// 複数のIPがある場合は最初のものを使用
		ips := strings.Split(xForwardedFor, ",")
		return strings.TrimSpace(ips[0])
	}

	// X-Real-IP ヘッダーをチェック
	xRealIP := c.GetHeader("X-Real-IP")
	if xRealIP != "" {
		return xRealIP
	}

	// それ以外の場合はRemoteAddrを使用
	return c.ClientIP()
}

// generateRandomString ランダムな文字列を生成
func generateRandomString(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:length]
}
