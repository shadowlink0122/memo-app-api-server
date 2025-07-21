package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	Log          *logrus.Logger
	currentFile  *os.File
	logDirectory = "logs"
)

// InitLogger ロガーを初期化し、ファイル出力を設定
func InitLogger() error {
	Log = logrus.New()

	// ログレベルを設定
	Log.SetLevel(logrus.InfoLevel)

	// JSON形式でログを出力
	Log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})

	// ログディレクトリを作成
	if err := os.MkdirAll(logDirectory, 0755); err != nil {
		return fmt.Errorf("ログディレクトリの作成に失敗: %v", err)
	}

	// 新しいログファイルを作成
	if err := rotateLogFile(); err != nil {
		return fmt.Errorf("ログファイルの作成に失敗: %v", err)
	}

	// 標準出力とファイルの両方に出力
	multiWriter := io.MultiWriter(os.Stdout, currentFile)
	Log.SetOutput(multiWriter)

	Log.Info("ロガーが初期化されました")
	return nil
}

// rotateLogFile 新しいログファイルを作成
func rotateLogFile() error {
	// 既存のファイルを閉じる
	if currentFile != nil {
		currentFile.Close()
	}

	// 新しいファイル名を生成（タイムスタンプ付き）
	filename := fmt.Sprintf("app_%s.log", time.Now().Format("2006-01-02_15-04-05"))
	filepath := filepath.Join(logDirectory, filename)

	// 新しいファイルを作成
	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	currentFile = file
	Log.WithField("file", filepath).Info("新しいログファイルを作成しました")
	return nil
}

// GetCurrentLogFile 現在のログファイルパスを取得
func GetCurrentLogFile() string {
	if currentFile != nil {
		return currentFile.Name()
	}
	return ""
}

// CloseLogger ロガーを終了
func CloseLogger() {
	if currentFile != nil {
		Log.Info("ログファイルを閉じます")
		currentFile.Close()
	}
}

// WithFields フィールド付きログエントリを作成
func WithFields(fields logrus.Fields) *logrus.Entry {
	return Log.WithFields(fields)
}

// WithField フィールド付きログエントリを作成（単一フィールド）
func WithField(key string, value interface{}) *logrus.Entry {
	return Log.WithField(key, value)
}
