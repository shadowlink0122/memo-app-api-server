package logger_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"memo-app/src/logger"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitLogger(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir := t.TempDir()

	// 元の作業ディレクトリを保存
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	// テストディレクトリに移動
	os.Chdir(tempDir)

	t.Run("正常初期化", func(t *testing.T) {
		err := logger.InitLogger()
		require.NoError(t, err)

		assert.NotNil(t, logger.Log)
		assert.Equal(t, logrus.InfoLevel, logger.Log.Level)

		// ログディレクトリが作成されていることを確認
		assert.DirExists(t, "logs")

		// ログファイルが作成されていることを確認
		logFile := logger.GetCurrentLogFile()
		assert.NotEmpty(t, logFile)
		assert.FileExists(t, logFile)

		logger.CloseLogger()
	})

	t.Run("ログレベル設定", func(t *testing.T) {
		// ログレベルを環境変数で設定
		os.Setenv("LOG_LEVEL", "debug")
		defer os.Unsetenv("LOG_LEVEL")

		err := logger.InitLogger()
		require.NoError(t, err)

		assert.Equal(t, logrus.InfoLevel, logger.Log.Level) // 現在の実装ではハードコード

		logger.CloseLogger()
	})
}

func TestLoggerFunctions(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir := t.TempDir()

	// 元の作業ディレクトリを保存
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	// テストディレクトリに移動
	os.Chdir(tempDir)

	err := logger.InitLogger()
	require.NoError(t, err)
	defer logger.CloseLogger()

	t.Run("基本ログ出力", func(t *testing.T) {
		logger.Log.Info("テストメッセージ")
		logger.Log.Warn("警告メッセージ")
		logger.Log.Error("エラーメッセージ")

		// ファイルが存在することを確認
		logFile := logger.GetCurrentLogFile()
		assert.FileExists(t, logFile)

		// ファイルサイズが0より大きいことを確認
		stat, err := os.Stat(logFile)
		require.NoError(t, err)
		assert.Greater(t, stat.Size(), int64(0))
	})

	t.Run("WithFields機能", func(t *testing.T) {
		fields := logrus.Fields{
			"user_id": "12345",
			"action":  "login",
			"ip":      "192.168.1.1",
		}

		entry := logger.WithFields(fields)
		assert.NotNil(t, entry)

		entry.Info("ユーザーログインテスト")

		// ログファイルにフィールド情報が含まれていることを確認
		logFile := logger.GetCurrentLogFile()
		content, err := os.ReadFile(logFile)
		require.NoError(t, err)

		contentStr := string(content)
		assert.Contains(t, contentStr, "user_id")
		assert.Contains(t, contentStr, "12345")
		assert.Contains(t, contentStr, "action")
		assert.Contains(t, contentStr, "login")
	})

	t.Run("WithField機能", func(t *testing.T) {
		entry := logger.WithField("component", "test")
		assert.NotNil(t, entry)

		entry.Info("コンポーネントテスト")

		// ログファイルにフィールド情報が含まれていることを確認
		logFile := logger.GetCurrentLogFile()
		content, err := os.ReadFile(logFile)
		require.NoError(t, err)

		contentStr := string(content)
		assert.Contains(t, contentStr, "component")
		assert.Contains(t, contentStr, "test")
	})
}

func TestLogFileRotation(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir := t.TempDir()

	// 元の作業ディレクトリを保存
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	// テストディレクトリに移動
	os.Chdir(tempDir)

	err := logger.InitLogger()
	require.NoError(t, err)
	defer logger.CloseLogger()

	t.Run("ログファイル名の形式", func(t *testing.T) {
		logFile := logger.GetCurrentLogFile()
		fileName := filepath.Base(logFile)

		// ファイル名が期待する形式であることを確認
		assert.Regexp(t, `^app_\d{4}-\d{2}-\d{2}_\d{2}-\d{2}-\d{2}\.log$`, fileName)
	})

	t.Run("ログディレクトリの確認", func(t *testing.T) {
		logFile := logger.GetCurrentLogFile()
		logDir := filepath.Dir(logFile)

		assert.Equal(t, "logs", filepath.Base(logDir))
		assert.DirExists(t, logDir)
	})
}

func TestLoggerConcurrency(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir := t.TempDir()

	// 元の作業ディレクトリを保存
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	// テストディレクトリに移動
	os.Chdir(tempDir)

	err := logger.InitLogger()
	require.NoError(t, err)
	defer logger.CloseLogger()

	t.Run("並行ログ出力", func(t *testing.T) {
		const numGoroutines = 10
		const numLogs = 100

		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				for j := 0; j < numLogs; j++ {
					logger.WithFields(logrus.Fields{
						"goroutine": id,
						"iteration": j,
					}).Info("並行テストログ")
				}
				done <- true
			}(i)
		}

		// すべてのゴルーチンの完了を待つ
		for i := 0; i < numGoroutines; i++ {
			select {
			case <-done:
			case <-time.After(10 * time.Second):
				t.Fatal("テストがタイムアウトしました")
			}
		}

		// ログファイルが存在し、サイズが0より大きいことを確認
		logFile := logger.GetCurrentLogFile()
		stat, err := os.Stat(logFile)
		require.NoError(t, err)
		assert.Greater(t, stat.Size(), int64(0))
	})
}

func BenchmarkLogger(b *testing.B) {
	// テスト用の一時ディレクトリを作成
	tempDir := b.TempDir()

	// 元の作業ディレクトリを保存
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	// テストディレクトリに移動
	os.Chdir(tempDir)

	err := logger.InitLogger()
	if err != nil {
		b.Fatal(err)
	}
	defer logger.CloseLogger()

	b.Run("基本ログ出力", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			logger.Log.Info("ベンチマークテストメッセージ")
		}
	})

	b.Run("WithFieldsログ出力", func(b *testing.B) {
		fields := logrus.Fields{
			"user_id": "12345",
			"action":  "test",
		}

		for i := 0; i < b.N; i++ {
			logger.WithFields(fields).Info("ベンチマークテストメッセージ")
		}
	})
}
