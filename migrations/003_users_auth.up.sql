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

-- IP登録管理テーブル
CREATE TABLE IF NOT EXISTS ip_registrations (
    id SERIAL PRIMARY KEY,
    ip_address VARCHAR(45) UNIQUE NOT NULL,
    user_count INTEGER DEFAULT 0,
    last_used_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- インデックス作成（カラム追加後）
CREATE INDEX IF NOT EXISTS idx_users_github_id ON users(github_id);
CREATE INDEX IF NOT EXISTS idx_users_created_ip ON users(created_ip);
CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active);
CREATE INDEX IF NOT EXISTS idx_ip_registrations_ip ON ip_registrations(ip_address);

-- updated_atの自動更新トリガー（既存テーブル向け）
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 既存のトリガーがある場合は削除してから作成
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
CREATE TRIGGER update_users_updated_at 
    BEFORE UPDATE ON users 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
