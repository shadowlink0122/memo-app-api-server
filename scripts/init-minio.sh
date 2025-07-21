#!/bin/bash

# MinIOクライアント（mc）を使用してバケットを作成するスクリプト

echo "MinIOの起動を待機中..."
sleep 10

# MinIOクライアントのエイリアスを設定
mc alias set local http://localhost:9000 minioadmin minioadmin

# バケットが存在しない場合は作成
if ! mc ls local/memo-app-logs > /dev/null 2>&1; then
    echo "バケット 'memo-app-logs' を作成中..."
    mc mb local/memo-app-logs
    echo "バケット作成完了"
else
    echo "バケット 'memo-app-logs' は既に存在します"
fi

# バケットのポリシーを設定（読み取り専用で公開）
mc policy set public local/memo-app-logs

echo "MinIOの初期化が完了しました"
echo "MinIO Console: http://localhost:9001"
echo "ユーザー名: minioadmin"
echo "パスワード: minioadmin"
