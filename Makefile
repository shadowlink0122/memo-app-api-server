# Makefile for memo-app API server

.PHONY: build run test test-coverage test-watch clean help docker-build docker-up docker-down docker-logs

# デフォルトターゲット
all: test build

# アプリケーションをビルド
build:
	go build -o bin/memo-app src/main.go

# アプリケーションを実行
run:
	go run src/main.go

# テストを実行
test:
	go test ./test -v

# 全テストスイートを実行
test-all:
	go test ./test/... -v

# ユニットテストのみ実行
test-unit:
	go test ./test/config ./test/middleware ./test/logger ./test/storage -v

# 統合テストを実行
test-integration:
	go test ./test/integration -v

# データベーステストを実行（要: データベース接続）
test-database:
	@echo "データベーステストを実行します（TEST_DATABASE_URLが必要）"
	TEST_DATABASE_URL="postgres://memo_user:memo_password@localhost:5432/memo_db?sslmode=disable" go test ./test/database -v

# E2Eテストを実行（要: Docker環境）
test-e2e:
	@echo "E2Eテストを実行します（Docker環境が必要）"
	go test ./test/e2e -v

# 短いテストのみ実行
test-short:
	go test ./test/... -short -v

# ベンチマークテストを実行
test-bench:
	go test ./test/... -bench=. -benchmem -v

# テストを監視モードで実行（ファイル変更時に自動実行）
test-watch:
	@echo "ファイル変更を監視してテストを自動実行します（Ctrl+Cで停止）"
	@while true; do \
		go test ./test -v; \
		sleep 2; \
	done

# テストカバレッジを生成
test-coverage:
	go test ./test/... -v -cover -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "カバレッジレポートが coverage.html に生成されました"

# テストカバレッジを表示
test-coverage-func:
	go test ./test/... -v -cover -coverprofile=coverage.out
	go tool cover -func=coverage.out

# 依存関係を整理
tidy:
	go mod tidy

# Docker: イメージをビルド
docker-build:
	docker-compose build

# Docker: 開発環境を起動
docker-up:
	docker-compose up -d

# Docker: 開発環境を起動（フォアグラウンド）
docker-up-fg:
	docker-compose up

# Docker: 開発環境を停止
docker-down:
	docker-compose down

# Docker: ログを表示
docker-logs:
	docker-compose logs -f

# Docker: 本番環境を起動
docker-prod-up:
	docker-compose -f docker-compose.prod.yml up -d

# Docker: 本番環境を停止
docker-prod-down:
	docker-compose -f docker-compose.prod.yml down

# Docker: データベースのみ起動
docker-db:
	docker-compose up -d db

# Docker: Adminer（DB管理ツール）付きで起動
docker-dev:
	docker-compose --profile dev up -d

# Docker: コンテナとボリュームを完全削除
docker-clean:
	docker-compose down -v --remove-orphans
	docker system prune -f

# クリーンアップ
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# ヘルプ
help:
	@echo "利用可能なコマンド:"
	@echo "  build            - アプリケーションをビルド"
	@echo "  run              - アプリケーションを実行"
	@echo ""
	@echo "テストコマンド:"
	@echo "  test             - 基本テストを実行"
	@echo "  test-all         - 全テストスイートを実行"
	@echo "  test-unit        - ユニットテストのみ実行"
	@echo "  test-integration - 統合テストを実行"
	@echo "  test-database    - データベーステストを実行"
	@echo "  test-e2e         - E2Eテストを実行"
	@echo "  test-short       - 短いテストのみ実行"
	@echo "  test-bench       - ベンチマークテストを実行"
	@echo "  test-watch       - テストを監視モードで実行"
	@echo "  test-coverage    - テストカバレッジレポートを生成"
	@echo "  test-coverage-func - テストカバレッジを関数別に表示"
	@echo "  tidy             - 依存関係を整理"
	@echo ""
	@echo "Docker コマンド:"
	@echo "  docker-build     - Dockerイメージをビルド"
	@echo "  docker-up        - 開発環境を起動（バックグラウンド）"
	@echo "  docker-up-fg     - 開発環境を起動（フォアグラウンド）"
	@echo "  docker-down      - 開発環境を停止"
	@echo "  docker-logs      - ログを表示"
	@echo "  docker-prod-up   - 本番環境を起動"
	@echo "  docker-prod-down - 本番環境を停止"
	@echo "  docker-db        - データベースのみ起動"
	@echo "  docker-dev       - Adminer付きで開発環境を起動"
	@echo "  docker-clean     - コンテナとボリュームを完全削除"
	@echo "  clean            - 生成ファイルを削除"
	@echo "  help             - このヘルプを表示"
