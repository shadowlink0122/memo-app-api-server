# Swagger/OpenAPI API Documentation

このプロジェクトではSwagger/OpenAPIによるAPI仕様書とドキュメント管理を行っています。

## 📚 概要

- **API仕様書**: `api/swagger.yaml` - OpenAPI 3.0.3形式でAPI仕様を定義
- **インタラクティブドキュメント**: Swagger UIでリアルタイムAPIドキュメント表示
- **仕様バリデーション**: API仕様の妥当性チェック

## 🚀 セットアップ

### 前提条件

- Docker（Swagger UIとバリデーション用）

### 基本的な使用方法

```bash
# API仕様のバリデーション
make swagger-validate

# Swagger UIでドキュメント表示
make swagger-serve

# Swagger関連ヘルプ
make swagger-docs
```

## 📖 ドキュメント表示

```bash
# Swagger UIでAPIドキュメントを表示
make swagger-serve

# http://localhost:8081/docs でアクセス可能
# 終了するには Ctrl+C を押してください
```

Swagger UIでは以下の機能が利用できます：
- 全APIエンドポイントの一覧表示
- リクエスト/レスポンススキーマの詳細
- インタラクティブなAPIテスト機能
- 認証設定の確認

## 🔍 API仕様のバリデーション

```bash
# OpenAPI仕様の妥当性チェック
make swagger-validate
```

このコマンドは以下をチェックします：
- YAML構文の正確性
- OpenAPI 3.0.3仕様への準拠
- スキーマ定義の整合性
- エンドポイント定義の妥当性

## 📝 API仕様書の編集

### ファイル構成

```
api/
└── swagger.yaml    # API仕様書（OpenAPI 3.0.3形式）
```

### 編集のベストプラクティス

1. **変更前のバリデーション**
   ```bash
   make swagger-validate
   ```

2. **仕様書の編集**
   `api/swagger.yaml`を編集

3. **変更後の確認**
   ```bash
   make swagger-validate
   make swagger-serve
   ```

4. **ドキュメントの確認**
   http://localhost:8081/docs でUI表示を確認

## 🏗️ 実装との連携

### 型安全性の確保

Swagger仕様書を参考に、以下の要素を実装で一致させてください：

1. **エンドポイントパス**
   ```yaml
   # swagger.yaml
   /api/memos:
     post: ...
   ```
   ```go
   // routes.go
   memos.POST("", memoHandler.CreateMemo)
   ```

2. **リクエスト/レスポンス構造**
   ```yaml
   # swagger.yaml
   CreateMemoRequest:
     properties:
       title:
         type: string
         maxLength: 200
   ```
   ```go
   // dto.go
   type CreateMemoRequestDTO struct {
       Title string `json:"title" binding:"required,max=200"`
   }
   ```

3. **HTTPステータスコード**
   ```yaml
   # swagger.yaml
   responses:
     '201':
       description: メモ作成成功
   ```
   ```go
   // handler.go
   c.JSON(http.StatusCreated, response)
   ```

### 一貫性の維持

- API仕様変更時は実装も同時に更新
- バリデーションルールを仕様書と実装で統一
- エラーレスポンス形式の一致

## 🔄 開発ワークフロー

### API仕様ファースト

1. `api/swagger.yaml`でAPI仕様を設計
2. `make swagger-validate`で妥当性チェック
3. `make swagger-serve`でドキュメント確認
4. 仕様に基づいて実装を作成/更新
5. 実装完了後、再度仕様書との整合性を確認

### 実装ファースト（既存機能）

1. 実装の変更/追加
2. `api/swagger.yaml`を実装に合わせて更新
3. `make swagger-validate`で妥当性チェック
4. `make swagger-serve`でドキュメント確認

## 📁 プロジェクト内での位置づけ

```
memo-app-api-server/
├── api/
│   └── swagger.yaml              # API仕様書
├── src/
│   ├── interface/handler/
│   │   ├── dto.go               # リクエスト/レスポンス型定義
│   │   └── memo_handler.go      # APIハンドラー実装
│   └── routes/
│       └── routes.go            # ルーティング設定
└── docs/
    └── SWAGGER_INTEGRATION.md   # このドキュメント
```

## 🎯 利点

1. **API設計の可視化** - 実装前にAPI設計を確認・共有
2. **ドキュメント自動化** - コードコメントに依存しない最新ドキュメント
3. **チーム連携** - 共通のAPI仕様による開発効率向上
4. **テスト支援** - Swagger UIでのインタラクティブテスト
5. **保守性向上** - 仕様変更の影響範囲を事前に把握

## ⚠️ 注意事項

### 仕様書と実装の同期

- API仕様を変更した場合は、対応する実装も必ず更新してください
- 逆に実装を変更した場合は、仕様書も更新してください
- 定期的に`make swagger-validate`でチェックしてください

### バージョン管理

- `api/swagger.yaml`はGitで管理されます
- 破壊的な変更を行う場合は、バージョン番号を更新してください
- 重要な変更はコミットメッセージで明記してください

## 🔗 参考リンク

- [OpenAPI Specification](https://swagger.io/specification/)
- [Swagger UI](https://swagger.io/tools/swagger-ui/)
- [OpenAPI 3.0.3 Documentation](https://spec.openapis.org/oas/v3.0.3)
