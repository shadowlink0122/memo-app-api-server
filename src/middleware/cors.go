package middleware

import (
	"memo-app/src/logger"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// CORSMiddleware CORS設定用のmiddleware
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		logger.WithFields(logrus.Fields{
			"method": c.Request.Method,
			"origin": origin,
			"uri":    c.Request.RequestURI,
		}).Debug("CORS middleware processing")

		// TODO: 将来的にここで適切なCORS設定を実装
		// セキュリティのため、本番環境では適切なオリジンを設定すること
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With")
		c.Header("Access-Control-Expose-Headers", "Content-Type, Authorization, X-Total-Count, X-Page, X-Limit, X-Total-Pages")
		c.Header("Access-Control-Max-Age", "86400") // 24時間

		if c.Request.Method == "OPTIONS" {
			logger.WithFields(logrus.Fields{
				"origin": origin,
				"uri":    c.Request.RequestURI,
			}).Debug("CORS preflight request handled")

			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
