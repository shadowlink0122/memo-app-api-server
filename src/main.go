package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"memo-app/src/config"
	"memo-app/src/logger"
	"memo-app/src/middleware"
	"memo-app/src/storage"

	"github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
)

func main() {
	// 設定を読み込み
	cfg := config.LoadConfig()

	// ロガーを初期化
	if err := logger.InitLogger(); err != nil {
		panic(fmt.Sprintf("ロガーの初期化に失敗: %v", err))
	}
	defer logger.CloseLogger()

	logger.Log.Info("アプリケーションを開始しています")

	// S3アップローダーを初期化（設定が有効な場合）
	var uploader *storage.LogUploader
	if cfg.Log.UploadEnabled {
		s3Config := &storage.S3Config{
			Endpoint:        cfg.S3.Endpoint,
			AccessKeyID:     cfg.S3.AccessKeyID,
			SecretAccessKey: cfg.S3.SecretAccessKey,
			Region:          cfg.S3.Region,
			Bucket:          cfg.S3.Bucket,
			UseSSL:          cfg.S3.UseSSL,
		}

		var err error
		uploader, err = storage.NewLogUploader(s3Config, logger.Log)
		if err != nil {
			logger.Log.WithError(err).Error("S3アップローダーの初期化に失敗")
		} else {
			// 定期的なログアップロードを開始
			uploader.StartPeriodicUpload(cfg.Log.Directory, cfg.Log.UploadInterval, cfg.Log.UploadMaxAge)
		}
	}

	// Ginルーターを初期化
	r := gin.Default()

	// NoRouteハンドラー（404）
	r.NoRoute(func(c *gin.Context) {
		logger.WithFields(logrus.Fields{
			"method":    c.Request.Method,
			"uri":       c.Request.RequestURI,
			"client_ip": c.ClientIP(),
		}).Warn("404: ルートが見つかりません")
		c.JSON(http.StatusNotFound, gin.H{"error": "Route not found"})
	})

	// NoMethodハンドラー（405）
	r.NoMethod(func(c *gin.Context) {
		logger.WithFields(logrus.Fields{
			"method":    c.Request.Method,
			"uri":       c.Request.RequestURI,
			"client_ip": c.ClientIP(),
		}).Warn("405: サポートされていないメソッド")
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "Method not allowed"})
	})

	// グローバルmiddlewareを適用
	r.Use(middleware.LoggerMiddleware())
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.RateLimitMiddleware())

	// 認証が不要なパブリックルート
	public := r.Group("/")
	{
		// Hello WorldのGETエンドポイント
		public.GET("/", func(c *gin.Context) {
			logger.WithField("endpoint", "/").Info("Hello Worldエンドポイントにアクセス")
			c.JSON(http.StatusOK, gin.H{
				"message": "Hello World",
				"version": "2.0",
				"service": "memo-app-api-server",
			})
		})

		// サポートされていないHTTPメソッドのハンドラー（405エラー）
		public.POST("/", func(c *gin.Context) {
			logger.WithFields(logrus.Fields{
				"method": c.Request.Method,
				"uri":    c.Request.RequestURI,
			}).Warn("405: サポートされていないメソッド")
			c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "Method not allowed"})
		})
		public.PUT("/", func(c *gin.Context) {
			logger.WithFields(logrus.Fields{
				"method": c.Request.Method,
				"uri":    c.Request.RequestURI,
			}).Warn("405: サポートされていないメソッド")
			c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "Method not allowed"})
		})
		public.DELETE("/", func(c *gin.Context) {
			logger.WithFields(logrus.Fields{
				"method": c.Request.Method,
				"uri":    c.Request.RequestURI,
			}).Warn("405: サポートされていないメソッド")
			c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "Method not allowed"})
		})
		public.PATCH("/", func(c *gin.Context) {
			logger.WithFields(logrus.Fields{
				"method": c.Request.Method,
				"uri":    c.Request.RequestURI,
			}).Warn("405: サポートされていないメソッド")
			c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "Method not allowed"})
		})

		// ヘルスチェック用のエンドポイント
		public.GET("/health", func(c *gin.Context) {
			logger.WithField("endpoint", "/health").Debug("ヘルスチェックエンドポイントにアクセス")
			c.JSON(http.StatusOK, gin.H{
				"status":    "OK",
				"timestamp": time.Now().Format(time.RFC3339),
				"uptime":    "running",
			})
		})

		// 別のHello Worldエンドポイント（テキスト形式）
		public.GET("/hello", func(c *gin.Context) {
			logger.WithField("endpoint", "/hello").Info("Hello（テキスト）エンドポイントにアクセス")
			c.String(http.StatusOK, "Hello World!")
		})
	}

	// 認証が必要なプライベートルート
	private := r.Group("/api")
	private.Use(middleware.AuthMiddleware())
	{
		// TODO: 将来的にここで認証が必要なAPIエンドポイントを実装
		private.GET("/protected", func(c *gin.Context) {
			logger.WithField("endpoint", "/api/protected").Info("保護されたエンドポイントにアクセス")
			c.JSON(http.StatusOK, gin.H{
				"message":   "これは認証が必要なエンドポイントです",
				"user":      "認証されたユーザー", // TODO: 実際のユーザー情報を返す
				"timestamp": time.Now().Format(time.RFC3339),
			})
		})
	}

	// グレースフルシャットダウンの設定
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		logger.Log.Info("シャットダウンシグナルを受信しました")

		// 最後のログアップロードを実行
		if uploader != nil {
			logger.Log.Info("最後のログアップロードを実行中...")
			if err := uploader.UploadOldLogs(cfg.Log.Directory, 0); err != nil {
				logger.Log.WithError(err).Error("最後のログアップロードに失敗")
			}
		}

		logger.CloseLogger()
		os.Exit(0)
	}()

	// サーバーを起動
	serverAddr := ":" + cfg.Server.Port
	logger.Log.WithField("port", cfg.Server.Port).Info("サーバーを開始します")

	if err := r.Run(serverAddr); err != nil {
		logger.Log.WithError(err).Fatal("サーバーの起動に失敗")
	}
}
