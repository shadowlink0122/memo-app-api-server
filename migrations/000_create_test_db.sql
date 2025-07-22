-- テスト用データベース作成スクリプト
-- このスクリプトはPostgreSQLコンテナの初期化時に最初に実行されます

-- テスト用データベースを作成（存在しない場合のみ）
SELECT 'CREATE DATABASE memo_db_test'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'memo_db_test')\gexec

-- ログ出力
\echo 'Test database memo_db_test created or already exists';
