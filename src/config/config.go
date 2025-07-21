package config

import (
	"os"
	"strconv"
	"time"
)

// Config アプリケーション設定
type Config struct {
	Server ServerConfig
	Log    LogConfig
	S3     S3Config
}

// ServerConfig サーバー設定
type ServerConfig struct {
	Port string
}

// LogConfig ログ設定
type LogConfig struct {
	Level          string
	Directory      string
	UploadEnabled  bool
	UploadMaxAge   time.Duration
	UploadInterval time.Duration
}

// S3Config S3設定
type S3Config struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	Bucket          string
	UseSSL          bool
}

// LoadConfig 環境変数から設定を読み込み
func LoadConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
		},
		Log: LogConfig{
			Level:          getEnv("LOG_LEVEL", "info"),
			Directory:      getEnv("LOG_DIRECTORY", "logs"),
			UploadEnabled:  getBoolEnv("LOG_UPLOAD_ENABLED", true),
			UploadMaxAge:   getDurationEnv("LOG_UPLOAD_MAX_AGE", 24*time.Hour),
			UploadInterval: getDurationEnv("LOG_UPLOAD_INTERVAL", 1*time.Hour),
		},
		S3: S3Config{
			Endpoint:        getEnv("S3_ENDPOINT", "http://localhost:9000"), // MinIO用のデフォルト
			AccessKeyID:     getEnv("S3_ACCESS_KEY_ID", "minioadmin"),
			SecretAccessKey: getEnv("S3_SECRET_ACCESS_KEY", "minioadmin"),
			Region:          getEnv("S3_REGION", "us-east-1"),
			Bucket:          getEnv("S3_BUCKET", "memo-app-logs"),
			UseSSL:          getBoolEnv("S3_USE_SSL", false),
		},
	}
}

// getEnv 環境変数を取得（デフォルト値付き）
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getBoolEnv 環境変数をboolで取得
func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// getDurationEnv 環境変数をtime.Durationで取得
func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}
