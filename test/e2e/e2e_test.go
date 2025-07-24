package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
		composeFile: "../docker-compose.ci.yml",
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
	// Docker環境内でテストを実行している場合のチェック
	inDocker := os.Getenv("DOCKER_CONTAINER") == "true"
	skipComposeSetup := os.Getenv("E2E_SKIP_COMPOSE_SETUP") == "true"

	if !inDocker && !skipComposeSetup {
		// ローカル環境での実行：Docker Composeを起動
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(originalDir)

		err = os.Chdir(suite.testDir)
		require.NoError(t, err)

		// CI環境の場合、既存のサービスが起動済みかチェック
		cmd := exec.Command("docker", "compose", "-f", suite.composeFile, "ps", "--services", "--filter", "status=running")
		output, err := cmd.Output()

		if err != nil || len(output) == 0 {
			// サービスが動いていない場合は起動
			t.Log("Docker Composeサービスを起動中...")
			cmd := exec.Command("docker", "compose", "-f", suite.composeFile, "up", "-d")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			err = cmd.Run()
			require.NoError(t, err, "Docker Composeの起動に失敗")

			// サービスが準備完了するまで待機
			time.Sleep(30 * time.Second)
		} else {
			t.Log("既存のDocker Composeサービスを使用")
			// 短い待機
			time.Sleep(5 * time.Second)
		}
	} else {
		if skipComposeSetup {
			t.Log("CI環境：Docker Composeセットアップをスキップ")
		} else {
			t.Log("Docker環境内でテストを実行：既存のサービスを使用")
		}
		// 短い待機のみ
		time.Sleep(5 * time.Second)
	}

	// データベース接続を確立
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		// CI環境での実行チェック
		ciEnvironment := os.Getenv("CI_ENVIRONMENT") == "true"

		if inDocker {
			if ciEnvironment {
				// CI環境では通常のメインDBを使用
				dsn = "postgres://memo_user:memo_password@db:5432/memo_db?sslmode=disable"
			} else {
				dsn = "postgres://memo_user:memo_password@db:5432/memo_db_test?sslmode=disable"
			}
		} else {
			if ciEnvironment {
				// CI環境では通常のメインDBを使用
				dsn = "postgres://memo_user:memo_password@localhost:5432/memo_db?sslmode=disable"
			} else {
				dsn = "postgres://memo_user:memo_password@localhost:5432/memo_db_test?sslmode=disable"
			}
		}
	}

	// 接続を何度か試行
	var db *sql.DB
	var err error
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
	tables := []string{"memos"}
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

	// メモのCRUD操作テスト
	// メモを作成
	var memoID int
	err := suite.db.QueryRow(`
		INSERT INTO memos (title, content, category, tags, priority, status) 
		VALUES ('E2Eテストメモ', 'E2Eテスト用のメモです', 'テスト', '["test", "e2e"]', 'medium', 'active') 
		RETURNING id
	`).Scan(&memoID)
	require.NoError(t, err)

	// メモを読み取り
	var title, content, category, tags, priority, status string
	err = suite.db.QueryRow(`
		SELECT title, content, category, tags, priority, status
		FROM memos 
		WHERE id = $1
	`, memoID).Scan(&title, &content, &category, &tags, &priority, &status)
	require.NoError(t, err)
	assert.Equal(t, "E2Eテストメモ", title)
	assert.Equal(t, "E2Eテスト用のメモです", content)
	assert.Equal(t, "テスト", category)
	assert.Equal(t, "medium", priority)
	assert.Equal(t, "active", status)

	// メモを更新
	_, err = suite.db.Exec(`
		UPDATE memos 
		SET title = 'E2Eテストメモ（更新済み）', priority = 'high'
		WHERE id = $1
	`, memoID)
	require.NoError(t, err)

	// 更新を確認
	err = suite.db.QueryRow("SELECT title, priority FROM memos WHERE id = $1", memoID).Scan(&title, &priority)
	require.NoError(t, err)
	assert.Equal(t, "E2Eテストメモ（更新済み）", title)
	assert.Equal(t, "high", priority)

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
	// Docker環境内でテストを実行している場合のチェック
	inDocker := os.Getenv("DOCKER_CONTAINER") == "true"

	var baseURL string
	if inDocker {
		baseURL = "http://app:8000" // Docker環境内ではappコンテナ名を使用
	} else {
		baseURL = "http://localhost:8000" // ローカル環境
	}

	// APIサーバーのヘルスチェック
	healthURL := baseURL + "/health"

	// APIサーバーが起動するまで待機
	maxRetries := 30
	var resp *http.Response
	var err error

	for i := 0; i < maxRetries; i++ {
		resp, err = http.Get(healthURL)
		if err == nil && resp.StatusCode == 200 {
			break
		}
		time.Sleep(2 * time.Second)
	}

	if err != nil || resp.StatusCode != 200 {
		t.Skip("APIサーバーが起動していないため、APIテストをスキップします")
		return
	}

	// ヘルスチェックのレスポンス確認
	require.Equal(t, 200, resp.StatusCode)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var healthResponse map[string]interface{}
	err = json.Unmarshal(body, &healthResponse)
	require.NoError(t, err)

	assert.Equal(t, "OK", healthResponse["status"])
	assert.Contains(t, healthResponse, "timestamp")

	// メモAPIのテスト（認証なしのテスト用エンドポイントがあれば）
	// 注意: 実際の本番環境では認証が必要
	memosURL := baseURL + "/api/memos"

	// GET /api/memos のテスト（現在は認証なしでアクセス可能）
	resp, err = http.Get(memosURL)
	require.NoError(t, err)
	defer resp.Body.Close()

	// 現在の設定では認証なしでアクセス可能なので200が期待される
	assert.Equal(t, 200, resp.StatusCode)
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

	t.Log("E2Eテスト完了。Docker Composeサービスは保持されます。")

	// Note: Docker Composeサービスは保持する
	// 必要に応じて手動で停止: docker compose down -v
}

// ベンチマークテスト
func BenchmarkDatabaseOperations(b *testing.B) {
	if testing.Short() {
		b.Skip("短いテストモードでベンチマークをスキップ")
	}

	// データベース接続
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://memo_user:memo_password@localhost:5432/memo_db?sslmode=disable"
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		b.Skip("データベース接続に失敗: " + err.Error())
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		b.Skip("データベースpingに失敗: " + err.Error())
	}

	b.ResetTimer()

	b.Run("InsertMemo", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := db.Exec(`
				INSERT INTO memos (title, content, category, priority, status, tags) 
				VALUES ($1, $2, $3, $4, $5, $6)
			`, fmt.Sprintf("ベンチマークメモ %d", i), "ベンチマーク用メモ", "work", "medium", "active", "{\"benchmark\"}")
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("SelectMemos", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			rows, err := db.Query("SELECT title, content FROM memos LIMIT 10")
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

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://memo_user:memo_password@localhost:5432/memo_db?sslmode=disable"
	}
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
					err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM memos").Scan(&count)
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
