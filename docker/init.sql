-- データベース初期化スクリプト
-- PostgreSQL用のテーブル作成とサンプルデータ挿入

-- 拡張機能を有効化
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ユーザーテーブル
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- メモテーブル
CREATE TABLE IF NOT EXISTS memos (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(200) NOT NULL,
    content TEXT,
    tags TEXT[], -- PostgreSQLの配列型
    is_public BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- インデックスの作成（パフォーマンス向上）
CREATE INDEX IF NOT EXISTS idx_memos_user_id ON memos(user_id);
CREATE INDEX IF NOT EXISTS idx_memos_created_at ON memos(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_memos_tags ON memos USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- 更新日時の自動更新関数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- トリガーの作成
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
CREATE TRIGGER update_users_updated_at 
    BEFORE UPDATE ON users 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_memos_updated_at ON memos;
CREATE TRIGGER update_memos_updated_at 
    BEFORE UPDATE ON memos 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- サンプルデータの挿入（開発用）
INSERT INTO users (username, email, password_hash) VALUES 
    ('testuser', 'test@example.com', '$2a$10$dummy_hash_for_development')
ON CONFLICT (username) DO NOTHING;

-- サンプルメモの挿入
INSERT INTO memos (user_id, title, content, tags, is_public) 
SELECT 
    u.id,
    'サンプルメモ',
    'これはサンプルのメモです。Docker環境での動作確認用です。',
    ARRAY['sample', 'docker', 'memo'],
    TRUE
FROM users u 
WHERE u.username = 'testuser'
ON CONFLICT DO NOTHING;

-- 権限設定
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO memo_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO memo_user;
GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public TO memo_user;
