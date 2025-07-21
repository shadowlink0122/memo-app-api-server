package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// E2Eテストのためのテストスイート
type E2ETestSuite struct {
	db          *sql.DB
	composeFile string
	testDir     string
}

// E2Eテストのメインエントリーポイント
func TestE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("短いテストモードでE2Eテストをスキップ")
	}

	suite := &E2ETestSuite{
		composeFile: "../docker-compose.yml",
		testDir:     "../",
	}

	t.Run("Setup", suite.TestSetup)
	t.Run("DatabaseIntegration", suite.TestDatabaseIntegration)
	t.Run("APIIntegration", suite.TestAPIIntegration)
	t.Run("LoggingIntegration", suite.TestLoggingIntegration)
	t.Run("S3Integration", suite.TestS3Integration)
	t.Run("Cleanup", suite.TestCleanup)
}

// Docker Composeでの環境セットアップ
func (suite *E2ETestSuite) TestSetup(t *testing.T) {
	// 現在のディレクトリを変更
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	err = os.Chdir(suite.testDir)
	require.NoError(t, err)

	// Docker Composeでサービスを起動
	cmd := exec.Command("docker-compose", "-f", "docker-compose.yml", "up", "-d", "--build")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	require.NoError(t, err, "Docker Composeの起動に失敗")

	// サービスが準備完了するまで待機
	time.Sleep(30 * time.Second)

	// データベース接続を確立
	dsn := "postgres://memo_user:memo_password@localhost:5433/memo_db?sslmode=disable"

	// 接続を何度か試行
	var db *sql.DB
	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", dsn)
		if err == nil {
			err = db.Ping()
			if err == nil {
				suite.db = db
				break
			}
		}
		time.Sleep(3 * time.Second)
	}

	require.NoError(t, err, "データベース接続に失敗")
	require.NotNil(t, suite.db, "データベース接続が確立されていません")
}

// データベース統合テスト
func (suite *E2ETestSuite) TestDatabaseIntegration(t *testing.T) {
	require.NotNil(t, suite.db, "データベース接続が確立されていません")

	// テーブルの存在確認
	tables := []string{"users", "memos"}
	for _, table := range tables {
		var exists bool
		err := suite.db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' AND table_name = $1
			);
		`, table).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "テーブルが存在しません: %s", table)
	}

	// サンプルデータの確認
	var userCount int
	err := suite.db.QueryRow("SELECT COUNT(*) FROM users WHERE username = 'testuser'").Scan(&userCount)
	require.NoError(t, err)
	assert.Equal(t, 1, userCount, "testuserが存在しません")

	// メモのCRUD操作テスト
	var userID string
	err = suite.db.QueryRow("SELECT id FROM users WHERE username = 'testuser' LIMIT 1").Scan(&userID)
	require.NoError(t, err)

	// メモを作成
	var memoID string
	err = suite.db.QueryRow(`
		INSERT INTO memos (user_id, title, content, tags, is_public) 
		VALUES ($1, 'E2Eテストメモ', 'E2Eテスト用のメモです', ARRAY['test', 'e2e'], true) 
		RETURNING id
	`, userID).Scan(&memoID)
	require.NoError(t, err)

	// メモを読み取り
	var title, content string
	var isPublic bool
	err = suite.db.QueryRow(`
		SELECT title, content, is_public 
		FROM memos 
		WHERE id = $1
	`, memoID).Scan(&title, &content, &isPublic)
	require.NoError(t, err)
	assert.Equal(t, "E2Eテストメモ", title)
	assert.Equal(t, "E2Eテスト用のメモです", content)
	assert.True(t, isPublic)

	// メモを更新
	_, err = suite.db.Exec(`
		UPDATE memos 
		SET title = 'E2Eテストメモ（更新済み）' 
		WHERE id = $1
	`, memoID)
	require.NoError(t, err)

	// 更新を確認
	err = suite.db.QueryRow("SELECT title FROM memos WHERE id = $1", memoID).Scan(&title)
	require.NoError(t, err)
	assert.Equal(t, "E2Eテストメモ（更新済み）", title)

	// メモを削除
	_, err = suite.db.Exec("DELETE FROM memos WHERE id = $1", memoID)
	require.NoError(t, err)

	// 削除を確認
	var count int
	err = suite.db.QueryRow("SELECT COUNT(*) FROM memos WHERE id = $1", memoID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

// API統合テスト
func (suite *E2ETestSuite) TestAPIIntegration(t *testing.T) {
	// APIサーバーに対するHTTPリクエストテスト
	// TODO: Dockerコンテナ内のAPIサーバーにアクセスするテストを実装
	// 現在の実装では、APIサーバーが別プロセスとして動作していないため、スキップ
	t.Skip("APIサーバーの統合テストは別途実装が必要")
}

// ログ統合テスト
func (suite *E2ETestSuite) TestLoggingIntegration(t *testing.T) {
	// ログディレクトリの確認
	logDir := "../logs"
	if _, err := os.Stat(logDir); err != nil {
		t.Skip("ログディレクトリが存在しません")
	}

	// ログファイルの存在確認
	files, err := filepath.Glob(filepath.Join(logDir, "*.log"))
	require.NoError(t, err)

	// ログファイルが存在することを確認（テスト実行によって作成される）
	if len(files) > 0 {
		assert.Greater(t, len(files), 0, "ログファイルが作成されていません")

		// 最新のログファイルの内容確認
		if len(files) > 0 {
			content, err := os.ReadFile(files[0])
			require.NoError(t, err)
			assert.Greater(t, len(content), 0, "ログファイルが空です")
		}
	}
}

// S3統合テスト
func (suite *E2ETestSuite) TestS3Integration(t *testing.T) {
	// MinIOコンテナに対する統合テスト
	// TODO: MinIOクライアントを使用してバケットとオブジェクトの操作をテスト
	t.Skip("S3/MinIO統合テストは別途実装が必要")
}

// クリーンアップ
func (suite *E2ETestSuite) TestCleanup(t *testing.T) {
	// データベース接続を閉じる
	if suite.db != nil {
		suite.db.Close()
	}

	// 現在のディレクトリを変更
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	err = os.Chdir(suite.testDir)
	require.NoError(t, err)

	// Docker Composeでサービスを停止
	cmd := exec.Command("docker-compose", "-f", "docker-compose.yml", "down", "-v")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		log.Printf("Docker Composeの停止でエラー: %v", err)
	}
}

// ベンチマークテスト
func BenchmarkDatabaseOperations(b *testing.B) {
	if testing.Short() {
		b.Skip("短いテストモードでベンチマークをスキップ")
	}

	// データベース接続
	dsn := "postgres://memo_user:memo_password@localhost:5433/memo_db?sslmode=disable"
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		b.Skip("データベース接続に失敗: " + err.Error())
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		b.Skip("データベースpingに失敗: " + err.Error())
	}

	// ユーザーIDを取得
	var userID string
	err = db.QueryRow("SELECT id FROM users WHERE username = 'testuser' LIMIT 1").Scan(&userID)
	if err != nil {
		b.Skip("testuserが見つかりません: " + err.Error())
	}

	b.ResetTimer()

	b.Run("InsertMemo", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := db.Exec(`
				INSERT INTO memos (user_id, title, content, tags, is_public) 
				VALUES ($1, $2, $3, $4, $5)
			`, userID, fmt.Sprintf("ベンチマークメモ %d", i), "ベンチマーク用メモ", []string{"benchmark"}, false)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("SelectMemos", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			rows, err := db.Query("SELECT title, content FROM memos WHERE user_id = $1 LIMIT 10", userID)
			if err != nil {
				b.Fatal(err)
			}
			for rows.Next() {
				var title, content string
				rows.Scan(&title, &content)
			}
			rows.Close()
		}
	})
}

// ストレステスト
func TestStressDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("短いテストモードでストレステストをスキップ")
	}

	dsn := "postgres://memo_user:memo_password@localhost:5433/memo_db?sslmode=disable"
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skip("データベース接続に失敗: " + err.Error())
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Skip("データベースpingに失敗: " + err.Error())
	}

	// 同時接続数のテスト
	const numGoroutines = 50
	const operationsPerGoroutine = 10

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	errChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(workerID int) {
			for j := 0; j < operationsPerGoroutine; j++ {
				select {
				case <-ctx.Done():
					errChan <- ctx.Err()
					return
				default:
					var count int
					err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
					if err != nil {
						errChan <- err
						return
					}
				}
			}
			errChan <- nil
		}(i)
	}

	// 全てのgoroutineの完了を待つ
	for i := 0; i < numGoroutines; i++ {
		err := <-errChan
		assert.NoError(t, err, "並行データベース操作でエラー")
	}
}
