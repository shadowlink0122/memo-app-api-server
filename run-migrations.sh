#!/bin/bash

# データベースマイグレーション実行スクリプト
# 使用方法: ./run-migrations.sh [up|down] [database_name] [migration_step]

set -e

# デフォルト値
ACTION=${1:-up}
DATABASE=${2:-memo_db}
MIGRATION_STEP=${3:-all}

# データベース接続情報
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-memo_user}
DB_PASSWORD=${DB_PASSWORD:-memo_password}

# カラー出力用
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== データベースマイグレーション実行 ===${NC}"
echo "アクション: $ACTION"
echo "データベース: $DATABASE"
echo "ステップ: $MIGRATION_STEP"
echo ""

# マイグレーションディレクトリの存在確認
MIGRATIONS_DIR="$(dirname "$0")/migrations"
if [ ! -d "$MIGRATIONS_DIR" ]; then
    echo -e "${RED}エラー: マイグレーションディレクトリが見つかりません: $MIGRATIONS_DIR${NC}"
    exit 1
fi

# PostgreSQLへの接続確認
echo -e "${YELLOW}データベース接続を確認中...${NC}"
export PGPASSWORD=$DB_PASSWORD
if ! psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -c '\q' 2>/dev/null; then
    echo -e "${RED}エラー: データベースに接続できません${NC}"
    exit 1
fi

# データベースの存在確認（存在しない場合は作成）
if ! psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -lqt | cut -d \| -f 1 | grep -qw $DATABASE; then
    echo -e "${YELLOW}データベース '$DATABASE' が存在しないため、作成します...${NC}"
    psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -c "CREATE DATABASE $DATABASE;"
fi

# マイグレーション実行関数
run_migration() {
    local file=$1
    local database=$2
    
    echo -e "${YELLOW}実行中: $(basename $file)${NC}"
    if psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $database -f "$file"; then
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
        for file in $MIGRATIONS_DIR/*.up.sql; do
            if [ -f "$file" ]; then
                run_migration "$file" "$DATABASE"
            fi
        done
    else
        # 指定されたステップまでのマイグレーションを実行
        for file in $MIGRATIONS_DIR/00[1-$MIGRATION_STEP]_*.up.sql; do
            if [ -f "$file" ]; then
                run_migration "$file" "$DATABASE"
            fi
        done
    fi
    
elif [ "$ACTION" = "down" ]; then
    echo -e "${GREEN}=== DOWN マイグレーション実行 ===${NC}"
    
    if [ "$MIGRATION_STEP" = "all" ]; then
        # 全てのDOWNマイグレーションを逆順で実行
        for file in $(ls -r $MIGRATIONS_DIR/*.down.sql); do
            if [ -f "$file" ]; then
                run_migration "$file" "$DATABASE"
            fi
        done
    else
        # 指定されたステップからのマイグレーションを逆順で実行
        for file in $(ls -r $MIGRATIONS_DIR/00[$MIGRATION_STEP-9]_*.down.sql); do
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
