package middleware

import (
	"memo-app/src/logger"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// RateLimitMiddleware レート制限用のmiddleware
func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: 将来的にここでレート制限機能を実装
		// 例：Redis やメモリベースのレート制限

		clientIP := c.ClientIP()

		logger.WithFields(logrus.Fields{
			"client_ip": clientIP,
			"method":    c.Request.Method,
			"uri":       c.Request.RequestURI,
		}).Debug("レート制限チェック中")

		// 実際のレート制限ロジックをここに実装予定
		// 例：
		// if isRateLimited(clientIP) {
		//     logger.WithField("client_ip", clientIP).Warn("レート制限に達しました")
		//     c.JSON(http.StatusTooManyRequests, gin.H{
		//         "error": "Too Many Requests",
		//         "retry_after": 60,
		//     })
		//     c.Abort()
		//     return
		// }

		c.Next()
	}
}
