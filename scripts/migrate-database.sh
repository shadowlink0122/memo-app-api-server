#!/bin/bash

# =============================================================================
# Database Migration Script
# =============================================================================
# このスクリプトは、PostgreSQLデータベースにマイグレーションファイルを適用します。
# migrations/ ディレクトリ内の *.sql ファイルを順次実行します。
#
# Usage:
#   ./scripts/migrate-database.sh [OPTIONS]
#
# Options:
#   -h, --host HOST          PostgreSQL host (default: localhost)
#   -p, --port PORT          PostgreSQL port (default: 5432)
#   -U, --username USER      PostgreSQL username (default: memo_user)
#   -d, --database DATABASE  PostgreSQL database name (default: memo_db)
#   -m, --migrations-dir DIR Migrations directory (default: migrations)
#   --password PASSWORD      PostgreSQL password (optional, uses PGPASSWORD if not set)
#   --test-db                Create and migrate test database (memo_db_test)
#   --main-db                Migrate main database only (default)
#   --both                   Migrate both main and test databases
#   --dry-run               Show SQL files that would be executed without running them
#   --verbose               Show verbose output
#   --help                  Show this help message
#
# Environment Variables:
#   PGPASSWORD              PostgreSQL password (alternative to --password)
#   DATABASE_URL            Full database connection string (overrides individual options)
#
# Examples:
#   # Migrate main database with default settings
#   ./scripts/migrate-database.sh
#
#   # Migrate both main and test databases
#   ./scripts/migrate-database.sh --both
#
#   # Migrate with custom connection settings
#   ./scripts/migrate-database.sh -h db.example.com -p 5433 -U myuser -d mydb
#
#   # Show what would be executed without running
#   ./scripts/migrate-database.sh --dry-run --verbose
# =============================================================================

set -euo pipefail

# デフォルト設定
DEFAULT_HOST="localhost"
DEFAULT_PORT="5432"
DEFAULT_USERNAME="memo_user"
DEFAULT_DATABASE="memo_db"
DEFAULT_MIGRATIONS_DIR="migrations"

# 設定変数
HOST="${DEFAULT_HOST}"
PORT="${DEFAULT_PORT}"
USERNAME="${DEFAULT_USERNAME}"
DATABASE="${DEFAULT_DATABASE}"
MIGRATIONS_DIR="${DEFAULT_MIGRATIONS_DIR}"
PASSWORD=""
MIGRATE_TEST_DB=false
MIGRATE_MAIN_DB=true
DRY_RUN=false
VERBOSE=false

# 色付きログ出力
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

log_verbose() {
    if [[ "${VERBOSE}" == "true" ]]; then
        echo -e "${BLUE}[VERBOSE]${NC} $1"
    fi
}

# ヘルプ表示
show_help() {
    sed -n '/^# Usage:/,/^# =============/p' "$0" | sed 's/^# //g' | head -n -1
}

# 引数解析
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--host)
                HOST="$2"
                shift 2
                ;;
            -p|--port)
                PORT="$2"
                shift 2
                ;;
            -U|--username)
                USERNAME="$2"
                shift 2
                ;;
            -d|--database)
                DATABASE="$2"
                shift 2
                ;;
            -m|--migrations-dir)
                MIGRATIONS_DIR="$2"
                shift 2
                ;;
            --password)
                PASSWORD="$2"
                shift 2
                ;;
            --test-db)
                MIGRATE_TEST_DB=true
                MIGRATE_MAIN_DB=false
                shift
                ;;
            --main-db)
                MIGRATE_MAIN_DB=true
                MIGRATE_TEST_DB=false
                shift
                ;;
            --both)
                MIGRATE_MAIN_DB=true
                MIGRATE_TEST_DB=true
                shift
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --verbose)
                VERBOSE=true
                shift
                ;;
            --help)
                show_help
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                echo "Use --help for usage information."
                exit 1
                ;;
        esac
    done
}

# パスワード設定
setup_password() {
    if [[ -n "${PASSWORD}" ]]; then
        export PGPASSWORD="${PASSWORD}"
        log_verbose "Using password from --password option"
    elif [[ -n "${PGPASSWORD:-}" ]]; then
        log_verbose "Using password from PGPASSWORD environment variable"
    else
        log_warning "No password specified. This may fail if PostgreSQL requires authentication."
    fi
}

# DATABASE_URL からの設定解析
parse_database_url() {
    if [[ -n "${DATABASE_URL:-}" ]]; then
        log_info "Using DATABASE_URL: ${DATABASE_URL}"
        # DATABASE_URL形式: postgres://user:password@host:port/database
        if [[ "${DATABASE_URL}" =~ postgres://([^:]+):([^@]+)@([^:]+):([^/]+)/(.+) ]]; then
            USERNAME="${BASH_REMATCH[1]}"
            export PGPASSWORD="${BASH_REMATCH[2]}"
            HOST="${BASH_REMATCH[3]}"
            PORT="${BASH_REMATCH[4]}"
            DATABASE="${BASH_REMATCH[5]}"
            # URLパラメータを除去
            DATABASE="${DATABASE%%\?*}"
            log_verbose "Parsed from DATABASE_URL: ${USERNAME}@${HOST}:${PORT}/${DATABASE}"
        else
            log_warning "Invalid DATABASE_URL format. Using default settings."
        fi
    fi
}

# PostgreSQL接続テスト
test_connection() {
    local db_name="$1"
    log_verbose "Testing connection to database: ${db_name}"
    
    if psql -h "${HOST}" -p "${PORT}" -U "${USERNAME}" -d "${db_name}" -c "SELECT 1;" > /dev/null 2>&1; then
        log_verbose "Connection to ${db_name} successful"
        return 0
    else
        log_error "Failed to connect to database: ${db_name}"
        return 1
    fi
}

# マイグレーションファイルの取得
get_migration_files() {
    local files=()
    
    if [[ ! -d "${MIGRATIONS_DIR}" ]]; then
        log_error "Migrations directory not found: ${MIGRATIONS_DIR}"
        exit 1
    fi
    
    # .sqlファイルを名前順にソート
    while IFS= read -r -d '' file; do
        files+=("$file")
    done < <(find "${MIGRATIONS_DIR}" -name "*.sql" -type f -print0 | sort -z)
    
    if [[ ${#files[@]} -eq 0 ]]; then
        log_warning "No SQL migration files found in ${MIGRATIONS_DIR}"
        return 1
    fi
    
    printf '%s\n' "${files[@]}"
}

# マイグレーション実行
run_migration() {
    local db_name="$1"
    local migration_file="$2"
    
    log_info "Applying migration to ${db_name}: $(basename "${migration_file}")"
    
    if [[ "${DRY_RUN}" == "true" ]]; then
        log_info "[DRY-RUN] Would execute: psql -h ${HOST} -p ${PORT} -U ${USERNAME} -d ${db_name} -f ${migration_file}"
        return 0
    fi
    
    if psql -h "${HOST}" -p "${PORT}" -U "${USERNAME}" -d "${db_name}" -f "${migration_file}"; then
        log_success "Migration applied successfully: $(basename "${migration_file}")"
    else
        log_error "Failed to apply migration: $(basename "${migration_file}")"
        return 1
    fi
}

# メインデータベースマイグレーション
migrate_main_database() {
    log_info "Starting migration for main database: ${DATABASE}"
    
    # 接続テスト
    if ! test_connection "${DATABASE}"; then
        return 1
    fi
    
    # マイグレーションファイルを取得して実行
    local migration_files
    if ! migration_files=($(get_migration_files)); then
        return 1
    fi
    
    local success_count=0
    local total_count=${#migration_files[@]}
    
    for migration_file in "${migration_files[@]}"; do
        # テスト用DB作成スクリプトはメインDBでは実行しない
        if [[ "$(basename "${migration_file}")" == *"create_test_db"* ]]; then
            log_verbose "Skipping test database creation script for main database: $(basename "${migration_file}")"
            continue
        fi
        
        if run_migration "${DATABASE}" "${migration_file}"; then
            ((success_count++))
        else
            log_error "Migration failed. Stopping execution."
            return 1
        fi
    done
    
    log_success "Main database migration completed: ${success_count} migrations applied"
}

# テストデータベースマイグレーション
migrate_test_database() {
    local test_db_name="${DATABASE}_test"
    log_info "Starting migration for test database: ${test_db_name}"
    
    # まずテスト用データベースを作成
    local test_db_creation_file=""
    while IFS= read -r file; do
        if [[ "$(basename "${file}")" == *"create_test_db"* ]]; then
            test_db_creation_file="${file}"
            break
        fi
    done < <(get_migration_files)
    
    if [[ -n "${test_db_creation_file}" ]]; then
        log_info "Creating test database using: $(basename "${test_db_creation_file}")"
        if [[ "${DRY_RUN}" == "false" ]]; then
            if ! run_migration "${DATABASE}" "${test_db_creation_file}"; then
                log_error "Failed to create test database"
                return 1
            fi
        fi
    else
        log_warning "Test database creation script not found"
    fi
    
    # テスト用データベースの接続テスト
    if [[ "${DRY_RUN}" == "false" ]] && ! test_connection "${test_db_name}"; then
        return 1
    fi
    
    # マイグレーションファイルを取得して実行
    local migration_files
    if ! migration_files=($(get_migration_files)); then
        return 1
    fi
    
    local success_count=0
    
    for migration_file in "${migration_files[@]}"; do
        # テスト用DB作成スクリプトはスキップ（既に実行済み）
        if [[ "$(basename "${migration_file}")" == *"create_test_db"* ]]; then
            continue
        fi
        
        if run_migration "${test_db_name}" "${migration_file}"; then
            ((success_count++))
        else
            log_error "Migration failed. Stopping execution."
            return 1
        fi
    done
    
    log_success "Test database migration completed: ${success_count} migrations applied"
}

# メイン関数
main() {
    log_info "Database Migration Script Starting..."
    log_verbose "Arguments: $*"
    
    # 引数解析
    parse_args "$@"
    
    # DATABASE_URLの解析
    parse_database_url
    
    # パスワード設定
    setup_password
    
    # 設定表示
    if [[ "${VERBOSE}" == "true" ]]; then
        log_verbose "Configuration:"
        log_verbose "  Host: ${HOST}"
        log_verbose "  Port: ${PORT}"
        log_verbose "  Username: ${USERNAME}"
        log_verbose "  Database: ${DATABASE}"
        log_verbose "  Migrations Directory: ${MIGRATIONS_DIR}"
        log_verbose "  Migrate Main DB: ${MIGRATE_MAIN_DB}"
        log_verbose "  Migrate Test DB: ${MIGRATE_TEST_DB}"
        log_verbose "  Dry Run: ${DRY_RUN}"
    fi
    
    # Dry runの場合はファイル一覧を表示
    if [[ "${DRY_RUN}" == "true" ]]; then
        log_info "DRY RUN MODE - Migration files found:"
        local migration_files
        if migration_files=($(get_migration_files)); then
            for file in "${migration_files[@]}"; do
                echo "  - $(basename "${file}")"
            done
        fi
        log_info "DRY RUN MODE - No actual migrations will be executed"
    fi
    
    # マイグレーション実行
    local exit_code=0
    
    if [[ "${MIGRATE_MAIN_DB}" == "true" ]]; then
        if ! migrate_main_database; then
            exit_code=1
        fi
    fi
    
    if [[ "${MIGRATE_TEST_DB}" == "true" ]]; then
        if ! migrate_test_database; then
            exit_code=1
        fi
    fi
    
    if [[ "${exit_code}" -eq 0 ]]; then
        log_success "Database migration completed successfully!"
    else
        log_error "Database migration failed!"
    fi
    
    exit "${exit_code}"
}

# スクリプトが直接実行された場合のみmainを呼び出し
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
