# Memo App API Server

Go + Ginを使用したREST APIサーバーです。

自己管理をしやすくするためのメモアプリを想定しています。

マイクロサービス化したい & 静的型付け言語が好き、という理由からGoを採用しています。

### 実現したいこと

- githubアカウントで登録可能
- private機能(メイン)
    - わからないもの、困ったことを簡単にメモできる
        - 後から追加コメント、解決できたかわかるようにする
        - 粒度は 一言以上、issue 未満
- public機能
    - 技術ブログ
        - コードや差分がわかりやすい

## プロジェクト構造

```
memo-app-api-server/
├── src/
│   ├── main.go                    # メインアプリケーション
│   ├── config/
│   │   └── config.go             # 設定管理
│   ├── logger/
│   │   └── logger.go             # 構造化ログシステム
│   ├── middleware/
│   │   ├── auth.go               # 認証ミドルウェア
│   │   ├── cors.go               # CORS設定
│   │   ├── logger.go             # ログミドルウェア
│   │   └── rate_limit.go         # レート制限
│   └── storage/
│       └── s3_uploader.go        # S3アップロード機能
├── test/
│   └── api_test.go               # APIテストコード
├── logs/                         # ログファイル出力先
├── scripts/
│   └── init-minio.sh            # MinIO初期化スクリプト
├── docker/
│   └── init.sql                 # DB初期化SQL
├── go.mod                       # Go モジュール定義
├── go.sum                       # 依存関係のハッシュ
├── Makefile                     # ビルド・テスト用コマンド
├── docker-compose.yml           # Docker構成（DB + MinIO）
└── README.md                    # このファイル
```

## 主な機能

### API エンドポイント

#### パブリック（認証不要）
- `GET /` - Hello World（JSON形式）
- `GET /health` - ヘルスチェック
- `GET /hello` - Hello World（テキスト形式）

#### プライベート（認証必要）
- `GET /api/protected` - 認証が必要なエンドポイント

### ミドルウェア

- **LoggerMiddleware** - 構造化ログによるリクエストログ
- **CORSMiddleware** - CORS設定
- **AuthMiddleware** - ユーザー認証（現在は空実装）
- **RateLimitMiddleware** - レート制限（現在は空実装）

### ログ機能

- **構造化ログ**: JSON形式でのログ出力
- **ファイル出力**: ローカルディスクへのログファイル保存
- **自動ローテーション**: タイムスタンプ付きファイル名
- **S3アップロード**: 一定期間後に古いログをS3互換ストレージに自動アップロード
- **MinIO統合**: 開発環境用S3互換ストレージ

## ログ機能の詳細

### ログ出力形式

構造化ログ（JSON形式）で以下の情報を記録：

```json
{
  "level": "info",
  "msg": "リクエスト完了 - 成功",
  "method": "GET",
  "uri": "/",
  "client_ip": "127.0.0.1",
  "status_code": 200,
  "latency_ms": 0,
  "time": "2024-07-21T10:30:00+09:00"
}
```

### ログファイル管理

- **出力先**: `logs/` ディレクトリ
- **ファイル名**: `app_YYYY-MM-DD_HH-mm-ss.log`
- **自動アップロード**: 設定可能な間隔でS3にアップロード
- **ローカル削除**: アップロード後に古いファイルを自動削除

### S3アップロード設定

環境変数で制御：

```bash
LOG_UPLOAD_ENABLED=true      # アップロード機能の有効/無効
LOG_UPLOAD_MAX_AGE=24h       # この期間を過ぎたファイルをアップロード
LOG_UPLOAD_INTERVAL=1h       # アップロードチェックの間隔
```

## 本番環境での使用

### AWS S3使用時の設定例

```bash
# .env または環境変数で設定
S3_ENDPOINT=                    # 空にするとAWS S3を使用
S3_ACCESS_KEY_ID=your-access-key
S3_SECRET_ACCESS_KEY=your-secret-key
S3_REGION=ap-northeast-1
S3_BUCKET=your-production-logs-bucket
S3_USE_SSL=true
```

### セキュリティ考慮事項

1. **認証情報**: AWS認証情報は環境変数で管理
2. **CORS設定**: 本番環境では適切なオリジンを設定
3. **ログレベル**: 本番環境では `warn` または `error` レベルを推奨
4. **バケットポリシー**: S3バケットは適切なアクセス制御を設定

## セットアップ

### 前提条件

- Go 1.21以上
- Docker & Docker Compose（MinIO使用時）

### 環境変数設定

環境変数ファイルをコピーして設定：

```bash
cp .env.example .env
```

主要な設定項目：

```bash
# サーバー設定
SERVER_PORT=8080

# ログ設定
LOG_LEVEL=info
LOG_DIRECTORY=logs
LOG_UPLOAD_ENABLED=true
LOG_UPLOAD_MAX_AGE=24h
LOG_UPLOAD_INTERVAL=1h

# S3/MinIO設定
S3_ENDPOINT=http://localhost:9000
S3_ACCESS_KEY_ID=minioadmin
S3_SECRET_ACCESS_KEY=minioadmin
S3_BUCKET=memo-app-logs
```

### 開発環境での起動

#### 1. MinIO（S3互換ストレージ）を起動

```bash
# Docker Composeでサービスを起動
docker-compose up -d minio

# MinIOバケットを初期化
./scripts/init-minio.sh
```

MinIO管理画面: http://localhost:9001
- ユーザー名: `minioadmin`
- パスワード: `minioadmin`

#### 2. アプリケーションの起動

```bash
# 依存関係のインストール
go mod tidy

# アプリケーション実行
make run

# または直接実行
go run src/main.go
```

サーバーは `http://localhost:8080` で起動します。

### Docker環境での起動

```bash
# 全サービスを起動（PostgreSQL + MinIO + アプリ）
docker-compose up -d

# ログを確認
docker-compose logs -f app
```

## テスト

### テスト実行

```bash
# 全テストを実行
make test

# 個別のテスト種別を実行
make test-unit          # ユニットテスト
make test-integration   # 統合テスト
make test-database      # データベーステスト（PostgreSQL必要）
make test-e2e          # E2Eテスト（Docker環境必要）

# テストカバレッジ付きで実行
make test-coverage

# テストカバレッジを関数別に表示
make test-coverage-func

# テストを監視モードで実行（開発時に便利）
make test-watch

# テストスクリプトを使用（全テスト順次実行）
./scripts/run-tests.sh
```

### テストファイル構成

```
test/
├── api_test.go                   # 基本APIテスト
├── middleware/
│   └── middleware_test.go        # ミドルウェアテスト
├── config/
│   └── config_test.go           # 設定テスト
├── logger/
│   └── logger_test.go           # ログシステムテスト
├── storage/
│   └── storage_test.go          # S3アップロードテスト
├── database/
│   └── database_test.go         # データベーステスト
├── integration/
│   └── integration_test.go      # 統合テスト
└── e2e/
    └── e2e_test.go             # E2Eテスト
```

### テスト環境の前提条件

- **ユニットテスト**: 依存関係なし、単独実行可能
- **統合テスト**: アプリケーションレベルのHTTPテスト、依存関係なし
- **データベーステスト**: PostgreSQLが必要（`docker-compose up -d postgres`）
- **E2Eテスト**: 完全なDocker環境が必要（`docker-compose up -d`）

## ビルド

```bash
# バイナリファイルを生成
make build

# 生成されたバイナリを実行
./bin/memo-app
```

## 開発

### 利用可能なMakeコマンド

```bash
make help              # 利用可能なコマンドを表示
make build             # アプリケーションをビルド
make run               # アプリケーションを実行
make test              # テストを実行
make test-watch        # テストを監視モードで実行
make test-coverage     # テストカバレッジレポートを生成
make test-coverage-func # テストカバレッジを関数別に表示
make tidy              # 依存関係を整理
make clean             # 生成ファイルを削除
```

## 使用技術

- **Go** 1.24.5
- **Gin** - HTTPウェブフレームワーク
- **Testify** - テストライブラリ

## 今後の拡張予定

- JWT認証の実装
- データベース連携
- API レート制限の実装
- ログ設定の改善
- Docker対応
- CI/CD パイプライン

## API使用例

```bash
# Hello World
curl http://localhost:8080/

# ヘルスチェック
curl http://localhost:8080/health

# 認証が必要なエンドポイント
curl http://localhost:8080/api/protected
```
