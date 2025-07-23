//go:build integration
// +build integration

package repository_test

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	"memo-app/src/repository"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUserRepository_Integration 実際のデータベースを使った統合テスト
func TestUserRepository_Integration(t *testing.T) {
	// 統合テスト用の環境変数チェック
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}

	dbURL := fmt.Sprintf("postgres://memo_user:memo_password@%s:5432/memo_db?sslmode=disable", dbHost)

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Skipf("データベース接続に失敗: %v", err)
	}
	defer db.Close()

	// 接続テスト
	if err := db.Ping(); err != nil {
		t.Skipf("データベースに接続できません: %v", err)
	}

	repo := repository.NewUserRepository(db)

	t.Run("rootユーザーの取得", func(t *testing.T) {
		user, err := repo.GetByUsername("root")
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "root", user.Username)
		assert.Equal(t, "root@example.com", user.Email)
		assert.True(t, user.IsActive)
		t.Logf("取得したrootユーザー: ID=%d, Username=%s, Email=%s", user.ID, user.Username, user.Email)
	})

	t.Run("rootユーザーのメールアドレスでの取得", func(t *testing.T) {
		user, err := repo.GetByEmail("root@example.com")
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "root", user.Username)
		assert.Equal(t, "root@example.com", user.Email)
		t.Logf("メールで取得したrootユーザー: ID=%d, Username=%s", user.ID, user.Username)
	})

	t.Run("ユーザー名存在確認", func(t *testing.T) {
		exists, err := repo.IsUsernameExists("root")
		require.NoError(t, err)
		assert.True(t, exists)
		t.Log("rootユーザーの存在が確認されました")

		exists, err = repo.IsUsernameExists("nonexistent")
		require.NoError(t, err)
		assert.False(t, exists)
		t.Log("存在しないユーザーは正しく false を返します")
	})

	t.Run("メールアドレス存在確認", func(t *testing.T) {
		exists, err := repo.IsEmailExists("root@example.com")
		require.NoError(t, err)
		assert.True(t, exists)
		t.Log("rootユーザーのメールアドレスの存在が確認されました")

		exists, err = repo.IsEmailExists("nonexistent@example.com")
		require.NoError(t, err)
		assert.False(t, exists)
		t.Log("存在しないメールアドレスは正しく false を返します")
	})

	t.Run("最終ログイン時刻の更新", func(t *testing.T) {
		// rootユーザーを取得
		user, err := repo.GetByUsername("root")
		require.NoError(t, err)

		// 最終ログイン時刻を更新
		err = repo.UpdateLastLogin(user.ID)
		require.NoError(t, err)
		t.Logf("rootユーザー(ID=%d)の最終ログイン時刻を更新しました", user.ID)

		// 更新後のユーザー情報を取得
		updatedUser, err := repo.GetByID(user.ID)
		require.NoError(t, err)
		assert.NotNil(t, updatedUser.LastLoginAt)
		t.Logf("更新後の最終ログイン時刻: %v", updatedUser.LastLoginAt)
	})
}
