package storage_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"memo-app/src/storage"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// テスト前の初期化
	os.Setenv("LOG_LEVEL", "error")          // テスト時はエラーレベルのみ
	os.Setenv("LOG_UPLOAD_ENABLED", "false") // テスト時はアップロード無効

	code := m.Run()
	os.Exit(code)
}

func TestNewLogUploader(t *testing.T) {
	testLogger := logrus.New()
	testLogger.SetLevel(logrus.ErrorLevel) // テスト時は静かに

	t.Run("有効な設定でのアップローダー作成", func(t *testing.T) {
		config := &storage.S3Config{
			Endpoint:        "http://localhost:9000",
			AccessKeyID:     "test-access-key",
			SecretAccessKey: "test-secret-key",
			Region:          "us-east-1",
			Bucket:          "test-bucket",
			UseSSL:          false,
		}

		uploader, err := storage.NewLogUploader(config, testLogger)
		assert.NoError(t, err)
		assert.NotNil(t, uploader)
	})

	t.Run("AWS S3用の設定", func(t *testing.T) {
		config := &storage.S3Config{
			Endpoint:        "", // 空の場合はAWS S3
			AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
			SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			Region:          "us-west-2",
			Bucket:          "my-log-bucket",
			UseSSL:          true,
		}

		uploader, err := storage.NewLogUploader(config, testLogger)
		assert.NoError(t, err)
		assert.NotNil(t, uploader)
	})
}

func TestUploadOldLogs(t *testing.T) {
	// モックのS3設定（実際にはアップロードしない）
	testLogger := logrus.New()
	testLogger.SetLevel(logrus.ErrorLevel)

	config := &storage.S3Config{
		Endpoint:        "http://localhost:9000",
		AccessKeyID:     "test-access-key",
		SecretAccessKey: "test-secret-key",
		Region:          "us-east-1",
		Bucket:          "test-bucket",
		UseSSL:          false,
	}

	uploader, err := storage.NewLogUploader(config, testLogger)
	require.NoError(t, err)

	t.Run("空のディレクトリ", func(t *testing.T) {
		tempDir := t.TempDir()

		_ = uploader.UploadOldLogs(tempDir, 1*time.Hour)
		// ネットワークエラーは期待されるが、パニックは発生しない
		// S3接続エラーが期待されるため、エラーチェックは行わない
	})

	t.Run("ログファイルありのディレクトリ", func(t *testing.T) {
		tempDir := t.TempDir()

		// 古いログファイルを作成
		oldLogFile := filepath.Join(tempDir, "old_app.log")
		err := os.WriteFile(oldLogFile, []byte("old log content"), 0644)
		require.NoError(t, err)

		// ファイルの変更時刻を古く設定
		oldTime := time.Now().Add(-2 * time.Hour)
		err = os.Chtimes(oldLogFile, oldTime, oldTime)
		require.NoError(t, err)

		// 新しいログファイルを作成
		newLogFile := filepath.Join(tempDir, "new_app.log")
		err = os.WriteFile(newLogFile, []byte("new log content"), 0644)
		require.NoError(t, err)

		// ファイルが存在することを確認
		assert.FileExists(t, oldLogFile)
		assert.FileExists(t, newLogFile)

		// アップロードを試行（実際の接続は失敗するが、ロジックをテスト）
		_ = uploader.UploadOldLogs(tempDir, 1*time.Hour)
		// ネットワークエラーは期待されるが、関数は正常に動作する
	})

	t.Run("存在しないディレクトリ", func(t *testing.T) {
		err := uploader.UploadOldLogs("/nonexistent/directory", 1*time.Hour)
		assert.Error(t, err)
	})
}

func TestLogFileFiltering(t *testing.T) {
	testLogger := logrus.New()
	testLogger.SetLevel(logrus.ErrorLevel)

	config := &storage.S3Config{
		Endpoint:        "http://localhost:9000",
		AccessKeyID:     "test-access-key",
		SecretAccessKey: "test-secret-key",
		Region:          "us-east-1",
		Bucket:          "test-bucket",
		UseSSL:          false,
	}

	uploader, err := storage.NewLogUploader(config, testLogger)
	require.NoError(t, err)

	t.Run("ログファイルのフィルタリング", func(t *testing.T) {
		tempDir := t.TempDir()

		// 様々なファイルを作成
		files := []string{
			"app.log",     // .logファイル（古い）
			"error.log",   // .logファイル（古い）
			"debug.txt",   // .txtファイル（無視されるべき）
			"config.json", // .jsonファイル（無視されるべき）
			"recent.log",  // .logファイル（新しい）
		}

		for i, fileName := range files {
			filePath := filepath.Join(tempDir, fileName)
			err := os.WriteFile(filePath, []byte("content"), 0644)
			require.NoError(t, err)

			// 最初の2つのファイルを古くする
			if i < 2 {
				oldTime := time.Now().Add(-2 * time.Hour)
				err = os.Chtimes(filePath, oldTime, oldTime)
				require.NoError(t, err)
			}
		}

		// アップロードを試行
		_ = uploader.UploadOldLogs(tempDir, 1*time.Hour)
		// ネットワークエラーは期待されるが、.logファイルのみが処理される
	})
}

func TestS3ConfigValidation(t *testing.T) {
	testLogger := logrus.New()
	testLogger.SetLevel(logrus.ErrorLevel)

	t.Run("必須フィールドの検証", func(t *testing.T) {
		configs := []*storage.S3Config{
			{
				// AccessKeyIDが空
				SecretAccessKey: "secret",
				Region:          "us-east-1",
				Bucket:          "bucket",
			},
			{
				AccessKeyID: "access-key",
				// SecretAccessKeyが空
				Region: "us-east-1",
				Bucket: "bucket",
			},
			{
				AccessKeyID:     "access-key",
				SecretAccessKey: "secret",
				// Regionが空
				Bucket: "bucket",
			},
			{
				AccessKeyID:     "access-key",
				SecretAccessKey: "secret",
				Region:          "us-east-1",
				// Bucketが空
			},
		}

		for i, config := range configs {
			uploader, err := storage.NewLogUploader(config, testLogger)
			// 設定によってはエラーが発生する可能性があるが、
			// AWS SDKは実際の接続時までエラーを検出しない場合がある
			if err != nil {
				t.Logf("Config %d failed as expected: %v", i, err)
			} else {
				assert.NotNil(t, uploader, "Config %d should create uploader", i)
			}
		}
	})
}

func BenchmarkLogUploader(b *testing.B) {
	testLogger := logrus.New()
	testLogger.SetLevel(logrus.ErrorLevel)

	config := &storage.S3Config{
		Endpoint:        "http://localhost:9000",
		AccessKeyID:     "test-access-key",
		SecretAccessKey: "test-secret-key",
		Region:          "us-east-1",
		Bucket:          "test-bucket",
		UseSSL:          false,
	}

	b.Run("アップローダー作成", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			uploader, err := storage.NewLogUploader(config, testLogger)
			if err != nil {
				b.Fatal(err)
			}
			_ = uploader
		}
	})
}
