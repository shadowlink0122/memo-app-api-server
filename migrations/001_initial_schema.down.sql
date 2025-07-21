-- 初期データベーススキーマ削除（Down Migration）

-- トリガー削除
DROP TRIGGER IF EXISTS update_memos_updated_at ON memos;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- トリガー関数削除
DROP FUNCTION IF EXISTS update_updated_at_column();

-- インデックス削除
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_username;
DROP INDEX IF EXISTS idx_memos_tags;
DROP INDEX IF EXISTS idx_memos_priority;
DROP INDEX IF EXISTS idx_memos_status;
DROP INDEX IF EXISTS idx_memos_created_at;
DROP INDEX IF EXISTS idx_memos_user_id;

-- テーブル削除
DROP TABLE IF EXISTS memos;
DROP TABLE IF EXISTS users;

-- 拡張機能削除
DROP EXTENSION IF EXISTS "uuid-ossp";
