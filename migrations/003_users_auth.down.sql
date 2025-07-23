-- トリガーを削除
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- ファンクションを削除
DROP FUNCTION IF EXISTS update_updated_at_column();

-- インデックスを削除
DROP INDEX IF EXISTS idx_users_username;
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_github_id;
DROP INDEX IF EXISTS idx_users_created_ip;
DROP INDEX IF EXISTS idx_users_is_active;
DROP INDEX IF EXISTS idx_ip_registrations_ip;

-- テーブルを削除
DROP TABLE IF EXISTS ip_registrations;
DROP TABLE IF EXISTS users;
