#!/bin/bash

# Docker環境でのマイグレーション実行スクリプト
# Docker Composeコンテナ内でマイグレーションを実行します

set -e

# デフォルト値
ACTION=${1:-up}
DATABASE=${2:-memo_db}
MIGRATION_STEP=${3:-all}

# カラー出力用
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Docker環境でのマイグレーション実行 ===${NC}"
echo "アクション: $ACTION"
echo "データベース: $DATABASE"
echo "ステップ: $MIGRATION_STEP"
echo ""

# Docker Composeサービスが起動しているか確認
if ! docker compose ps db | grep -q "running"; then
    echo -e "${YELLOW}データベースコンテナを起動中...${NC}"
    docker compose up -d db
    echo -e "${YELLOW}データベースの準備を待機中...${NC}"
    sleep 10
fi

# マイグレーションディレクトリの存在確認
if [ ! -d "./migrations" ]; then
    echo -e "${RED}エラー: マイグレーションディレクトリが見つかりません: ./migrations${NC}"
    exit 1
fi

# データベースの存在確認（存在しない場合は作成）
echo -e "${YELLOW}データベース '$DATABASE' の存在確認...${NC}"
if ! docker compose exec -T db psql -U memo_user -d postgres -lqt | cut -d \| -f 1 | grep -qw $DATABASE; then
    echo -e "${YELLOW}データベース '$DATABASE' が存在しないため、作成します...${NC}"
    docker compose exec -T db psql -U memo_user -d postgres -c "CREATE DATABASE $DATABASE;"
fi

# マイグレーション実行関数
run_migration() {
    local file=$1
    local database=$2
    
    echo -e "${YELLOW}実行中: $(basename $file)${NC}"
    if docker compose exec -T db psql -U memo_user -d $database < "$file"; then
        echo -e "${GREEN}成功: $(basename $file)${NC}"
    else
        echo -e "${RED}失敗: $(basename $file)${NC}"
        return 1
    fi
}

# マイグレーション実行
if [ "$ACTION" = "up" ]; then
    echo -e "${GREEN}=== UP マイグレーション実行 ===${NC}"
    
    if [ "$MIGRATION_STEP" = "all" ]; then
        # 全てのUPマイグレーションを順番に実行
        for file in ./migrations/*.up.sql; do
            if [ -f "$file" ]; then
                run_migration "$file" "$DATABASE"
            fi
        done
    else
        # 指定されたステップまでのマイグレーションを実行
        for file in ./migrations/00[1-$MIGRATION_STEP]_*.up.sql; do
            if [ -f "$file" ]; then
                run_migration "$file" "$DATABASE"
            fi
        done
    fi
    
elif [ "$ACTION" = "down" ]; then
    echo -e "${GREEN}=== DOWN マイグレーション実行 ===${NC}"
    
    if [ "$MIGRATION_STEP" = "all" ]; then
        # 全てのDOWNマイグレーションを逆順で実行
        for file in $(ls -r ./migrations/*.down.sql); do
            if [ -f "$file" ]; then
                run_migration "$file" "$DATABASE"
            fi
        done
    else
        # 指定されたステップからのマイグレーションを逆順で実行
        for file in $(ls -r ./migrations/00[$MIGRATION_STEP-9]_*.down.sql); do
            if [ -f "$file" ]; then
                run_migration "$file" "$DATABASE"
            fi
        done
    fi
    
else
    echo -e "${RED}エラー: 無効なアクション '$ACTION'. 'up' または 'down' を指定してください${NC}"
    exit 1
fi

echo -e "${GREEN}=== マイグレーション完了 ===${NC}"
