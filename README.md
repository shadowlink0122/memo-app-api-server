# Memo App API Server

**🐳 DOCKER EXCLUSIVE APPLICATION**

**重要:** このアプリケーションはDocker専用で設計されており、ローカル環境でのGoコマンド実行はサポートしていません。すべての操作はDocker Composeを通じて行ってください。

Go + Ginを使用したREST APIサーバーです。

自己管理をしやすくするためのメモアプリを想定しています。

マイクロサービス化したい & 静的型付け言語が好き、という理由からGoを採用しています。

### 実現したいこと

- **githubアカウントで登録可能** ✅
- **ローカル認証（ID/パスワード）もサポート** ✅
- **セキュリティ機能**
  - パスワード強度チェック ✅
  - 同一IPアドレスからの複数アカウント作成制限 ✅
  - JWT認証によるセキュアなトークン管理 ✅
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
│   │   └── config.go             # 設定管理（DB、S3、ログ等）
│   ├── models/
│   │   └── memo.go               # データモデル定義
│   ├── database/
│   │   └── database.go           # データベース接続管理
│   ├── repository/
│   │   ├── interface.go          # リポジトリインターフェース
│   │   └── memo_repository.go    # メモデータアクセス層
│   ├── service/
│   │   ├── interface.go          # サービスインターフェース
│   │   └── memo_service.go       # メモビジネスロジック
│   ├── handlers/
│   │   └── memo_handler.go       # メモAPIハンドラー
│   ├── routes/
│   │   └── routes.go             # ルート設定
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
│   ├── api_test.go               # APIテストコード
│   ├── models/                   # モデルテスト
│   ├── service/                  # サービステスト
│   ├── handlers/                 # ハンドラーテスト
│   ├── middleware/               # ミドルウェアテスト
│   ├── logger/                   # ログテスト
│   ├── storage/                  # ストレージテスト
│   ├── database/                 # データベーステスト
│   ├── integration/              # 統合テスト
│   └── e2e/                      # E2Eテスト
├── logs/                         # ログファイル出力先
├── scripts/
│   └── init-minio.sh            # MinIO初期化スクリプト
├── migrations/                  # データベースマイグレーション
│   ├── 000_create_test_db.sql      # テスト用データベース作成
│   ├── 001_initial_schema.up.sql   # 初期スキーマ作成
│   ├── 001_initial_schema.down.sql # 初期スキーマ削除
│   ├── 002_sample_data.up.sql      # サンプルデータ挿入
│   └── 002_sample_data.down.sql    # サンプルデータ削除
├── go.mod                       # Go モジュール定義
├── go.sum                       # 依存関係のハッシュ
├── Makefile                     # ビルド・テスト用コマンド
├── docker-compose.yml           # Docker構成（DB + MinIO）
└── README.md                    # このファイル
```

## 主な機能

### 認証システム

#### 新規登録・ログイン機能
- **GitHub OAuth認証**: GitHubアカウントを使用した簡単登録
- **ローカル認証**: ID/パスワードによる従来型の認証
- **セキュリティ機能**:
  - パスワード強度チェック（8文字以上、大文字・小文字・数字・記号を含む）
  - ユーザー名フォーマット検証（3-30文字、英数字とアンダースコア）
  - 同一IPアドレスからの複数アカウント作成制限（デフォルト: 3アカウント/IP）
- **JWT認証**: セキュアなアクセストークンとリフレッシュトークンの管理
- **アカウント管理**: アクティブ/非アクティブ状態の管理

#### APIエンドポイント
- `POST /api/auth/register` - ローカル認証での新規登録
- `POST /api/auth/login` - ローカル認証でのログイン
- `GET /api/auth/github/url` - GitHub認証URL取得
- `GET /api/auth/github/callback` - GitHub認証コールバック
- `POST /api/auth/refresh` - アクセストークンの更新
- `GET /api/profile` - 現在のユーザープロフィール取得

### メモAPI

#### メモ管理機能
- **CRUD操作**: メモの作成、読み取り、更新、削除
- **カテゴリ機能**: メモをカテゴリ別に分類
- **タグ機能**: 複数のタグによるメモの分類
- **優先度設定**: low/medium/high の優先度設定
- **ステータス管理**: active/archived によるメモの状態管理
- **検索機能**: タイトルとコンテンツの全文検索
- **フィルタリング**: カテゴリ、ステータス、優先度による絞り込み
- **ページネーション**: 大量のメモの効率的な取得

#### APIエンドポイント

##### パブリック（認証不要）
- `GET /` - Hello World（JSON形式）
- `GET /health` - ヘルスチェック
- `GET /hello` - Hello World（テキスト形式）

##### メモAPI（認証必要）
- `POST /api/memos` - メモの作成
- `GET /api/memos` - メモ一覧取得（フィルタ・ページネーション対応）
- `GET /api/memos/:id` - 特定のメモ取得
- `PUT /api/memos/:id` - メモの更新
- `DELETE /api/memos/:id` - メモの削除（アクティブ→アーカイブ、アーカイブ済み→完全削除）
- `DELETE /api/memos/:id/permanent` - メモの完全削除
- `PATCH /api/memos/:id/archive` - メモのアーカイブ
- `PATCH /api/memos/:id/restore` - アーカイブメモの復元
- `GET /api/memos/search?q=検索語` - メモの検索

##### その他プライベート
- `GET /api/protected` - 認証が必要なエンドポイント（デモ用）

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

**重要:** このアプリケーションはDocker専用で設計されています。ローカル環境でのGoコマンド実行はサポートしていません。

- Docker 20.x以上
- Docker Compose 2.x以上

### 環境変数設定

環境変数ファイルをコピーして設定：

```bash
cp .env.example .env
```

主要な設定項目：

```bash
# サーバー設定
SERVER_PORT=8000

# 認証設定
JWT_SECRET=your-jwt-secret-key
JWT_EXPIRES_IN=24h
REFRESH_TOKEN_EXPIRES_IN=168h

# GitHub OAuth設定
GITHUB_CLIENT_ID=your-github-client-id
GITHUB_CLIENT_SECRET=your-github-client-secret
GITHUB_REDIRECT_URL=http://localhost:8000/api/auth/github/callback

# IP制限設定
MAX_REGISTRATIONS_PER_IP=3
IP_LIMIT_DURATION=24h

# データベース設定
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=memo_app

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

### Docker環境での起動

#### 1. 全サービスの起動

```bash
# 全サービスを起動（PostgreSQL + MinIO + アプリ）
docker compose up -d

# MinIOバケットを初期化
./scripts/init-minio.sh

# ログを確認
docker compose logs -f app
```

#### 2. 個別サービスの起動

```bash
# データベースとMinIOのみ起動
docker compose up -d db minio

# アプリケーションも含めて起動
docker compose up -d
```

サーバーは `http://localhost:8000` で起動します。

#### 3. サービス管理

```bash
# サービス停止
docker compose down

# ボリュームも含めて完全削除
docker compose down -v

# ログリアルタイム表示
docker compose logs -f

# 特定サービスのログ
docker compose logs -f app
```

### 本番環境での起動

本番環境では外部のマネージドサービス（AWS RDS、S3等）を使用し、アプリケーションコンテナのみを実行します。

#### 1. 本番環境用設定

```bash
# 本番環境用の環境変数をコピー
cp .env.production .env

# 必要な値を設定（特に以下は必須）
# DB_HOST=your-rds-endpoint.region.rds.amazonaws.com
# DB_PASSWORD=your-secure-password
# S3_ACCESS_KEY_ID=your-aws-access-key
# S3_SECRET_ACCESS_KEY=your-aws-secret-key
```

#### 2. 本番環境での起動

```bash
# 本番環境用Docker Composeで起動
make docker-prod-up

# または直接実行
docker compose -f docker-compose.prod.yml up -d

# ログを確認
docker compose -f docker-compose.prod.yml logs -f app

# 監視サービスも起動する場合
docker compose -f docker-compose.prod.yml --profile monitoring up -d
```

#### 3. 本番環境での停止

```bash
# 本番環境用サービスを停止
make docker-prod-down

# または直接実行
docker compose -f docker-compose.prod.yml down
```

#### 4. 前提条件（本番環境）

- **AWS RDS**: PostgreSQL 15以上のインスタンス
- **AWS S3**: ログ保存用のバケット
- **セキュリティグループ**: 適切なポート（80, 443）の開放
- **SSL証明書**: HTTPS通信用の証明書設定

### 管理画面アクセス

- **MinIO管理画面**: http://localhost:9001（開発環境のみ）
  - ユーザー名: `minioadmin`
  - パスワード: `minioadmin`
- **API サーバー**: http://localhost:8000（開発環境）/ http://your-domain.com（本番環境）

## テスト

### Docker環境でのテスト実行

**重要:** すべてのテストはDocker環境でのみ実行してください。ローカルGoコマンドでのテスト実行はサポートしていません。

```bash
# Docker環境でテストコンテナを実行
docker compose -f docker-compose.test.yml up --build --abort-on-container-exit

# または個別にテスト実行
docker compose exec app make test

# 特定のテスト種別を実行
docker compose exec app make test-unit          # ユニットテスト
docker compose exec app make test-integration   # 統合テスト
docker compose exec app make test-database      # データベーステスト
docker compose exec app make test-e2e          # E2Eテスト

# テストカバレッジ付きで実行
docker compose exec app make test-coverage
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

- **すべてのテスト**: Docker Composeが起動している必要があります
- **データベーステスト**: PostgreSQLサービスが必要（`docker compose up -d db`）
- **E2Eテスト**: 全サービスが必要（`docker compose up -d`）

## 開発とビルド

### Docker環境での開発

**重要:** ローカルでのGoコマンド実行はサポートしていません。すべての開発作業はDocker環境で行ってください。

```bash
# アプリケーションをDocker環境でビルド
docker compose exec app make build

# Dockerコンテナ内でシェルを開く（開発作業用）
docker compose exec app /bin/sh

# ファイル変更の監視（Hot Reload）
# docker-compose.ymlでvolumesがマウントされているため
# ローカルでファイルを編集すると自動的にコンテナに反映されます
```

### 利用可能なDockerコマンド

```bash
# Docker環境の管理
docker compose up -d          # 全サービス起動
docker compose down           # 全サービス停止
docker compose logs -f app    # アプリログの監視
docker compose restart app   # アプリサービスの再起動

# コンテナ内でのコマンド実行
docker compose exec app make test              # テストを実行
docker compose exec app make test-coverage     # テストカバレッジを生成
docker compose exec app make build             # アプリケーションをビルド
```

## 使用技術

- **Go** 1.24.5
- **Gin** - HTTPウェブフレームワーク  
- **Testify** - テストライブラリ
- **Docker** - コンテナ化プラットフォーム
- **Docker Compose** - マルチコンテナ管理
- **PostgreSQL** - リレーショナルデータベース
- **MinIO** - S3互換オブジェクトストレージ

## 今後の拡張予定

- JWT認証の実装
- API レート制限の実装
- ログ設定の改善
- CI/CD パイプライン

## Swagger/OpenAPI統合

このプロジェクトはSwagger/OpenAPIによるAPI仕様書とドキュメント管理を行っています。

### API仕様書とドキュメント

- **API仕様書**: `api/swagger.yaml` - OpenAPI 3.0.3形式でAPI仕様を定義
- **インタラクティブドキュメント**: `make swagger-serve` でSwagger UIを起動（http://localhost:7000/docs）

### 利用可能なコマンド

```bash
# Swagger UIでドキュメント表示
make swagger-serve

# API仕様の妥当性チェック
make swagger-validate

# Swagger関連ヘルプ
make swagger-docs
```

詳細な使用方法については [`docs/SWAGGER_INTEGRATION.md`](docs/SWAGGER_INTEGRATION.md) をご覧ください。

## バリデーションとセキュリティ

このプロジェクトは包括的なバリデーションとセキュリティ機能を実装しています。

### 多層バリデーション戦略

1. **HTTP層**: Ginのbindingタグによる基本的な構造・型チェック
2. **アプリケーション層**: カスタムバリデーターによる詳細なルール検証
3. **データベース層**: CHECK制約、NOT NULL制約による最終防衛線

### セキュリティ対策

- **SQLインジェクション対策**: パラメータ化クエリ + カスタムサニタイザー
- **XSS対策**: HTMLエスケープ処理
- **入力サニタイゼーション**: 危険な文字列パターンの検出と除去
- **ID検証**: 数値以外の文字列や異常に長いIDの拒否

### バリデーション機能

```bash
# バリデーションテストの実行
make docker-test-validation

# セキュリティテストの実行
make docker-test-security

# 全てのテストを実行
make docker-test
```

### 対応するバリデーションルール

- **必須フィールド**: `required` タグ
- **長さ制限**: `max`, `min` タグ
- **列挙値**: `oneof` タグ
- **安全な文字**: `safe_text`, `safe_category`, `safe_tag` カスタムルール
- **SQLインジェクション検出**: `no_sql_injection` カスタムルール

詳細については [`docs/VALIDATION_BEST_PRACTICES.md`](docs/VALIDATION_BEST_PRACTICES.md) をご覧ください。
