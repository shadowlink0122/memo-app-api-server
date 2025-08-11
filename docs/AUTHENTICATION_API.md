# 認証API仕様書

このドキュメントでは、memo-app-api-serverの認証システムについて詳しく説明します。

## 目次

1. [認証システム概要](#認証システム概要)
2. [認証方式](#認証方式)
3. [APIエンドポイント](#apiエンドポイント)
4. [セキュリティ機能](#セキュリティ機能)
5. [使用例](#使用例)

## 認証システム概要

memo-app-api-serverは以下の認証機能を提供します：

- **JWT ベース認証**: アクセストークンとリフレッシュトークンによる認証
- **ローカル認証**: メールアドレス/パスワードでの認証
- **GitHub OAuth認証**: GitHubアカウントでの認証
- **セキュアなログアウト**: トークンブラックリスト機能
- **IP制限**: 同一IPからの複数アカウント作成制限

## 認証方式

### 1. JWT トークン

アクセストークン（短期間有効）とリフレッシュトークン（長期間有効）の2つのトークンを使用。

#### トークンの種類
- **アクセストークン**: API呼び出しに必要（デフォルト: 24時間有効）
- **リフレッシュトークン**: アクセストークンの更新に使用（デフォルト: 7日間有効）

#### 認証ヘッダー
```
Authorization: Bearer <アクセストークン>
```

### 2. トークンブラックリスト

ログアウト時にトークンを無効化し、不正利用を防止。

## APIエンドポイント

### 1. 新規登録

**エンドポイント**: `POST /api/auth/register`

**リクエスト**:
```json
{
  "username": "testuser",
  "email": "user@example.com",
  "password": "SecurePass123!"
}
```

**レスポンス**:
```json
{
  "message": "Registration successful",
  "data": {
    "user": {
      "id": 1,
      "username": "testuser",
      "email": "user@example.com",
      "is_active": true,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    },
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }
}
```

### 2. ログイン

**エンドポイント**: `POST /api/auth/login`

**リクエスト**:
```json
{
  "email": "user@example.com",
  "password": "SecurePass123!"
}
```

**レスポンス**:
```json
{
  "message": "Login successful",
  "data": {
    "user": {
      "id": 1,
      "username": "testuser",
      "email": "user@example.com",
      "is_active": true,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    },
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }
}
```

### 3. ログアウト

**エンドポイント**: `POST /api/auth/logout`

**ヘッダー**:
```
Authorization: Bearer <アクセストークン>
```

**レスポンス**:
```json
{
  "message": "Successfully logged out"
}
```

**機能**:
- 送信されたアクセストークンをブラックリストに追加
- 以降そのトークンでの認証は無効になる

### 4. トークン更新

**エンドポイント**: `POST /api/auth/refresh`

**リクエスト**:
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**レスポンス**:
```json
{
  "message": "Token refreshed successfully",
  "data": {
    "user": {
      "id": 1,
      "username": "testuser",
      "email": "user@example.com",
      "is_active": true,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    },
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }
}
```

### 5. GitHub OAuth認証

#### 認証URL取得
**エンドポイント**: `GET /api/auth/github/url`

**パラメータ**:
- `state` (オプション): CSRF保護用の状態パラメータ

**レスポンス**:
```json
{
  "auth_url": "https://github.com/login/oauth/authorize?client_id=...&state=..."
}
```

#### コールバック処理
**エンドポイント**: `GET /api/auth/github/callback`

**パラメータ**:
- `code`: GitHubから返される認証コード
- `state`: CSRF保護用の状態パラメータ

### 6. プロフィール取得

**エンドポイント**: `GET /api/profile`

**ヘッダー**:
```
Authorization: Bearer <アクセストークン>
```

**レスポンス**:
```json
{
  "data": {
    "id": 1,
    "username": "testuser",
    "email": "user@example.com",
    "is_active": true,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

## セキュリティ機能

### 1. パスワード要件

- **最小長**: 8文字以上
- **文字種**: 大文字、小文字、数字、記号を含む
- **ハッシュ化**: bcryptを使用

### 2. ユーザー名要件

- **長さ**: 3-30文字
- **文字**: 英数字とアンダースコアのみ
- **一意性**: 重複不可

### 3. IP制限

- **制限数**: 同一IPから最大3アカウントまで
- **クールダウン**: 24時間のクールダウン期間

### 4. トークンセキュリティ

- **署名**: HMAC-SHA256
- **有効期限**: アクセストークン24時間、リフレッシュトークン7日
- **ブラックリスト**: ログアウト時のトークン無効化

## 使用例

### 一般的な認証フロー

```bash
# 1. 新規登録
curl -X POST http://localhost:8000/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "SecurePass123!"
  }'

# 2. ログイン
curl -X POST http://localhost:8000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "SecurePass123!"
  }'

# 3. 認証が必要なAPIの呼び出し
curl -X GET http://localhost:8000/api/profile \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

# 4. ログアウト
curl -X POST http://localhost:8000/api/auth/logout \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

### GitHub OAuth認証フロー

```bash
# 1. 認証URL取得
curl -X GET "http://localhost:8000/api/auth/github/url?state=random-state"

# 2. ブラウザでGitHub認証
# 返されたauth_urlにアクセスし、GitHubで認証

# 3. コールバック処理（自動）
# GitHubがコールバックURLにリダイレクト
```

## エラーハンドリング

### よくあるエラー

```json
// 401 Unauthorized
{
  "error": "Invalid token"
}

// 400 Bad Request
{
  "error": "Invalid request format"
}

// 409 Conflict
{
  "error": "Email already exists"
}

// 429 Too Many Requests
{
  "error": "Rate limit exceeded"
}
```

## 設定

### 環境変数

```env
# JWT設定
JWT_SECRET=your-secret-key
JWT_EXPIRES_IN=24h
REFRESH_EXPIRES_IN=168h

# GitHub OAuth設定
GITHUB_CLIENT_ID=your-github-client-id
GITHUB_CLIENT_SECRET=your-github-client-secret
GITHUB_REDIRECT_URL=http://localhost:8000/api/auth/github/callback

# セキュリティ設定
MAX_ACCOUNTS_PER_IP=3
IP_COOLDOWN_PERIOD=24h
```
