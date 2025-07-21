# Makefile for memo-app API server (ローカルビルド + Docker実行)

.PHONY: build build-linux build-darwin build-windows docker-build docker-up docker-down docker-logs docker-test docker-clean help

# デフォルトターゲット（ローカルビルド + Docker環境での起動）
all: build docker-up

# === ローカルビルドコマンド ===

# OS検出用変数
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

# ローカル環境に応じたビルド
build:
ifeq ($(UNAME_S),Linux)
	@echo "Linuxバイナリをビルドします..."
	@$(MAKE) build-linux
else ifeq ($(UNAME_S),Darwin)
	@echo "macOS用バイナリをビルドします..."
	@$(MAKE) build-darwin
else ifeq ($(OS),Windows_NT)
	@echo "Windowsバイナリをビルドします..."
	@$(MAKE) build-windows
else
	@echo "不明なOS: $(UNAME_S), Linux用バイナリをビルドします..."
	@$(MAKE) build-linux
endif

# Linux用クロスコンパイル（本番環境用）
build-linux:
	@echo "Linux/amd64用バイナリをビルド中..."
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o bin/memo-app src/main.go
	@echo "✅ bin/memo-app (Linux/amd64) が生成されました"

# macOS用ビルド
build-darwin:
	@echo "macOS用バイナリをビルド中..."
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-w -s" -o bin/memo-app src/main.go
	@echo "✅ bin/memo-app (macOS/amd64) が生成されました"

# Windows用クロスコンパイル
build-windows:
	@echo "Windows用バイナリをビルド中..."
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-w -s" -o bin/memo-app.exe src/main.go
	@echo "✅ bin/memo-app.exe (Windows/amd64) が生成されました"

# === Dockerコマンド ===

# Docker: イメージをビルド（事前にローカルビルドが必要）
docker-build: build-linux
	@echo "Dockerイメージをビルドします（Linux用バイナリを使用）..."
	docker compose build

# Docker: 開発環境を起動（事前ビルドされたバイナリを使用）
docker-up: build-linux
	@echo "Linux用バイナリでDocker環境を起動します..."
	docker compose up -d

# Docker: 開発環境を起動（フォアグラウンド）
docker-up-fg:
	docker compose up

# Docker: 開発環境を停止
docker-down:
	docker compose down

# Docker: ログを表示
docker-logs:
	docker compose logs -f

# Docker環境でのテスト実行（テスト用ステージを使用）
docker-test:
	@echo "開発用コンテナ（テストステージ）でテストを実行します..."
	docker compose --profile dev up -d app-dev
	docker compose exec app-dev make test-all-internal
	docker compose --profile dev down app-dev

# Docker環境でのテストカバレッジ（テスト用ステージを使用）
docker-test-coverage:
	@echo "開発用コンテナ（テストステージ）でテストカバレッジを生成します..."
	docker compose --profile dev up -d app-dev
	docker compose exec app-dev make test-coverage-internal
	docker compose --profile dev down app-dev

# Docker環境でのビルド（コンテナ内でビルド - 非推奨）
docker-build-app:
	@echo "⚠️  警告: ローカルビルドを推奨します"
	@echo "   代わりに 'make build' を使用してください"
	@echo "開発用コンテナでアプリケーションをビルドします..."
	docker compose --profile dev up -d app-dev
	docker compose exec app-dev make build-internal
	docker compose --profile dev down app-dev

# 個別テスト実行用のヘルパーコマンド
docker-test-unit:
	@echo "開発用コンテナでユニットテストを実行します..."
	docker compose --profile dev up -d app-dev
	docker compose exec app-dev make test-unit-internal
	docker compose --profile dev down app-dev

docker-test-integration:
	@echo "開発用コンテナで統合テストを実行します..."
	docker compose --profile dev up -d app-dev
	docker compose exec app-dev make test-integration-internal
	docker compose --profile dev down app-dev

docker-test-database:
	@echo "開発用コンテナでデータベーステストを実行します..."
	docker compose --profile dev up -d app-dev
	docker compose exec app-dev make test-database-internal
	docker compose --profile dev down app-dev

docker-test-e2e:
	@echo "開発用コンテナでE2Eテストを実行します..."
	docker compose --profile dev up -d app-dev
	docker compose exec app-dev make test-e2e-internal
	docker compose --profile dev down app-dev

# Docker: 本番環境を起動
docker-prod-up:
	docker compose -f docker-compose.prod.yml up -d

# Docker: 本番環境を停止
docker-prod-down:
	docker compose -f docker-compose.prod.yml down

# Docker: データベースのみ起動
docker-db:
	docker compose up -d db

# Docker: MinIOのみ起動
docker-minio:
	docker compose up -d minio

# Docker: Adminer（DB管理ツール）付きで起動
docker-dev:
	docker compose --profile dev up -d

# Docker: コンテナとボリュームを完全削除
docker-clean:
	docker compose down -v --remove-orphans
	docker system prune -f

# === 以下はコンテナ内でのみ実行される内部コマンド ===

# アプリケーションをビルド（コンテナ内でのみ実行）
build-internal:
	@echo "⚠️  コンテナ内ビルドは非推奨です。ローカルビルド（make build）を推奨します"
	go build -o bin/memo-app src/main.go

# テストを実行（コンテナ内でのみ実行）
test-internal:
	go test ./test -v

# 全テストスイートを実行（コンテナ内でのみ実行）
test-all-internal:
	go test ./test/... -v

# ユニットテストのみ実行（コンテナ内でのみ実行）
test-unit-internal:
	go test ./test/config ./test/middleware ./test/logger ./test/storage -v

# 統合テストを実行（コンテナ内でのみ実行）
test-integration-internal:
	go test ./test/integration -v

# データベーステストを実行（コンテナ内でのみ実行）
test-database-internal:
	@echo "データベーステストを実行します"
	go test ./test/database -v

# E2Eテストを実行（コンテナ内でのみ実行）
test-e2e-internal:
	@echo "E2Eテストを実行します"
	go test ./test/e2e -v

# テストカバレッジを生成（コンテナ内でのみ実行）
test-coverage-internal:
	go test ./test/... -v -cover -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "カバレッジレポートが coverage.html に生成されました"

# テストカバレッジを表示（コンテナ内でのみ実行）
test-coverage-func-internal:
	go test ./test/... -v -cover -coverprofile=coverage.out
	go tool cover -func=coverage.out

# 依存関係を整理（コンテナ内でのみ実行）
tidy-internal:
	go mod tidy

# クリーンアップ（コンテナ内でのみ実行）
clean-internal:
	rm -rf bin/
	rm -f coverage.out coverage.html

# === 非推奨コマンド（ローカル実行はローカルビルド + Docker実行を推奨） ===

# ローカル実行は推奨しませんが、開発時のテスト用に提供
run-local:
	@echo "⚠️  注意: ローカル実行は開発テスト用です"
	@echo "   本番環境ではDocker環境を使用してください"
	@if [ ! -f bin/memo-app ]; then echo "バイナリが見つかりません。'make build' を実行してください"; exit 1; fi
	./bin/memo-app

# 非推奨: Docker環境での実行を推奨
run:
	@echo "⚠️  警告: ローカル実行よりもDocker環境を推奨します"
	@echo "   代わりに 'make docker-up' を使用してください"
	@echo "   テスト用のローカル実行は 'make run-local' を使用してください"
	@exit 1

test:
	@echo "⚠️  警告: このアプリケーションはDocker専用です"
	@echo "   代わりに 'make docker-test' を使用してください"
	@exit 1

test-coverage:
	@echo "⚠️  警告: このアプリケーションはDocker専用です"
	@echo "   代わりに 'make docker-test-coverage' を使用してください"
	@exit 1

# ヘルプ
help:
	@echo "=========================================="
	@echo "Memo App API Server - ローカルビルド + Docker実行"
	@echo "=========================================="
	@echo ""
	@echo "🔧 ローカルビルド:"
	@echo "  build            - OS検出して適切なバイナリをビルド"
	@echo "  build-linux      - Linux/amd64用バイナリをビルド（本番用）"
	@echo "  build-darwin     - macOS用バイナリをビルド"
	@echo "  build-windows    - Windows用バイナリをビルド"
	@echo ""
	@echo "🐳 Docker環境管理:"
	@echo "  docker-up        - ローカルビルド後、開発環境を起動"
	@echo "  docker-up-fg     - 開発環境を起動（フォアグラウンド）"
	@echo "  docker-down      - 開発環境を停止"
	@echo "  docker-logs      - ログを表示"
	@echo "  docker-build     - ローカルビルド後、Dockerイメージをビルド"
	@echo "  docker-clean     - コンテナとボリュームを完全削除"
	@echo ""
	@echo "🧪 テスト:"
	@echo "  docker-test      - テストを実行（testステージのコンテナ使用）"
	@echo "  docker-test-coverage - テストカバレッジを生成"
	@echo ""
	@echo "📦 個別サービス:"
	@echo "  docker-db        - データベースのみ起動"
	@echo "  docker-minio     - MinIOのみ起動"
	@echo "  docker-dev       - Adminer付きで開発環境を起動"
	@echo ""
	@echo "🚀 本番環境:"
	@echo "  docker-prod-up   - 本番環境を起動（appコンテナのみ）"
	@echo "  docker-prod-down - 本番環境を停止"
	@echo ""
	@echo "📚 使用例:"
	@echo "  make build && make docker-up         # ビルド後に開発環境を起動"
	@echo "  make docker-test                     # テスト実行"
	@echo "  make build-linux && make docker-prod-up  # 本番環境用"
	@echo "  make docker-down                     # 開発環境停止"
	@echo ""
	@echo "💡 推奨ワークフロー:"
	@echo "  1. make build          - ローカルでクロスコンパイル"
	@echo "  2. make docker-up      - Docker環境で実行"
	@echo "  3. make docker-test    - テスト実行"
