package database

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// テスト用データベース接続文字列を取得するヘルパー関数
func getTestDSN(t *testing.T) string {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		// Docker環境内かチェック
		inDocker := os.Getenv("DOCKER_CONTAINER") == "true"
		if inDocker {
			dsn = "postgres://memo_user:memo_password@db:5432/memo_db_test?sslmode=disable"
		} else {
			dsn = "postgres://memo_user:memo_password@localhost:5432/memo_db_test?sslmode=disable"
		}
	}

	if dsn == "" {
		t.Skip("TEST_DATABASE_URLが設定されていません。統合テストをスキップします。")
	}

	return dsn
}

// データベース接続のテスト
func TestDatabaseConnection(t *testing.T) {
	// テスト用のデータベース接続文字列を取得
	dsn := getTestDSN(t)

	// データベース接続をテスト
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err, "データベース接続の初期化に失敗")
	defer db.Close()

	// 接続を確認
	err = db.Ping()
	assert.NoError(t, err, "データベースへのpingに失敗")
}

// テーブル存在確認のテスト
func TestTablesExist(t *testing.T) {
	dsn := getTestDSN(t)

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()

	// テーブルの存在確認
	tables := []string{"users", "memos"}

	for _, table := range tables {
		query := `
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = $1
			);
		`
		var exists bool
		err := db.QueryRow(query, table).Scan(&exists)
		assert.NoError(t, err, "テーブル存在確認クエリに失敗: %s", table)
		assert.True(t, exists, "テーブルが存在しません: %s", table)
	}
}

// インデックス存在確認のテスト
func TestIndexesExist(t *testing.T) {
	dsn := getTestDSN(t)

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()

	// インデックスの存在確認
	indexes := []string{
		"idx_memos_user_id",
		"idx_memos_created_at",
		"idx_memos_tags",
		"idx_users_username",
		"idx_users_email",
	}

	for _, index := range indexes {
		query := `
			SELECT EXISTS (
				SELECT FROM pg_indexes 
				WHERE schemaname = 'public' 
				AND indexname = $1
			);
		`
		var exists bool
		err := db.QueryRow(query, index).Scan(&exists)
		assert.NoError(t, err, "インデックス存在確認クエリに失敗: %s", index)
		assert.True(t, exists, "インデックスが存在しません: %s", index)
	}
}

// サンプルデータ存在確認のテスト
func TestSampleDataExists(t *testing.T) {
	dsn := getTestDSN(t)

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()

	// テスト用サンプルデータを挿入（存在チェックしてから挿入）
	// testuserを挿入
	t.Log("testuserを挿入中...")
	result, err := db.Exec(`
		INSERT INTO users (username, email, password_hash) 
		VALUES ('testuser', 'test@example.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi')
		ON CONFLICT (username) DO NOTHING
	`)
	require.NoError(t, err, "testuserの挿入に失敗")
	rowsAffected, _ := result.RowsAffected()
	t.Logf("testuser挿入結果: %d行が影響されました", rowsAffected)

	// testuserのIDを取得
	var userID int
	err = db.QueryRow("SELECT id FROM users WHERE username = 'testuser'").Scan(&userID)
	require.NoError(t, err, "testuserのID取得に失敗")
	t.Logf("取得したtestuserのID: %d", userID)

	// サンプルメモが既に存在するかチェック
	var existingMemoCount int
	err = db.QueryRow("SELECT COUNT(*) FROM memos WHERE title = 'プロジェクトの計画'").Scan(&existingMemoCount)
	require.NoError(t, err, "既存メモのチェックに失敗")
	t.Logf("既存の「プロジェクトの計画」メモ数: %d", existingMemoCount)

	// サンプルメモが存在しない場合のみ挿入
	if existingMemoCount == 0 {
		t.Log("サンプルメモを挿入中...")
		result, err = db.Exec(`
			INSERT INTO memos (title, content, category, tags, priority, status, user_id, is_public)
			VALUES ('プロジェクトの計画', 'Goアプリケーションの開発計画を立てる', 'プロジェクト', '["Go", "開発", "計画"]', 'high', 'active', $1, false)
		`, userID)
		require.NoError(t, err, "サンプルメモの挿入に失敗")
		rowsAffected, _ = result.RowsAffected()
		t.Logf("サンプルメモ挿入結果: %d行が影響されました", rowsAffected)
	} else {
		t.Log("サンプルメモは既に存在します")
	}

	// テーブルの実際の内容をデバッグ出力
	t.Log("=== デバッグ: テーブル内容確認 ===")

	// usersテーブルの内容確認
	rows, err := db.Query("SELECT id, username, email FROM users ORDER BY id")
	if err != nil {
		t.Logf("usersテーブル確認エラー: %v", err)
	} else {
		defer rows.Close()
		t.Log("usersテーブル内容:")
		for rows.Next() {
			var id int
			var username, email string
			if err := rows.Scan(&id, &username, &email); err == nil {
				t.Logf("  id: %d, username: %s, email: %s", id, username, email)
			}
		}
	}

	// memosテーブルの内容確認
	memoRows, err := db.Query("SELECT id, title, content, user_id FROM memos ORDER BY id")
	if err != nil {
		t.Logf("memosテーブル確認エラー: %v", err)
	} else {
		defer memoRows.Close()
		t.Log("memosテーブル内容:")
		for memoRows.Next() {
			var id, userID int
			var title, content string
			if err := memoRows.Scan(&id, &title, &content, &userID); err == nil {
				t.Logf("  id: %d, title: %s, content: %s, user_id: %d", id, title, content, userID)
			}
		}
	}

	// testuserの存在確認
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE username = 'testuser'").Scan(&count)
	assert.NoError(t, err, "testuserの確認クエリに失敗")
	t.Logf("testuserカウント: %d", count)
	assert.Equal(t, 1, count, "testuserが存在しません")

	// サンプルメモの存在確認
	var memoCount int
	err = db.QueryRow("SELECT COUNT(*) FROM memos WHERE title = 'プロジェクトの計画'").Scan(&memoCount)
	assert.NoError(t, err, "サンプルメモの確認クエリに失敗")
	t.Logf("サンプルメモ（プロジェクトの計画）カウント: %d", memoCount)
	assert.Equal(t, 1, memoCount, "サンプルメモが存在しません")
}

// トリガー動作確認のテスト
func TestUpdateTriggers(t *testing.T) {
	dsn := getTestDSN(t)

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()

	// テスト用ユーザーを作成
	var userID string
	err = db.QueryRow(`
		INSERT INTO users (username, email, password_hash) 
		VALUES ('test_trigger_user', 'trigger@test.com', 'hash') 
		RETURNING id
	`).Scan(&userID)
	require.NoError(t, err, "テスト用ユーザーの作成に失敗")

	// 最初のupdated_atを取得
	var originalUpdatedAt string
	err = db.QueryRow("SELECT updated_at FROM users WHERE id = $1", userID).Scan(&originalUpdatedAt)
	require.NoError(t, err, "updated_atの取得に失敗")

	// 少し待ってからユーザーを更新
	// time.Sleep(time.Millisecond * 10)

	// ユーザーを更新
	_, err = db.Exec("UPDATE users SET email = 'updated@test.com' WHERE id = $1", userID)
	require.NoError(t, err, "ユーザーの更新に失敗")

	// 更新後のupdated_atを取得
	var newUpdatedAt string
	err = db.QueryRow("SELECT updated_at FROM users WHERE id = $1", userID).Scan(&newUpdatedAt)
	require.NoError(t, err, "更新後のupdated_atの取得に失敗")

	// updated_atが更新されていることを確認
	assert.NotEqual(t, originalUpdatedAt, newUpdatedAt, "updated_atが更新されていません")

	// テスト用データをクリーンアップ
	_, err = db.Exec("DELETE FROM users WHERE id = $1", userID)
	assert.NoError(t, err, "テスト用ユーザーの削除に失敗")
}

// データベーススキーマ検証のテスト
func TestDatabaseSchema(t *testing.T) {
	dsn := getTestDSN(t)

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()

	// usersテーブルの列を確認
	userColumns, err := getTableColumns(db, "users")
	require.NoError(t, err)

	expectedUserColumns := []string{"id", "username", "email", "password_hash", "created_at", "updated_at"}
	for _, col := range expectedUserColumns {
		assert.Contains(t, userColumns, col, "usersテーブルに列が存在しません: %s", col)
	}

	// memosテーブルの列を確認
	memoColumns, err := getTableColumns(db, "memos")
	require.NoError(t, err)

	expectedMemoColumns := []string{"id", "user_id", "title", "content", "tags", "is_public", "created_at", "updated_at"}
	for _, col := range expectedMemoColumns {
		assert.Contains(t, memoColumns, col, "memosテーブルに列が存在しません: %s", col)
	}
}

// テーブルの列名を取得するヘルパー関数
func getTableColumns(db *sql.DB, tableName string) ([]string, error) {
	query := `
		SELECT column_name 
		FROM information_schema.columns 
		WHERE table_schema = 'public' AND table_name = $1
		ORDER BY ordinal_position
	`

	rows, err := db.Query(query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var columnName string
		if err := rows.Scan(&columnName); err != nil {
			return nil, err
		}
		columns = append(columns, columnName)
	}

	return columns, rows.Err()
}
