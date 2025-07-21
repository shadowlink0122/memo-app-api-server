package middleware

import (
	"time"

	"memo-app/src/logger"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// LoggerMiddleware 構造化ログを使用したロギングmiddleware
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// リクエスト開始時刻を記録
		start := time.Now()

		// リクエスト情報をログに記録
		logger.WithFields(logrus.Fields{
			"method":     c.Request.Method,
			"uri":        c.Request.RequestURI,
			"client_ip":  c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
			"referer":    c.Request.Referer(),
		}).Info("リクエスト開始")

		// 次のmiddlewareまたはハンドラーを実行
		c.Next()

		// レスポンス処理後のログ出力
		latency := time.Since(start)
		statusCode := c.Writer.Status()

		logEntry := logger.WithFields(logrus.Fields{
			"method":        c.Request.Method,
			"uri":           c.Request.RequestURI,
			"client_ip":     c.ClientIP(),
			"status_code":   statusCode,
			"latency_ms":    latency.Milliseconds(),
			"latency":       latency.String(),
			"response_size": c.Writer.Size(),
		})

		// ステータスコードに応じてログレベルを変更
		switch {
		case statusCode >= 500:
			logEntry.Error("リクエスト完了 - サーバーエラー")
		case statusCode >= 400:
			logEntry.Warn("リクエスト完了 - クライアントエラー")
		case statusCode >= 300:
			logEntry.Info("リクエスト完了 - リダイレクト")
		default:
			logEntry.Info("リクエスト完了 - 成功")
		}

		// エラーがある場合は追加でログ出力
		if len(c.Errors) > 0 {
			logger.WithFields(logrus.Fields{
				"method": c.Request.Method,
				"uri":    c.Request.RequestURI,
				"errors": c.Errors.String(),
			}).Error("リクエスト処理中にエラーが発生")
		}
	}
}
