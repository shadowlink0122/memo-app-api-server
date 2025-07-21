package middleware

import (
	"memo-app/src/logger"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AuthMiddleware ユーザー認証用のmiddleware
func AuthMiddleware() gin.HandlerFunc {
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

		// 簡単なtoken検証（テスト用）
		// 実際のプロダクションではJWT検証やデータベースでの検証を行う
		if !validateToken(token) {
			logger.WithFields(logrus.Fields{
				"client_ip": c.ClientIP(),
				"token":     token[:min(len(token), 10)] + "...", // tokenの最初の10文字のみログ出力
			}).Warn("認証失敗: 無効なtoken")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// 認証成功
		logger.WithField("client_ip", c.ClientIP()).Info("認証成功")
		c.Next()
	}
}

// validateToken tokenの検証を行う（簡単な実装）
func validateToken(token string) bool {
	// テスト用の簡単なtoken検証
	// 実際にはJWTの検証やデータベースでの確認を行う
	validTokens := []string{
		"valid-token-123",
		"test-token",
		"admin-token",
	}

	for _, validToken := range validTokens {
		if token == validToken {
			return true
		}
	}
	return false
}

// min は2つの整数のうち小さい方を返すヘルパー関数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
