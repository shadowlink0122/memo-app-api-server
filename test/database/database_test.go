package database

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// データベース接続のテスト
func TestDatabaseConnection(t *testing.T) {
	// テスト用のデータベース接続文字列を取得
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URLが設定されていません。統合テストをスキップします。")
	}

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
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URLが設定されていません。統合テストをスキップします。")
	}

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
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URLが設定されていません。統合テストをスキップします。")
	}

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
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URLが設定されていません。統合テストをスキップします。")
	}

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()

	// testuserの存在確認
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE username = 'testuser'").Scan(&count)
	assert.NoError(t, err, "testuserの確認クエリに失敗")
	assert.Equal(t, 1, count, "testuserが存在しません")

	// サンプルメモの存在確認
	var memoCount int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM memos m 
		JOIN users u ON m.user_id = u.id 
		WHERE u.username = 'testuser' AND m.title = 'サンプルメモ'
	`).Scan(&memoCount)
	assert.NoError(t, err, "サンプルメモの確認クエリに失敗")
	assert.Equal(t, 1, memoCount, "サンプルメモが存在しません")
}

// トリガー動作確認のテスト
func TestUpdateTriggers(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URLが設定されていません。統合テストをスキップします。")
	}

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
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URLが設定されていません。統合テストをスキップします。")
	}

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
