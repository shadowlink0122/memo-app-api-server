package middleware

import (
	"memo-app/src/logger"
	"memo-app/src/repository"
	"memo-app/src/service"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AuthMiddleware ユーザー認証用のmiddleware
func AuthMiddleware(jwtService service.JWTService, userRepo repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger.WithFields(logrus.Fields{
			"method":    c.Request.Method,
			"uri":       c.Request.RequestURI,
			"client_ip": c.ClientIP(),
		}).Info("認証ミドルウェア: リクエストを処理中")

		// Authorizationヘッダーを取得
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logger.WithField("client_ip", c.ClientIP()).Warn("認証失敗: Authorizationヘッダーがありません")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Bearer tokenの形式をチェック
		if !strings.HasPrefix(authHeader, "Bearer ") {
			logger.WithField("client_ip", c.ClientIP()).Warn("認証失敗: Bearer tokenの形式が正しくありません")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
			c.Abort()
			return
		}

		// tokenを抽出
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			logger.WithField("client_ip", c.ClientIP()).Warn("認証失敗: tokenが空です")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token is empty"})
			c.Abort()
			return
		}

		// JWT token検証
		userID, err := jwtService.ValidateAccessToken(token)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"client_ip": c.ClientIP(),
				"error":     err.Error(),
			}).Warn("認証失敗: 無効なJWTトークン")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// ユーザー情報を取得
		user, err := userRepo.GetByID(userID)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"client_ip": c.ClientIP(),
				"user_id":   userID,
				"error":     err.Error(),
			}).Warn("認証失敗: ユーザーが見つかりません")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			c.Abort()
			return
		}

		// ユーザーがアクティブかチェック
		if !user.IsActive {
			logger.WithFields(logrus.Fields{
				"client_ip": c.ClientIP(),
				"user_id":   userID,
			}).Warn("認証失敗: ユーザーアカウントが無効です")
			c.JSON(http.StatusForbidden, gin.H{"error": "Account is deactivated"})
			c.Abort()
			return
		}

		// リクエストコンテキストにユーザー情報を設定
		c.Set("user", user)
		c.Set("user_id", userID)

		// 認証成功
		logger.WithFields(logrus.Fields{
			"client_ip": c.ClientIP(),
			"user_id":   userID,
			"username":  user.Username,
		}).Info("認証成功")
		c.Next()
	}
}
