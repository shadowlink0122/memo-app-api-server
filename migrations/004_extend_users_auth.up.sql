-- 既存のusersテーブルに認証機能用のカラムを追加

-- GitHub認証用カラムの追加
ALTER TABLE users ADD COLUMN IF NOT EXISTS github_id INTEGER UNIQUE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS github_username VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url VARCHAR(500);

-- セキュリティ管理用カラムの追加
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true;
ALTER TABLE users ADD COLUMN IF NOT EXISTS created_ip VARCHAR(45);
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMP WITH TIME ZONE;

-- password_hashをNULLABLEに変更（GitHub認証のみのユーザーのため）
ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;

-- created_at/updated_atをTIMESTAMP WITH TIME ZONEに変更
ALTER TABLE users ALTER COLUMN created_at TYPE TIMESTAMP WITH TIME ZONE;
ALTER TABLE users ALTER COLUMN updated_at TYPE TIMESTAMP WITH TIME ZONE;

-- IP登録管理テーブル
CREATE TABLE IF NOT EXISTS ip_registrations (
    id SERIAL PRIMARY KEY,
    ip_address VARCHAR(45) UNIQUE NOT NULL,
    registration_count INTEGER DEFAULT 0,
    first_registration_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_registration_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 新しいインデックス作成
CREATE INDEX IF NOT EXISTS idx_users_github_id ON users(github_id);
CREATE INDEX IF NOT EXISTS idx_users_created_ip ON users(created_ip);
CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active);
CREATE INDEX IF NOT EXISTS idx_ip_registrations_ip ON ip_registrations(ip_address);
