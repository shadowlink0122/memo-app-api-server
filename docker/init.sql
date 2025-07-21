-- データベース初期化スクリプト
-- PostgreSQL用のデータベース作成とマイグレーション実行

-- ログ出力
\echo 'Starting database initialization...'

-- テスト用データベースを作成
CREATE DATABASE memo_db_test;
\echo 'Test database created: memo_db_test'

-- メインデータベースのマイグレーション実行
\c memo_db;
\echo 'Connected to main database: memo_db'

-- 拡張機能を有効化
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ユーザーテーブル
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- メモテーブル
CREATE TABLE IF NOT EXISTS memos (
    id SERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    content TEXT NOT NULL,
    category VARCHAR(50),
    tags JSONB DEFAULT '[]'::jsonb,
    priority VARCHAR(10) NOT NULL DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high')),
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived')),
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE
);

-- テスト用データベースのマイグレーション実行
\c memo_db_test;
\echo 'Connected to test database: memo_db_test'

-- テスト用データベースに拡張機能を追加
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ユーザーテーブル（テスト用）
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- メモテーブル（テスト用）
CREATE TABLE IF NOT EXISTS memos (
    id SERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    content TEXT NOT NULL,
    category VARCHAR(50),
    tags JSONB DEFAULT '[]'::jsonb,
    priority VARCHAR(10) NOT NULL DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high')),
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived')),
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE
);

-- メインデータベースに戻る
\c memo_db;

-- インデックスの作成（パフォーマンス向上）
CREATE INDEX IF NOT EXISTS idx_memos_status ON memos (status);
CREATE INDEX IF NOT EXISTS idx_memos_category ON memos (category);
CREATE INDEX IF NOT EXISTS idx_memos_priority ON memos (priority);
CREATE INDEX IF NOT EXISTS idx_memos_created_at ON memos (created_at);
CREATE INDEX IF NOT EXISTS idx_memos_updated_at ON memos (updated_at);
CREATE INDEX IF NOT EXISTS idx_memos_tags ON memos USING GIN (tags);

-- 全文検索用インデックス（タイトルとコンテンツ）
CREATE INDEX IF NOT EXISTS idx_memos_search ON memos USING GIN (to_tsvector('japanese', title || ' ' || content));

-- 更新日時の自動更新関数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- テスト用データベースにもインデックスと関数を追加
\c memo_db_test;

-- インデックスの作成（パフォーマンス向上）
CREATE INDEX IF NOT EXISTS idx_memos_status ON memos (status);
CREATE INDEX IF NOT EXISTS idx_memos_category ON memos (category);
CREATE INDEX IF NOT EXISTS idx_memos_priority ON memos (priority);
CREATE INDEX IF NOT EXISTS idx_memos_created_at ON memos (created_at);
CREATE INDEX IF NOT EXISTS idx_memos_updated_at ON memos (updated_at);
CREATE INDEX IF NOT EXISTS idx_memos_tags ON memos USING GIN (tags);

-- 全文検索用インデックス（タイトルとコンテンツ）
CREATE INDEX IF NOT EXISTS idx_memos_search ON memos USING GIN (to_tsvector('japanese', title || ' ' || content));

-- 更新日時の自動更新関数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- メインデータベースに戻る
\c memo_db;

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

-- テスト用データベースにもトリガーを追加
\c memo_db_test;

-- トリガーの作成
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP TRIGGER IF EXISTS update_memos_updated_at ON memos;
CREATE TRIGGER update_memos_updated_at 
    BEFORE UPDATE ON memos 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- テスト用データベースにもサンプルデータを挿入
INSERT INTO users (username, email, password_hash) 
VALUES 
    ('testuser', 'test@example.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi'), -- password: password
    ('admin', 'admin@example.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi'),     -- password: password
    ('demo', 'demo@example.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi')       -- password: password
ON CONFLICT (username) DO NOTHING;

INSERT INTO memos (title, content, category, tags, priority, status, user_id) 
VALUES 
    ('プロジェクトの計画', 'Goアプリケーションの開発計画を立てる', 'プロジェクト', '["Go", "開発", "計画"]', 'high', 'active', 1),
    ('Docker設定', 'Docker Composeの設定を確認する', '技術', '["Docker", "DevOps"]', 'medium', 'active', 1),
    ('データベース設計', 'PostgreSQLのテーブル設計を見直す', 'データベース', '["PostgreSQL", "設計"]', 'medium', 'active', 1),
    ('最初のメモ', 'これは最初のテストメモです。', 'テスト', '["サンプル", "テスト"]', 'medium', 'active', 1),
    ('重要なタスク', 'これは重要なタスクのメモです。', '仕事', '["重要", "タスク"]', 'high', 'active', 1),
    ('買い物リスト', '牛乳、パン、卵を買う', '個人', '["買い物", "日用品"]', 'low', 'active', 1),
    ('会議のメモ', '明日の会議の議題について', '仕事', '["会議", "議題"]', 'medium', 'active', 1),
    ('アーカイブされたメモ', 'これは完了したタスクです。', '完了', '["完了", "アーカイブ"]', 'low', 'archived', 1)
ON CONFLICT DO NOTHING;

-- メインデータベースに戻る
\c memo_db;

-- サンプルデータの挿入（開発用）
-- まずサンプルユーザーを挿入
INSERT INTO users (username, email, password_hash) 
VALUES 
    ('testuser', 'test@example.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi'), -- password: password
    ('admin', 'admin@example.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi'),     -- password: password
    ('demo', 'demo@example.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi')       -- password: password
ON CONFLICT (username) DO NOTHING;

-- サンプルメモを挿入（テストで期待されるデータを含む）
INSERT INTO memos (title, content, category, tags, priority, status, user_id) 
VALUES 
    ('プロジェクトの計画', 'Goアプリケーションの開発計画を立てる', 'プロジェクト', '["Go", "開発", "計画"]', 'high', 'active', 1),
    ('Docker設定', 'Docker Composeの設定を確認する', '技術', '["Docker", "DevOps"]', 'medium', 'active', 1),
    ('データベース設計', 'PostgreSQLのテーブル設計を見直す', 'データベース', '["PostgreSQL", "設計"]', 'medium', 'active', 1),
    ('最初のメモ', 'これは最初のテストメモです。', 'テスト', '["サンプル", "テスト"]', 'medium', 'active', 1),
    ('重要なタスク', 'これは重要なタスクのメモです。', '仕事', '["重要", "タスク"]', 'high', 'active', 1),
    ('買い物リスト', '牛乳、パン、卵を買う', '個人', '["買い物", "日用品"]', 'low', 'active', 1),
    ('会議のメモ', '明日の会議の議題について', '仕事', '["会議", "議題"]', 'medium', 'active', 1),
    ('アーカイブされたメモ', 'これは完了したタスクです。', '完了', '["完了", "アーカイブ"]', 'low', 'archived', 1)
ON CONFLICT DO NOTHING;

-- 権限設定（環境に合わせて調整）
-- GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO memo_user;
-- GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO memo_user;
-- GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public TO memo_user;
