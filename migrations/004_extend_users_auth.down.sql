-- ユーザー認証機能拡張の削除（Down Migration）

-- 新しいインデックス削除
DROP INDEX IF EXISTS idx_users_is_active;
DROP INDEX IF EXISTS idx_users_github_id;

-- 新しいカラム削除
ALTER TABLE users DROP COLUMN IF EXISTS created_ip;
ALTER TABLE users DROP COLUMN IF EXISTS last_login_at;
ALTER TABLE users DROP COLUMN IF EXISTS is_active;
ALTER TABLE users DROP COLUMN IF EXISTS avatar_url;
ALTER TABLE users DROP COLUMN IF EXISTS github_username;
ALTER TABLE users DROP COLUMN IF EXISTS github_id;
