package config_test

import (
	"os"
	"testing"
	"time"

	"memo-app/src/config"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	// テスト前に環境変数をクリア
	defer func() {
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("LOG_DIRECTORY")
		os.Unsetenv("LOG_UPLOAD_ENABLED")
		os.Unsetenv("LOG_UPLOAD_MAX_AGE")
		os.Unsetenv("LOG_UPLOAD_INTERVAL")
		os.Unsetenv("S3_ENDPOINT")
		os.Unsetenv("S3_ACCESS_KEY_ID")
		os.Unsetenv("S3_SECRET_ACCESS_KEY")
		os.Unsetenv("S3_REGION")
		os.Unsetenv("S3_BUCKET")
		os.Unsetenv("S3_USE_SSL")
	}()

	t.Run("デフォルト値でのconfig読み込み", func(t *testing.T) {
		cfg := config.LoadConfig()

		assert.Equal(t, "8080", cfg.Server.Port)
		assert.Equal(t, "info", cfg.Log.Level)
		assert.Equal(t, "logs", cfg.Log.Directory)
		assert.True(t, cfg.Log.UploadEnabled)
		assert.Equal(t, 24*time.Hour, cfg.Log.UploadMaxAge)
		assert.Equal(t, 1*time.Hour, cfg.Log.UploadInterval)

		assert.Equal(t, "http://localhost:9000", cfg.S3.Endpoint)
		assert.Equal(t, "minioadmin", cfg.S3.AccessKeyID)
		assert.Equal(t, "minioadmin", cfg.S3.SecretAccessKey)
		assert.Equal(t, "us-east-1", cfg.S3.Region)
		assert.Equal(t, "memo-app-logs", cfg.S3.Bucket)
		assert.False(t, cfg.S3.UseSSL)
	})

	t.Run("環境変数でのconfig上書き", func(t *testing.T) {
		// テスト用の環境変数を設定
		os.Setenv("SERVER_PORT", "9090")
		os.Setenv("LOG_LEVEL", "debug")
		os.Setenv("LOG_DIRECTORY", "test-logs")
		os.Setenv("LOG_UPLOAD_ENABLED", "false")
		os.Setenv("LOG_UPLOAD_MAX_AGE", "12h")
		os.Setenv("LOG_UPLOAD_INTERVAL", "30m")
		os.Setenv("S3_ENDPOINT", "https://s3.amazonaws.com")
		os.Setenv("S3_ACCESS_KEY_ID", "test-access-key")
		os.Setenv("S3_SECRET_ACCESS_KEY", "test-secret-key")
		os.Setenv("S3_REGION", "ap-northeast-1")
		os.Setenv("S3_BUCKET", "test-bucket")
		os.Setenv("S3_USE_SSL", "true")

		cfg := config.LoadConfig()

		assert.Equal(t, "9090", cfg.Server.Port)
		assert.Equal(t, "debug", cfg.Log.Level)
		assert.Equal(t, "test-logs", cfg.Log.Directory)
		assert.False(t, cfg.Log.UploadEnabled)
		assert.Equal(t, 12*time.Hour, cfg.Log.UploadMaxAge)
		assert.Equal(t, 30*time.Minute, cfg.Log.UploadInterval)

		assert.Equal(t, "https://s3.amazonaws.com", cfg.S3.Endpoint)
		assert.Equal(t, "test-access-key", cfg.S3.AccessKeyID)
		assert.Equal(t, "test-secret-key", cfg.S3.SecretAccessKey)
		assert.Equal(t, "ap-northeast-1", cfg.S3.Region)
		assert.Equal(t, "test-bucket", cfg.S3.Bucket)
		assert.True(t, cfg.S3.UseSSL)
	})

	t.Run("不正な環境変数でのフォールバック", func(t *testing.T) {
		// 不正な値を設定
		os.Setenv("LOG_UPLOAD_ENABLED", "invalid-bool")
		os.Setenv("LOG_UPLOAD_MAX_AGE", "invalid-duration")
		os.Setenv("S3_USE_SSL", "not-a-bool")

		cfg := config.LoadConfig()

		// デフォルト値にフォールバックすることを確認
		assert.True(t, cfg.Log.UploadEnabled)
		assert.Equal(t, 24*time.Hour, cfg.Log.UploadMaxAge)
		assert.False(t, cfg.S3.UseSSL)
	})
}

func TestConfigStructure(t *testing.T) {
	cfg := config.LoadConfig()

	// 設定構造体が適切に初期化されていることを確認
	assert.NotNil(t, cfg)
	assert.NotNil(t, cfg.Server)
	assert.NotNil(t, cfg.Log)
	assert.NotNil(t, cfg.S3)

	// 必須フィールドが空でないことを確認
	assert.NotEmpty(t, cfg.Server.Port)
	assert.NotEmpty(t, cfg.Log.Level)
	assert.NotEmpty(t, cfg.Log.Directory)
	assert.NotEmpty(t, cfg.S3.Region)
	assert.NotEmpty(t, cfg.S3.Bucket)
}

func BenchmarkLoadConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		config.LoadConfig()
	}
}
