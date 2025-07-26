# アーカイブ機能の実装

## 概要

メモアプリケーションにアーカイブ機能を実装しました。この機能により、アーカイブされたメモは通常のAPIエンドポイントからは取得できなくなり、専用の `/archive` エンドポイントからのみアクセス可能になります。

## 実装内容

### 1. アーカイブ専用エンドポイント

新しく追加されたエンドポイント：
- `GET /api/memos/archive` - アーカイブされたメモのみを取得

### 2. 既存エンドポイントの動作変更

以下のエンドポイントでは、アーカイブされたメモが除外されるようになりました：

#### メモ一覧取得
- `GET /api/memos` - デフォルトで `status=active` のメモのみを返す
- `GET /api/memos?status=active` - 明示的にアクティブなメモのみを取得
- `GET /api/memos?status=archived` - 明示的にアーカイブされたメモのみを取得（非推奨、`/archive`を使用推奨）

#### メモ検索
- `GET /api/memos/search?q=検索語` - デフォルトで `status=active` のメモのみを検索対象とする
- `GET /api/memos/search?q=検索語&status=active` - 明示的にアクティブなメモのみを検索
- `GET /api/memos/search?q=検索語&status=archived` - 明示的にアーカイブされたメモのみを検索

#### 個別メモ取得
- `GET /api/memos/{id}` - ステータスに関係なく取得可能（変更なし）

## 技術実装詳細

### 1. ルーティング変更

`src/routes/routes.go` に新しいエンドポイントを追加：

```go
// アーカイブされたメモの一覧を取得
memoGroup.GET("/archive", memoHandler.ListArchivedMemos)
```

### 2. ハンドラ実装

`src/interface/handler/memo_handler.go` に以下の変更を実装：

#### 新規追加: ListArchivedMemos メソッド
```go
func (h *MemoHandler) ListArchivedMemos(c *gin.Context) {
    // アーカイブされたメモのみを取得するハンドラ
    // status="archived" を強制設定
}
```

#### 変更: ListMemos メソッド
- statusパラメータが指定されていない場合、デフォルトで `"active"` を設定
- アーカイブされたメモを通常のリストから除外

#### 変更: SearchMemos メソッド
- statusパラメータが指定されていない場合、デフォルトで `"active"` を設定
- アーカイブされたメモを検索対象から除外

### 3. テスト実装

`test/integration/archive_integration_test.go` に包括的なテストスイートを実装：

1. **TestArchiveExclusionFromRegularList**: 通常のメモ一覧からアーカイブメモが除外されることを確認
2. **TestArchiveEndpointShowsOnlyArchivedMemos**: `/archive` エンドポイントがアーカイブメモのみを返すことを確認
3. **TestSearchExcludesArchivedMemos**: 検索機能からアーカイブメモが除外されることを確認
4. **TestIndividualMemoAccessStillWorks**: 個別メモアクセスがステータスに関係なく動作することを確認

## 使用例

### アクティブなメモのみを取得
```bash
# デフォルトでアクティブなメモのみ
curl -X GET "http://localhost:8080/api/memos"

# 明示的にアクティブなメモを指定
curl -X GET "http://localhost:8080/api/memos?status=active"
```

### アーカイブされたメモを取得
```bash
# 推奨方法：専用エンドポイント
curl -X GET "http://localhost:8080/api/memos/archive"

# 従来方法（非推奨）
curl -X GET "http://localhost:8080/api/memos?status=archived"
```

### 検索機能
```bash
# デフォルトでアクティブなメモのみを検索
curl -X GET "http://localhost:8080/api/memos/search?q=golang"

# 明示的にアクティブなメモを検索
curl -X GET "http://localhost:8080/api/memos/search?q=golang&status=active"
```

### 個別メモアクセス
```bash
# ステータスに関係なくアクセス可能
curl -X GET "http://localhost:8080/api/memos/123"
```

## 後方互換性

この実装は後方互換性を保っています：

1. 既存のAPIエンドポイントは引き続き動作します
2. `status` パラメータを明示的に指定することで、従来の動作を維持できます
3. 個別メモの取得は変更されていません

## 期待される動作

- **通常のユーザー体験**: メモ一覧や検索でアーカイブされたメモが表示されない
- **アーカイブ管理**: 専用エンドポイントでアーカイブされたメモを管理可能
- **データの安全性**: アーカイブされたメモは削除されず、必要時にアクセス可能

## テスト結果

全てのテストケースが成功し、アーカイブ機能が正常に動作することが確認されています：

```
=== RUN   TestArchiveTestSuite
=== RUN   TestArchiveTestSuite/TestArchiveEndpointShowsOnlyArchivedMemos
=== RUN   TestArchiveTestSuite/TestArchiveExclusionFromRegularList
=== RUN   TestArchiveTestSuite/TestIndividualMemoAccessStillWorks
=== RUN   TestArchiveTestSuite/TestSearchExcludesArchivedMemos
--- PASS: TestArchiveTestSuite (0.04s)
```

この実装により、ユーザーはアーカイブされたメモに煩わされることなく、通常のメモ管理を行えるようになりました。
