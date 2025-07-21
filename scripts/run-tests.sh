#!/bin/bash

# テスト実行スクリプト
# CI/CD環境やローカルでのテスト実行に使用

set -e  # エラー時に終了

echo "======================================"
echo "memo-app-api-server テストスイート"
echo "======================================"

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

# Go関連の確認
check_go() {
    info "Go環境を確認中..."
    
    if ! command -v go &> /dev/null; then
        error "Goがインストールされていません"
        exit 1
    fi
    
    GO_VERSION=$(go version | awk '{print $3}')
    success "Go環境: $GO_VERSION"
}

# 依存関係の確認
check_dependencies() {
    info "依存関係を確認中..."
    
    go mod tidy
    go mod verify
    
    success "依存関係の確認完了"
}

# ユニットテスト実行
run_unit_tests() {
    info "ユニットテストを実行中..."
    
    local test_dirs=("config" "middleware" "logger" "storage")
    local cmd_args=()
    
    if [[ "$VERBOSE" == "true" ]]; then
        cmd_args+=("-v")
    fi
    
    if [[ "$COVERAGE" == "true" ]]; then
        cmd_args+=("-cover" "-coverprofile=coverage-unit.out")
    fi
    
    cmd_args+=("-timeout=$TIMEOUT")
    
    for dir in "${test_dirs[@]}"; do
        info "テスト実行: test/$dir"
        if go test "./test/$dir" "${cmd_args[@]}"; then
            success "ユニットテスト成功: $dir"
        else
            error "ユニットテスト失敗: $dir"
            return 1
        fi
    done
    
    success "全ユニットテスト完了"
}

# 統合テスト実行
run_integration_tests() {
    info "統合テストを実行中..."
    
    local cmd_args=()
    
    if [[ "$VERBOSE" == "true" ]]; then
        cmd_args+=("-v")
    fi
    
    if [[ "$COVERAGE" == "true" ]]; then
        cmd_args+=("-cover" "-coverprofile=coverage-integration.out")
    fi
    
    cmd_args+=("-timeout=$TIMEOUT")
    
    if go test "./test/integration" "${cmd_args[@]}"; then
        success "統合テスト完了"
    else
        error "統合テスト失敗"
        return 1
    fi
}

# データベーステスト実行
run_database_tests() {
    info "データベーステストを実行中..."
    
    # データベース接続の確認
    if [[ -z "$TEST_DATABASE_URL" ]]; then
        warning "TEST_DATABASE_URLが設定されていません。データベーステストをスキップします。"
        return 0
    fi
    
    local cmd_args=()
    
    if [[ "$VERBOSE" == "true" ]]; then
        cmd_args+=("-v")
    fi
    
    cmd_args+=("-timeout=$TIMEOUT")
    
    if TEST_DATABASE_URL="$TEST_DATABASE_URL" go test "./test/database" "${cmd_args[@]}"; then
        success "データベーステスト完了"
    else
        error "データベーステスト失敗"
        return 1
    fi
}

# E2Eテスト実行
run_e2e_tests() {
    info "E2Eテストを実行中..."
    
    # Docker環境の確認
    if ! command -v docker &> /dev/null; then
        warning "Dockerがインストールされていません。E2Eテストをスキップします。"
        return 0
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        warning "Docker Composeがインストールされていません。E2Eテストをスキップします。"
        return 0
    fi
    
    local cmd_args=()
    
    if [[ "$VERBOSE" == "true" ]]; then
        cmd_args+=("-v")
    fi
    
    cmd_args+=("-timeout=10m")  # E2Eテストは時間がかかる
    
    if go test "./test/e2e" "${cmd_args[@]}"; then
        success "E2Eテスト完了"
    else
        error "E2Eテスト失敗"
        return 1
    fi
}

# API テスト実行
run_api_tests() {
    info "APIテストを実行中..."
    
    local cmd_args=()
    
    if [[ "$VERBOSE" == "true" ]]; then
        cmd_args+=("-v")
    fi
    
    if [[ "$COVERAGE" == "true" ]]; then
        cmd_args+=("-cover" "-coverprofile=coverage-api.out")
    fi
    
    cmd_args+=("-timeout=$TIMEOUT")
    
    if go test "./test" "${cmd_args[@]}"; then
        success "APIテスト完了"
    else
        error "APIテスト失敗"
        return 1
    fi
}

# カバレッジレポート生成
generate_coverage_report() {
    if [[ "$COVERAGE" != "true" ]]; then
        return 0
    fi
    
    info "カバレッジレポートを生成中..."
    
    # カバレッジファイルを結合
    echo "mode: set" > coverage-total.out
    for coverage_file in coverage-*.out; do
        if [[ -f "$coverage_file" ]]; then
            tail -n +2 "$coverage_file" >> coverage-total.out
        fi
    done
    
    # HTMLレポートを生成
    if go tool cover -html=coverage-total.out -o coverage.html; then
        success "HTMLカバレッジレポート生成: coverage.html"
    fi
    
    # 関数別カバレッジを表示
    if [[ "$VERBOSE" == "true" ]]; then
        info "関数別カバレッジ:"
        go tool cover -func=coverage-total.out
    fi
    
    # 総合カバレッジ率を計算・表示
    local total_coverage=$(go tool cover -func=coverage-total.out | tail -1 | awk '{print $3}')
    success "総合カバレッジ率: $total_coverage"
}

# メイン実行部分
main() {
    info "テストモード: $TEST_MODE"
    
    check_go
    check_dependencies
    
    case "$TEST_MODE" in
        "unit")
            run_unit_tests
            ;;
        "integration")
            run_integration_tests
            ;;
        "database")
            run_database_tests
            ;;
        "e2e")
            run_e2e_tests
            ;;
        "api")
            run_api_tests
            ;;
        "all")
            run_unit_tests
            run_api_tests
            run_integration_tests
            run_database_tests
            run_e2e_tests
            ;;
        *)
            error "無効なテストモード: $TEST_MODE"
            echo "利用可能なモード: unit, integration, database, e2e, api, all"
            exit 1
            ;;
    esac
    
    generate_coverage_report
    
    success "全テスト完了!"
}

# 使用方法の表示
show_usage() {
    echo "使用方法: $0 [TEST_MODE]"
    echo ""
    echo "TEST_MODE:"
    echo "  unit        - ユニットテストのみ"
    echo "  integration - 統合テストのみ"
    echo "  database    - データベーステストのみ"
    echo "  e2e         - E2Eテストのみ"
    echo "  api         - APIテストのみ"
    echo "  all         - 全テスト（デフォルト）"
    echo ""
    echo "環境変数:"
    echo "  VERBOSE=true           - 詳細出力"
    echo "  COVERAGE=true          - カバレッジ測定"
    echo "  TIMEOUT=5m             - テストタイムアウト"
    echo "  TEST_DATABASE_URL=...  - データベーステスト用接続文字列"
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
