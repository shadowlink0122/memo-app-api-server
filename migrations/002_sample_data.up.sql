-- サンプルデータ挿入（Up Migration）

-- サンプルユーザー
INSERT INTO users (username, email, password_hash) 
VALUES 
    ('testuser', 'test@example.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi'), -- password: password
    ('admin', 'admin@example.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi'),     -- password: password
    ('demo', 'demo@example.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi')       -- password: password
ON CONFLICT (username) DO NOTHING;

-- サンプルメモ
INSERT INTO memos (title, content, category, tags, priority, status, user_id, is_public)
VALUES 
    ('プロジェクトの計画', 'Goアプリケーションの開発計画を立てる', 'プロジェクト', '["Go", "開発", "計画"]', 'high', 'active', 1, false),
    ('Docker設定', 'Docker Composeの設定を確認する', '技術', '["Docker", "DevOps"]', 'medium', 'active', 1, false),
    ('データベース設計', 'PostgreSQLのテーブル設計を見直す', 'データベース', '["PostgreSQL", "設計"]', 'medium', 'active', 1, false),
    ('API仕様書作成', 'RESTful APIの仕様書を作成する', 'ドキュメント', '["API", "仕様書"]', 'low', 'archived', 1, true),
    ('テスト実装', 'ユニットテストとE2Eテストを実装する', 'テスト', '["テスト", "品質保証"]', 'high', 'active', 2, false),
    ('デプロイ設定', 'EC2へのデプロイ設定を構築する', 'インフラ', '["AWS", "EC2", "デプロイ"]', 'medium', 'active', 2, true)
ON CONFLICT DO NOTHING;
