package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/sirupsen/logrus"
)

type S3Config struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	Bucket          string
	UseSSL          bool
}

type LogUploader struct {
	s3Client *s3.S3
	config   *S3Config
	logger   *logrus.Logger
}

// NewLogUploader S3アップローダーを作成
func NewLogUploader(config *S3Config, logger *logrus.Logger) (*LogUploader, error) {
	// AWS設定
	awsConfig := &aws.Config{
		Region:           aws.String(config.Region),
		Credentials:      credentials.NewStaticCredentials(config.AccessKeyID, config.SecretAccessKey, ""),
		DisableSSL:       aws.Bool(!config.UseSSL),
		S3ForcePathStyle: aws.Bool(true), // MinIOなどのS3互換ストレージ用
	}

	// エンドポイントが指定されている場合（MinIOなど）
	if config.Endpoint != "" {
		awsConfig.Endpoint = aws.String(config.Endpoint)
	}

	// セッションを作成
	sess, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, fmt.Errorf("AWSセッションの作成に失敗: %v", err)
	}

	return &LogUploader{
		s3Client: s3.New(sess),
		config:   config,
		logger:   logger,
	}, nil
}

// UploadLogFile ログファイルをS3にアップロード
func (u *LogUploader) UploadLogFile(filePath string) error {
	// ファイルを開く
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("ファイルの読み込みに失敗: %v", err)
	}
	defer file.Close()

	// S3オブジェクトキーを生成（ファイル名にタイムスタンプを追加）
	fileName := filepath.Base(filePath)
	objectKey := fmt.Sprintf("logs/%s", fileName)

	// S3にアップロード
	_, err = u.s3Client.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(u.config.Bucket),
		Key:         aws.String(objectKey),
		Body:        file,
		ContentType: aws.String("text/plain"),
		Metadata: map[string]*string{
			"upload-time": aws.String(time.Now().Format(time.RFC3339)),
			"source":      aws.String("memo-app-api-server"),
		},
	})

	if err != nil {
		return fmt.Errorf("S3アップロードに失敗: %v", err)
	}

	u.logger.WithFields(logrus.Fields{
		"file":   fileName,
		"bucket": u.config.Bucket,
		"key":    objectKey,
	}).Info("ログファイルをS3にアップロードしました")

	return nil
}

// UploadOldLogs 古いログファイルをアップロードして削除
func (u *LogUploader) UploadOldLogs(logDir string, maxAge time.Duration) error {
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return fmt.Errorf("ログディレクトリの読み取りに失敗: %v", err)
	}

	cutoffTime := time.Now().Add(-maxAge)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}

		filePath := filepath.Join(logDir, entry.Name())
		fileInfo, err := entry.Info()
		if err != nil {
			u.logger.WithError(err).WithField("file", entry.Name()).Error("ファイル情報の取得に失敗")
			continue
		}

		// ファイルが古い場合はアップロードして削除
		if fileInfo.ModTime().Before(cutoffTime) {
			u.logger.WithFields(logrus.Fields{
				"file":    entry.Name(),
				"modTime": fileInfo.ModTime(),
				"cutoff":  cutoffTime,
			}).Info("古いログファイルをアップロード中")

			// S3にアップロード
			if err := u.UploadLogFile(filePath); err != nil {
				u.logger.WithError(err).WithField("file", entry.Name()).Error("ログファイルのアップロードに失敗")
				continue
			}

			// ローカルファイルを削除
			if err := os.Remove(filePath); err != nil {
				u.logger.WithError(err).WithField("file", entry.Name()).Error("ローカルファイルの削除に失敗")
			} else {
				u.logger.WithField("file", entry.Name()).Info("ローカルファイルを削除しました")
			}
		}
	}

	return nil
}

// StartPeriodicUpload 定期的なアップロードを開始
func (u *LogUploader) StartPeriodicUpload(logDir string, interval time.Duration, maxAge time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			u.logger.Info("定期的なログアップロードを開始")
			if err := u.UploadOldLogs(logDir, maxAge); err != nil {
				u.logger.WithError(err).Error("定期的なログアップロードに失敗")
			}
		}
	}()

	u.logger.WithFields(logrus.Fields{
		"interval": interval,
		"maxAge":   maxAge,
	}).Info("定期的なログアップロードを開始しました")
}
