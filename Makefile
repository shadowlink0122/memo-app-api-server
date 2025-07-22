# Makefile for memo-app API server (ローカルビルド + Docker実行)

.PHONY: build build-linux build-darwin build-windows docker-build docker-up docker-down docker-logs docker-test docker-clean help fmt fmt-check fmt-imports lint lint-ci migrate migrate-test migrate-all migrate-dry-run docker-migrate docker-migrate-test docker-migrate-all security security-ci docker-security init pr-create pr-ready pr-check pr-merge pr-merge-commit pr-status pr-wip pr-unwip pr-info swagger-serve swagger-validate swagger-docs docker-test-validation docker-test-security test-validation-internal test-security-internal git-setup-hooks git-remove-hooks git-hooks-status

# デフォルトターゲット（ローカルビルド + Docker環境での起動）
all: build docker-up

# === 開発環境初期化 ===

# 開発環境の初期化（必要なツールのインストールとセットアップ）
init:
	@echo "🚀 開発環境を初期化しています..."
	@echo ""
	@echo "📦 必要なツールをインストール中..."
	
	# Go依存関係のダウンロード
	@echo "  • Go依存関係をダウンロード中..."
	@go mod download
	
	# 開発ツールのインストール
	@echo "  • 開発ツールをインストール中..."
	@echo "    - goimports（インポート整理）"
	@go install golang.org/x/tools/cmd/goimports@latest || echo "    ⚠️  goimportsのインストールに失敗（手動でインストールしてください）"
	
	@echo "    - golangci-lint（リンター）"
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		if command -v brew >/dev/null 2>&1; then \
			echo "      Homebrewでインストール中..."; \
			brew install golangci-lint || echo "      ⚠️  golangci-lintのインストールに失敗"; \
		else \
			echo "      手動インストール中..."; \
			curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.55.2 || echo "      ⚠️  golangci-lintのインストールに失敗"; \
		fi; \
	else \
		echo "      ✅ 既にインストール済み"; \
	fi
	
	@echo "    - gosec（セキュリティスキャナー）"
	@go install github.com/securego/gosec/v2/cmd/gosec@v2.21.4 || echo "    ⚠️  gosecのインストールに失敗"
	
	# Docker環境の確認
	@echo ""
	@echo "🐳 Docker環境を確認中..."
	@if command -v docker >/dev/null 2>&1; then \
		echo "  ✅ Docker がインストールされています"; \
		if docker info >/dev/null 2>&1; then \
			echo "  ✅ Docker デーモンが実行中です"; \
		else \
			echo "  ⚠️  Docker デーモンが実行されていません。Dockerを起動してください"; \
		fi; \
	else \
		echo "  ❌ Docker がインストールされていません"; \
		echo "     以下からインストールしてください: https://docs.docker.com/get-docker/"; \
	fi
	
	@if command -v docker-compose >/dev/null 2>&1 || docker compose version >/dev/null 2>&1; then \
		echo "  ✅ Docker Compose が利用可能です"; \
	else \
		echo "  ❌ Docker Compose が利用できません"; \
	fi
	
	# 必要なディレクトリの作成
	@echo ""
	@echo "📁 必要なディレクトリを作成中..."
	@mkdir -p bin logs
	@echo "  ✅ bin/, logs/ ディレクトリを作成しました"
	
	# 環境設定ファイルの確認
	@echo ""
	@echo "⚙️  環境設定を確認中..."
	@if [ ! -f .env ]; then \
		if [ -f .env.example ]; then \
			echo "  📝 .env ファイルを .env.example から作成中..."; \
			cp .env.example .env; \
			echo "  ✅ .env ファイルを作成しました"; \
			echo "     必要に応じて設定を編集してください"; \
		else \
			echo "  📝 .env ファイルが見つかりません"; \
			echo "     必要に応じて .env.example をコピーして設定してください"; \
		fi; \
	else \
		echo "  ✅ .env ファイルが存在します"; \
	fi
	
	# Gitフックの設定（オプション）
	@echo ""
	@echo "🔗 Git設定を確認中..."
	@if [ -d .git ]; then \
		echo "  ✅ Gitリポジトリが初期化されています"; \
		if [ ! -f .git/hooks/pre-commit ]; then \
			echo "  📝 pre-commitフックを設定中..."; \
			echo '#!/bin/sh' > .git/hooks/pre-commit; \
			echo 'echo "Running pre-commit checks..."' >> .git/hooks/pre-commit; \
			echo 'make fmt-check || exit 1' >> .git/hooks/pre-commit; \
			echo 'make lint-ci || exit 1' >> .git/hooks/pre-commit; \
			echo 'echo "✅ Pre-commit checks passed!"' >> .git/hooks/pre-commit; \
			chmod +x .git/hooks/pre-commit; \
			echo "  ✅ pre-commitフックを設定しました（フォーマット・リントチェック）"; \
		else \
			echo "  ✅ pre-commitフックが既に設定されています"; \
		fi; \
		if [ ! -f .git/hooks/pre-push ]; then \
			echo "  📝 pre-pushフックを設定中..."; \
			echo '#!/bin/sh' > .git/hooks/pre-push; \
			echo 'echo "🔍 Running pre-push format check..."' >> .git/hooks/pre-push; \
			echo 'echo "📝 Checking code format before push..."' >> .git/hooks/pre-push; \
			echo '' >> .git/hooks/pre-push; \
			echo '# フォーマットを実行' >> .git/hooks/pre-push; \
			echo 'make fmt' >> .git/hooks/pre-push; \
			echo '' >> .git/hooks/pre-push; \
			echo '# 変更があるかチェック' >> .git/hooks/pre-push; \
			echo 'if ! git diff --exit-code --quiet; then' >> .git/hooks/pre-push; \
			echo '    echo "❌ フォーマットにより変更が発生しました。以下のファイルに差分があります:"' >> .git/hooks/pre-push; \
			echo '    git diff --name-only' >> .git/hooks/pre-push; \
			echo '    echo ""' >> .git/hooks/pre-push; \
			echo '    echo "🔧 以下の手順で修正してください:"' >> .git/hooks/pre-push; \
			echo '    echo "  1. git add -A"' >> .git/hooks/pre-push; \
			echo '    echo "  2. git commit -m \"Format code\""' >> .git/hooks/pre-push; \
			echo '    echo "  3. git push"' >> .git/hooks/pre-push; \
			echo '    echo ""' >> .git/hooks/pre-push; \
			echo '    exit 1' >> .git/hooks/pre-push; \
			echo 'fi' >> .git/hooks/pre-push; \
			echo '' >> .git/hooks/pre-push; \
			echo 'echo "✅ Format check passed - no changes needed!"' >> .git/hooks/pre-push; \
			chmod +x .git/hooks/pre-push; \
			echo "  ✅ pre-pushフックを設定しました（フォーマット・差分チェック）"; \
		else \
			echo "  ✅ pre-pushフックが既に設定されています"; \
		fi; \
	else \
		echo "  ⚠️  Gitリポジトリが見つかりません"; \
	fi
	
	# 初期ビルドの実行
	@echo ""
	@echo "🔨 初期ビルドを実行中..."
	@$(MAKE) build
	
	@echo ""
	@echo "🎉 開発環境の初期化が完了しました！"
	@echo ""
	@echo "📚 次のステップ:"
	@echo "  1. 環境設定ファイルを確認: .env"
	@echo "  2. Docker環境を起動: make docker-up"
	@echo "  3. データベースマイグレーション: make docker-migrate-all"
	@echo "  4. テストを実行: make docker-test"
	@echo "  5. セキュリティスキャン: make security"
	@echo ""
	@echo "🔗 便利なコマンド:"
	@echo "  make help           - 利用可能なコマンド一覧"
	@echo "  make docker-logs    - アプリケーションログを表示"
	@echo "  make docker-clean   - Docker環境をクリーンアップ"
	@echo ""

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

# Docker環境でバリデーションテストを実行
docker-test-validation:
	@echo "開発用コンテナでバリデーションテストを実行します..."
	docker compose --profile dev up -d app-dev
	docker compose exec app-dev make test-validation-internal
	docker compose --profile dev down app-dev

# Docker環境でセキュリティテストを実行
docker-test-security:
	@echo "開発用コンテナでセキュリティテストを実行します..."
	docker compose --profile dev up -d app-dev
	docker compose exec app-dev make test-security-internal
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

# バリデーションテストを実行（コンテナ内でのみ実行）
test-validation-internal:
	@echo "バリデーションテストを実行します"
	go test ./test/validator -v

# セキュリティテストを実行（コンテナ内でのみ実行）
test-security-internal:
	@echo "セキュリティテストを実行します"
	go test ./test/security -v

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

# === コードフォーマット・品質管理 ===

# コードフォーマット
fmt:
	@echo "🎨 Goコードをフォーマットしています..."
	go fmt ./...
	@echo "✅ フォーマット完了"

# フォーマットのチェック（CI用）
fmt-check:
	@echo "🔍 コードフォーマットをチェックしています..."
	@result=$$(go fmt ./...); \
	if [ -n "$$result" ]; then \
		echo "❌ 以下のファイルがフォーマットされていません:"; \
		echo "$$result"; \
		exit 1; \
	else \
		echo "✅ すべてのファイルが正しくフォーマットされています"; \
	fi

# インポート文を含むフォーマット（goimportsが必要）
fmt-imports:
	@echo "📦 インポート文を含むフォーマットを実行しています..."
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
		echo "✅ goimportsでフォーマットを実行しました"; \
	else \
		echo "⚠️  goimportsがインストールされていません。インストールするには:"; \
		echo "   go install golang.org/x/tools/cmd/goimports@latest"; \
		echo ""; \
		echo "📝 代わりにgo fmtを実行します:"; \
		go fmt ./...; \
		echo "✅ go fmtでフォーマット完了"; \
	fi

# リント（golangci-lintが必要）
lint:
	@echo "🔍 コードをリントしています..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
		echo "✅ リント完了"; \
	else \
		echo "⚠️  golangci-lintがインストールされていません。インストールするには:"; \
		echo "   # Homebrewの場合"; \
		echo "   brew install golangci-lint"; \
		echo ""; \
		echo "   # 手動インストールの場合"; \
		echo "   curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$$(go env GOPATH)/bin v1.55.2"; \
		echo ""; \
		echo "📝 代わりにgo vetを実行します:"; \
		go vet ./...; \
		echo "✅ go vet完了"; \
	fi

# CI用の軽量リント（go標準ツールのみ使用）
lint-ci:
	@echo "🔍 CI用リントを実行しています..."
	@echo "📝 go fmt をチェック中..."
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "❌ フォーマットされていないファイルがあります:"; \
		gofmt -l .; \
		echo "以下のコマンドで修正してください: make fmt"; \
		exit 1; \
	else \
		echo "✅ go fmt チェック完了"; \
	fi
	@echo "📝 go vet を実行中..."
	@go vet ./...
	@echo "✅ go vet 完了"
	@echo "📝 go mod tidy をチェック中..."
	@go mod tidy
	@if ! git diff --exit-code go.mod go.sum; then \
		echo "❌ go.mod または go.sum が最新ではありません"; \
		echo "以下のコマンドで修正してください: go mod tidy"; \
		exit 1; \
	else \
		echo "✅ go mod tidy チェック完了"; \
	fi
	@echo "🎉 CI用リント完了"

# セキュリティスキャン（gosecが必要）
security:
	@echo "🔒 セキュリティスキャンを実行しています..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
		echo "✅ セキュリティスキャン完了"; \
	else \
		echo "⚠️  gosecがインストールされていません。インストールするには:"; \
		echo "   # Homebrewの場合"; \
		echo "   brew install gosec"; \
		echo ""; \
		echo "   # 手動インストールの場合"; \
		echo "   curl -sfL https://raw.githubusercontent.com/securecodewarrior/gosec/master/install.sh | sh -s -- -b \$$(go env GOPATH)/bin v2.21.4"; \
		echo ""; \
		echo "   # go installの場合"; \
		echo "   go install github.com/securego/gosec/v2/cmd/gosec@latest"; \
		exit 1; \
	fi

# CI用のセキュリティスキャン（gosecを自動インストール）
security-ci:
	@echo "🔒 CI用セキュリティスキャンを実行しています..."
	@echo "📦 gosecをインストール中..."
	@go install github.com/securego/gosec/v2/cmd/gosec@v2.21.4
	@echo "🔍 セキュリティスキャンを実行中..."
	@set +e; \
	$$(go env GOPATH)/bin/gosec ./...; \
	exit_code=$$?; \
	if [ $$exit_code -ne 0 ]; then \
		echo ""; \
		echo "⚠️  セキュリティスキャンで問題が見つかりましたが、CIは継続します"; \
		echo "📋 上記の問題を確認して修正することを推奨します"; \
	fi; \
	echo "✅ セキュリティスキャン完了"

# Docker環境でのセキュリティスキャン実行
docker-security:
	@echo "🐳 Docker環境でセキュリティスキャンを実行します..."
	docker compose --profile dev up -d app-dev
	docker compose exec app-dev make security-ci
	docker compose --profile dev down app-dev

# === データベースマイグレーション ===

# データベースマイグレーション（メインDB）
migrate:
	@echo "メインデータベースにマイグレーションを実行します..."
	./scripts/migrate-database.sh --main-db --verbose

# データベースマイグレーション（テストDB）
migrate-test:
	@echo "テスト用データベースにマイグレーションを実行します..."
	./scripts/migrate-database.sh --test-db --verbose

# データベースマイグレーション（両方）
migrate-all:
	@echo "メイン・テスト両方のデータベースにマイグレーションを実行します..."
	./scripts/migrate-database.sh --both --verbose

# マイグレーションのドライラン（実行せずにファイル一覧表示）
migrate-dry-run:
	@echo "マイグレーションファイルの確認（ドライラン）..."
	./scripts/migrate-database.sh --dry-run --verbose

# Docker環境でのマイグレーション（メインDB）
docker-migrate:
	@echo "Docker環境のメインデータベースにマイグレーションを実行します..."
	PGPASSWORD=memo_password ./scripts/migrate-database.sh \
		--host localhost \
		--port 5432 \
		--username memo_user \
		--database memo_db \
		--main-db \
		--verbose

# Docker環境でのマイグレーション（テストDB）
docker-migrate-test:
	@echo "Docker環境のテスト用データベースにマイグレーションを実行します..."
	PGPASSWORD=memo_password ./scripts/migrate-database.sh \
		--host localhost \
		--port 5432 \
		--username memo_user \
		--database memo_db \
		--test-db \
		--verbose

# Docker環境でのマイグレーション（両方）
docker-migrate-all:
	@echo "Docker環境の両方のデータベースにマイグレーションを実行します..."
	PGPASSWORD=memo_password ./scripts/migrate-database.sh \
		--host localhost \
		--port 5432 \
		--username memo_user \
		--database memo_db \
		--both \
		--verbose

# === Git / PR管理 ===

# Git pre-pushフックを手動で設定
git-setup-hooks:
	@echo "🔗 Gitフックを設定しています..."
	@if [ ! -d .git ]; then \
		echo "❌ Gitリポジトリが見つかりません"; \
		exit 1; \
	fi
	@echo "📝 pre-pushフックを設定中..."
	@echo '#!/bin/sh' > .git/hooks/pre-push
	@echo 'echo "🔍 Running pre-push format check..."' >> .git/hooks/pre-push
	@echo 'echo "📝 Checking code format before push..."' >> .git/hooks/pre-push
	@echo '' >> .git/hooks/pre-push
	@echo '# フォーマットを実行' >> .git/hooks/pre-push
	@echo 'make fmt' >> .git/hooks/pre-push
	@echo '' >> .git/hooks/pre-push
	@echo '# 変更があるかチェック' >> .git/hooks/pre-push
	@echo 'if ! git diff --exit-code --quiet; then' >> .git/hooks/pre-push
	@echo '    echo "❌ フォーマットにより変更が発生しました。以下のファイルに差分があります:"' >> .git/hooks/pre-push
	@echo '    git diff --name-only' >> .git/hooks/pre-push
	@echo '    echo ""' >> .git/hooks/pre-push
	@echo '    echo "🔧 以下の手順で修正してください:"' >> .git/hooks/pre-push
	@echo '    echo "  1. git add -A"' >> .git/hooks/pre-push
	@echo '    echo "  2. git commit -m \"Format code\""' >> .git/hooks/pre-push
	@echo '    echo "  3. git push"' >> .git/hooks/pre-push
	@echo '    echo ""' >> .git/hooks/pre-push
	@echo '    exit 1' >> .git/hooks/pre-push
	@echo 'fi' >> .git/hooks/pre-push
	@echo '' >> .git/hooks/pre-push
	@echo 'echo "✅ Format check passed - no changes needed!"' >> .git/hooks/pre-push
	@chmod +x .git/hooks/pre-push
	@echo "✅ pre-pushフックを設定しました"
	@echo ""
	@echo "📋 設定内容:"
	@echo "  - git push前に自動でmake fmtを実行"
	@echo "  - フォーマットにより差分が発生した場合はpushを中止"
	@echo "  - 差分がある場合は修正手順を表示"

# Gitフックを削除
git-remove-hooks:
	@echo "🗑️  Gitフックを削除しています..."
	@if [ -f .git/hooks/pre-push ]; then \
		rm .git/hooks/pre-push; \
		echo "✅ pre-pushフックを削除しました"; \
	else \
		echo "ℹ️  pre-pushフックは存在しません"; \
	fi
	@if [ -f .git/hooks/pre-commit ]; then \
		rm .git/hooks/pre-commit; \
		echo "✅ pre-commitフックを削除しました"; \
	else \
		echo "ℹ️  pre-commitフックは存在しません"; \
	fi

# 現在のGitフック状態を確認
git-hooks-status:
	@echo "📊 現在のGitフック状態:"
	@echo ""
	@if [ -f .git/hooks/pre-commit ]; then \
		echo "✅ pre-commitフック: 設定済み"; \
		echo "   内容: フォーマット・リントチェック"; \
	else \
		echo "❌ pre-commitフック: 未設定"; \
	fi
	@if [ -f .git/hooks/pre-push ]; then \
		echo "✅ pre-pushフック: 設定済み"; \
		echo "   内容: フォーマット・差分チェック"; \
	else \
		echo "❌ pre-pushフック: 未設定"; \
	fi
	@echo ""
	@echo "🔧 管理コマンド:"
	@echo "  make git-setup-hooks  - フックを設定"
	@echo "  make git-remove-hooks - フックを削除"

# PR作成（現在のブランチから）
pr-create:
	@echo "📝 Pull Requestを作成しています..."
	@if ! command -v gh >/dev/null 2>&1; then \
		echo "❌ GitHub CLIがインストールされていません"; \
		echo "   インストール方法: https://cli.github.com/"; \
		exit 1; \
	fi
	@current_branch=$$(git rev-parse --abbrev-ref HEAD); \
	if [ "$$current_branch" = "main" ] || [ "$$current_branch" = "master" ]; then \
		echo "❌ mainブランチからはPRを作成できません"; \
		exit 1; \
	fi
	@gh pr create --fill --draft
	@echo "✅ Draft PRが作成されました"
	@echo "   Ready for reviewにするには: make pr-ready"

# PR を Ready for review にする
pr-ready:
	@echo "🚀 PRをReady for reviewにしています..."
	@gh pr ready
	@echo "✅ PRがReady for reviewになりました"

# 現在のPRの状態とマージ可能性をチェック
pr-check:
	@echo "🔍 PRの状態をチェックしています..."
	@pr_number=$$(gh pr view --json number -q '.number' 2>/dev/null || echo ""); \
	if [ -z "$$pr_number" ]; then \
		echo "❌ 現在のブランチにPRが見つかりません"; \
		exit 1; \
	fi; \
	echo "📊 PR #$$pr_number の詳細:"; \
	gh pr view $$pr_number; \
	echo ""; \
	echo "🔧 チェック状況:"; \
	gh pr checks $$pr_number

# PRをマージ（Squash merge）
pr-merge:
	@echo "🔀 PRをマージしています..."
	@echo "⚠️  この操作は元に戻すことができません。続行しますか？ [y/N]"
	@read -r confirm; \
	if [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ]; then \
		gh pr merge --squash --delete-branch; \
		echo "✅ PRが正常にマージされ、ブランチが削除されました"; \
	else \
		echo "❌ マージがキャンセルされました"; \
	fi

# PRをマージ（Merge commit）
pr-merge-commit:
	@echo "� PRをMerge commitでマージしています..."
	@echo "⚠️  この操作は元に戻すことができません。続行しますか？ [y/N]"
	@read -r confirm; \
	if [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ]; then \
		gh pr merge --merge --delete-branch; \
		echo "✅ PRが正常にマージされ、ブランチが削除されました"; \
	else \
		echo "❌ マージがキャンセルされました"; \
	fi

# 現在のPRステータスを確認
pr-status:
	@echo "📊 現在のPRステータス:"
	@gh pr status

# PR を WIP としてマーク（マージを防ぐ）
pr-wip:
	@echo "🚧 PRをWIPとしてマークしています..."
	@current_title=$$(gh pr view --json title -q '.title'); \
	if ! echo "$$current_title" | grep -q "^\[WIP\]"; then \
		gh pr edit --title "[WIP] $$current_title"; \
		echo "✅ PRタイトルに[WIP]を追加しました"; \
	else \
		echo "✅ 既に[WIP]マークが付いています"; \
	fi

# PR から WIP マークを削除
pr-unwip:
	@echo "✅ PRからWIPマークを削除しています..."
	@current_title=$$(gh pr view --json title -q '.title'); \
	new_title=$$(echo "$$current_title" | sed 's/^\[WIP\] *//'); \
	gh pr edit --title "$$new_title"
	@echo "✅ WIPマークを削除しました"

# PRの詳細情報を表示
pr-info:
	@echo "📋 PR詳細情報:"
	@gh pr view --json title,number,state,isDraft,mergeable,reviewDecision,statusCheckRollup,headRefName,baseRefName | jq '.'

# === Swagger/OpenAPI管理 ===

# Swagger UIでAPIドキュメントを表示
swagger-serve:
	@echo "🌐 Swagger UIでAPIドキュメントを表示します..."
	@echo "   http://localhost:7000/docs でアクセスできます"
	@echo "   終了するには Ctrl+C を押してください"
	@docker run --rm -p 7000:8080 \
		-v $$(pwd)/api:/app \
		-e SWAGGER_JSON=/app/swagger.yaml \
		swaggerapi/swagger-ui

# Swaggerファイルをバリデーション
swagger-validate:
	@echo "🔍 Swagger YAML ファイルをバリデーションしています..."
	@if command -v docker >/dev/null 2>&1; then \
		docker run --rm -v $$(pwd)/api:/app \
			openapitools/openapi-generator-cli:latest \
			validate -i /app/swagger.yaml; \
		echo "✅ Swagger YAML ファイルは有効です"; \
	else \
		echo "❌ Dockerが必要です"; \
		exit 1; \
	fi

# Swagger仕様書とドキュメントの管理
swagger-docs:
	@echo "� Swagger/OpenAPI ドキュメント管理"
	@echo ""
	@echo "利用可能なコマンド:"
	@echo "  make swagger-serve     - Swagger UIでドキュメント表示"
	@echo "  make swagger-validate  - API仕様の妥当性チェック"
	@echo ""
	@echo "� ファイル構成:"
	@echo "  api/swagger.yaml       - API仕様書（OpenAPI 3.0.3形式）"
	@echo ""
	@echo "🌐 アクセス先:"
	@echo "  http://localhost:7000/docs  - Swagger UI（swagger-serve実行時）"

# ヘルプ
help:
	@echo "=========================================="
	@echo "Memo App API Server - ローカルビルド + Docker実行"
	@echo "=========================================="
	@echo ""
	@echo "🚀 初期化:"
	@echo "  init             - 開発環境を初期化（必要なツールのインストール）"
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
	@echo "🎨 コード品質:"
	@echo "  fmt              - Goコードをフォーマット"
	@echo "  fmt-check        - フォーマットチェック（CI用）"
	@echo "  fmt-imports      - インポート文を含むフォーマット（goimports使用）"
	@echo "  lint             - リント実行（golangci-lint使用、なければgo vet）"
	@echo "  lint-ci          - CI用リント（go標準ツールのみ）"
	@echo "  security         - セキュリティスキャン（gosec使用）"
	@echo "  security-ci      - CI用セキュリティスキャン（gosec自動インストール）"
	@echo "  docker-fmt       - Docker環境でフォーマット実行"
	@echo "  docker-security  - Docker環境でセキュリティスキャン実行"
	@echo ""
	@echo "🗃️  データベースマイグレーション:"
	@echo "  migrate          - メインデータベースにマイグレーション実行"
	@echo "  migrate-test     - テスト用データベースにマイグレーション実行"
	@echo "  migrate-all      - 両方のデータベースにマイグレーション実行"
	@echo "  migrate-dry-run  - マイグレーションファイル確認（実行なし）"
	@echo "  docker-migrate   - Docker環境でメインDBマイグレーション"
	@echo "  docker-migrate-test   - Docker環境でテストDBマイグレーション"
	@echo "  docker-migrate-all    - Docker環境で両方のDBマイグレーション"
	@echo ""
	@echo "🔀 Git管理:"
	@echo "  git-setup-hooks  - pre-push/pre-commitフックを設定"
	@echo "  git-remove-hooks - Gitフックを削除"
	@echo "  git-hooks-status - 現在のGitフック状態を確認"
	@echo ""
	@echo "🔀 PR管理（GitHub CLI必要）:"
	@echo "  pr-create        - 現在のブランチからPRを作成（Draft）"
	@echo "  pr-ready         - PRをReady for reviewにする"
	@echo "  pr-check         - PRの状態とマージ可能性をチェック"
	@echo "  pr-merge         - PRをマージ（Squash merge）"
	@echo "  pr-merge-commit  - PRをマージ（Merge commit）"
	@echo "  pr-status        - 現在のPRステータスを確認"
	@echo "  pr-wip           - PRを[WIP]としてマーク（マージ防止）"
	@echo "  pr-unwip         - PRから[WIP]マークを削除"
	@echo "  pr-info          - PRの詳細情報を表示"
	@echo ""
	@echo "📚 Swagger/OpenAPI:"
	@echo "  swagger-serve    - Swagger UIでAPIドキュメントを表示"
	@echo "  swagger-validate - Swagger YAMLファイルをバリデーション"
	@echo "  swagger-docs     - Swagger関連ヘルプと情報表示"
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
	@echo "  0. make init           - 初回セットアップ（初回のみ）"
	@echo "  1. make build          - ローカルでクロスコンパイル"
	@echo "  2. make docker-up      - Docker環境で実行"
	@echo "  3. make docker-test    - テスト実行"
	@echo "  4. make security       - セキュリティスキャン"
