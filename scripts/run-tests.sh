#!/bin/bash

# Docker専用テスト実行スクリプト
# このアプリケーションはDocker環境でのみ動作します

set -e  # エラー時に終了

echo "=========================================="
echo "memo-app-api-server Docker専用テストスイート"
echo "=========================================="

# カラー出力用の関数
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

success() {
    echo -e "${GREEN}✓ $1${NC}"
}

error() {
    echo -e "${RED}✗ $1${NC}"
}

warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

info() {
    echo -e "ℹ $1"
}

# 設定
TEST_MODE=${1:-all}  # all, unit, integration, e2e, database
VERBOSE=${VERBOSE:-false}
COVERAGE=${COVERAGE:-false}
TIMEOUT=${TIMEOUT:-5m}

# Docker関連の確認
check_docker() {
    info "Docker環境を確認中..."
    
    if ! command -v docker &> /dev/null; then
        error "Dockerがインストールされていません"
        exit 1
    fi
    
    if ! docker compose version &> /dev/null; then
        error "Docker Composeがインストールされていません"
        exit 1
    fi
    
    DOCKER_VERSION=$(docker --version | awk '{print $3}' | sed 's/,//')
    COMPOSE_VERSION=$(docker compose version --short)
    success "Docker環境: $DOCKER_VERSION, Compose: $COMPOSE_VERSION"
}
# Docker環境の確認とテスト実行
run_docker_tests() {
    info "Docker環境でのテスト実行を開始..."
    
    # Docker Composeが起動しているかチェック
    if ! docker compose ps | grep -q "Up"; then
        warning "Docker Composeサービスが起動していません"
        info "サービスを起動中..."
        docker compose up -d
        sleep 10  # サービス起動待機
    fi
    
    case "$TEST_MODE" in
        "unit")
            run_unit_tests_docker
            ;;
        "integration")
            run_integration_tests_docker
            ;;
        "database")
            run_database_tests_docker
            ;;
        "e2e")
            run_e2e_tests_docker
            ;;
        "all")
            run_unit_tests_docker
            run_integration_tests_docker
            run_database_tests_docker
            run_e2e_tests_docker
            ;;
        *)
            error "無効なテストモード: $TEST_MODE"
            echo "有効な値: unit, integration, database, e2e, all"
            exit 1
            ;;
    esac
}

# ユニットテスト実行（Docker環境）
run_unit_tests_docker() {
    info "Docker環境でユニットテストを実行中..."
    
    local cmd_args=()
    
    if [[ "$VERBOSE" == "true" ]]; then
        cmd_args+=("-v")
    fi
    
    if [[ "$COVERAGE" == "true" ]]; then
        cmd_args+=("-cover" "-coverprofile=coverage-unit.out")
    fi
    
    cmd_args+=("-timeout=$TIMEOUT")
    
    if docker compose exec app go test ./test/config ./test/middleware ./test/logger ./test/storage "${cmd_args[@]}"; then
        success "ユニットテスト成功"
    else
        error "ユニットテスト失敗"
        return 1
    fi
}

# 統合テスト実行（Docker環境）
run_integration_tests_docker() {
    info "Docker環境で統合テストを実行中..."
    
    local cmd_args=()
    
    if [[ "$VERBOSE" == "true" ]]; then
        cmd_args+=("-v")
    fi
    
    if [[ "$COVERAGE" == "true" ]]; then
        cmd_args+=("-cover" "-coverprofile=coverage-integration.out")
    fi
    
    cmd_args+=("-timeout=$TIMEOUT")
    
    if docker compose exec app go test ./test/integration "${cmd_args[@]}"; then
        success "統合テスト完了"
    else
        error "統合テスト失敗"
        return 1
    fi
}

# データベーステスト実行（Docker環境）
run_database_tests_docker() {
    info "Docker環境でデータベーステストを実行中..."
    
    # データベースサービスが起動しているかチェック
    if ! docker compose ps db | grep -q "Up"; then
        warning "データベースサービスが起動していません"
        info "データベースサービスを起動中..."
        docker compose up -d db
        sleep 15  # データベース起動待機
    fi
    
    local cmd_args=()
    
    if [[ "$VERBOSE" == "true" ]]; then
        cmd_args+=("-v")
    fi
    
    cmd_args+=("-timeout=$TIMEOUT")
    
    if docker compose exec app go test ./test/database "${cmd_args[@]}"; then
        success "データベーステスト完了"
    else
        error "データベーステスト失敗"
        return 1
    fi
}

# E2Eテスト実行（Docker環境）
run_e2e_tests_docker() {
    info "Docker環境でE2Eテストを実行中..."
    
    # 全サービスが起動しているかチェック
    services=("app" "db" "minio")
    for service in "${services[@]}"; do
        if ! docker compose ps "$service" | grep -q "Up"; then
            warning "$service サービスが起動していません"
            info "全サービスを起動中..."
            docker compose up -d
            sleep 30  # 全サービス起動待機
            break
        fi
    done
    
    local cmd_args=()
    
    if [[ "$VERBOSE" == "true" ]]; then
        cmd_args+=("-v")
    fi
    
    cmd_args+=("-timeout=10m")  # E2Eテストは時間がかかる
    
    if docker compose exec app go test ./test/e2e "${cmd_args[@]}"; then
        success "E2Eテスト完了"
    else
        error "E2Eテスト失敗"
        return 1
    fi
}

# カバレッジレポート生成（Docker環境）
generate_coverage_report_docker() {
    if [[ "$COVERAGE" != "true" ]]; then
        return 0
    fi
    
    info "Docker環境でカバレッジレポートを生成中..."
    
    if docker compose exec app bash -c "
        echo 'mode: set' > coverage-total.out
        for coverage_file in coverage-*.out; do
            if [[ -f \"\$coverage_file\" ]]; then
                tail -n +2 \"\$coverage_file\" >> coverage-total.out
            fi
        done
        go tool cover -html=coverage-total.out -o coverage.html
        go tool cover -func=coverage-total.out
    "; then
        success "HTMLカバレッジレポート生成: coverage.html"
    fi
}

# メイン実行部分
main() {
    echo ""
    info "⚠️  重要: このアプリケーションはDocker専用です"
    info "テストモード: $TEST_MODE"
    echo ""
    
    check_docker
    run_docker_tests
    
    if [[ "$COVERAGE" == "true" ]]; then
        generate_coverage_report_docker
    fi
    
    success "全テスト完了!"
}

# 使用方法の表示
show_usage() {
    echo "使用方法: $0 [TEST_MODE]"
    echo ""
    echo "⚠️  重要: このスクリプトはDocker環境でのみ動作します"
    echo ""
    echo "TEST_MODE:"
    echo "  unit        - ユニットテストのみ"
    echo "  integration - 統合テストのみ"
    echo "  database    - データベーステストのみ"
    echo "  e2e         - E2Eテストのみ"
    echo "  all         - 全テスト（デフォルト）"
    echo ""
    echo "環境変数:"
    echo "  VERBOSE=true           - 詳細出力"
    echo "  COVERAGE=true          - カバレッジ測定"
    echo "  TIMEOUT=5m             - テストタイムアウト"
    echo ""
    echo "前提条件:"
    echo "  - Docker & Docker Composeがインストール済み"
    echo "  - docker-compose.ymlが存在すること"
    echo ""
    echo "例:"
    echo "  $0 unit                           # ユニットテストのみ"
    echo "  VERBOSE=true $0 all               # 詳細出力で全テスト"
    echo "  COVERAGE=true $0 unit             # カバレッジ付きユニットテスト"
}

# ヘルプオプションの処理
if [[ "$1" == "-h" ]] || [[ "$1" == "--help" ]]; then
    show_usage
    exit 0
fi

# メイン実行
main "$@"
