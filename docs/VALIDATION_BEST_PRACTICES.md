# バリデーションベストプラクティス

このドキュメントでは、memo-app-api-serverプロジェクトにおけるGoとデータベース側のバリデーションベストプラクティスを説明します。

## 目次

1. [Go言語側のバリデーション](#go言語側のバリデーション)
2. [データベース側のバリデーション](#データベース側のバリデーション)
3. [現在の実装](#現在の実装)
4. [推奨改善点](#推奨改善点)
5. [実装例](#実装例)

## Go言語側のバリデーション

### 多層バリデーション戦略

プロジェクトでは以下の3層でバリデーションを実装しています：

#### 1. HTTP層（DTOレベル）
**場所**: `src/interface/handler/dto.go`
**目的**: HTTPリクエストの基本的な構造・型チェック
**技術**: Ginのbindingタグ

```go
type CreateMemoRequestDTO struct {
    Title    string   `json:"title" binding:"required,max=200"`
    Content  string   `json:"content" binding:"required"`
    Category string   `json:"category" binding:"max=50"`
    Priority string   `json:"priority" binding:"omitempty,oneof=low medium high"`
}
```

**利点**:
- 早期のバリデーションエラー検出
- 自動的なHTTPエラーレスポンス
- 標準的なGoバリデーションタグの活用

#### 2. ビジネスロジック層（Usecaseレベル）
**場所**: `src/usecase/memo.go`
**目的**: ビジネスルールの検証とデータ正規化
**技術**: カスタムバリデーション関数

```go
func (u *memoUsecase) validateCreateRequest(req CreateMemoRequest) error {
    if req.Title == "" || len(req.Title) > 200 {
        return ErrInvalidTitle
    }
    if req.Content == "" {
        return ErrInvalidContent
    }
    if req.Priority != "" && !domain.Priority(req.Priority).IsValid() {
        return ErrInvalidPriority
    }
    return nil
}
```

**利点**:
- ビジネスルール固有の検証
- ドメイン型との連携
- エラーの詳細な制御

#### 3. ドメイン層（Entityレベル）
**場所**: `src/domain/entity.go`
**目的**: 型安全性とドメイン不変条件の保証
**技術**: 型定義とメソッド

```go
type Priority string

const (
    PriorityLow    Priority = "low"
    PriorityMedium Priority = "medium"
    PriorityHigh   Priority = "high"
)

func (p Priority) IsValid() bool {
    switch p {
    case PriorityLow, PriorityMedium, PriorityHigh:
        return true
    default:
        return false
    }
}
```

**利点**:
- コンパイル時の型安全性
- ドメインロジックの明確化
- 不正値の防止

## データベース側のバリデーション

### 現在の実装
**場所**: `migrations/001_initial_schema.up.sql`

```sql
CREATE TABLE IF NOT EXISTS memos (
    id SERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,                    -- 長さ制限
    content TEXT NOT NULL,                          -- NULL制約
    category VARCHAR(50),                           -- 長さ制限
    tags JSONB DEFAULT '[]'::jsonb,                -- 型制約とデフォルト値
    priority VARCHAR(10) NOT NULL DEFAULT 'medium' 
        CHECK (priority IN ('low', 'medium', 'high')), -- 値制約
    status VARCHAR(20) NOT NULL DEFAULT 'active' 
        CHECK (status IN ('active', 'archived')),      -- 値制約
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE, -- 外部キー制約
    is_public BOOLEAN NOT NULL DEFAULT false,       -- NULL制約とデフォルト値
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE
);
```

### DB制約の種類と利点

1. **NOT NULL制約**: 必須フィールドの保証
2. **CHECK制約**: 値の範囲・形式の制限
3. **UNIQUE制約**: 一意性の保証
4. **外部キー制約**: 参照整合性の保証
5. **デフォルト値**: データの一貫性確保

## 推奨改善点

### 1. バリデーションライブラリの導入（オプション）

現在の実装は良好ですが、より複雑なバリデーションが必要な場合は`github.com/go-playground/validator`の導入を検討：

```go
type CreateMemoRequestDTO struct {
    Title    string   `json:"title" validate:"required,max=200,min=1"`
    Content  string   `json:"content" validate:"required,min=1"`
    Category string   `json:"category" validate:"omitempty,max=50"`
    Tags     []string `json:"tags" validate:"dive,max=30"`
    Priority string   `json:"priority" validate:"omitempty,oneof=low medium high"`
}
```

### 2. サニタイゼーション機能の強化

```go
func (u *memoUsecase) sanitizeInput(req *CreateMemoRequest) {
    req.Title = strings.TrimSpace(req.Title)
    req.Content = strings.TrimSpace(req.Content)
    req.Category = strings.TrimSpace(req.Category)
    // HTMLエスケープやXSS対策が必要な場合
    req.Content = html.EscapeString(req.Content)
}
```

### 3. DB制約の追加検討

```sql
-- カテゴリの標準化
ALTER TABLE memos ADD CONSTRAINT check_category_format 
    CHECK (category ~ '^[a-zA-Z0-9_-]+$' OR category IS NULL);

-- タイトルの最小長制約
ALTER TABLE memos ADD CONSTRAINT check_title_length 
    CHECK (char_length(trim(title)) >= 1);

-- 完了日の論理制約
ALTER TABLE memos ADD CONSTRAINT check_completed_at_logic 
    CHECK (
        (status = 'archived' AND completed_at IS NOT NULL) OR 
        (status = 'active' AND completed_at IS NULL)
    );
```

### 4. エラーハンドリングの改善

```go
type ValidationError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
    Value   any    `json:"value,omitempty"`
}

type ValidationErrors struct {
    Errors []ValidationError `json:"errors"`
}

func (ve ValidationErrors) Error() string {
    return fmt.Sprintf("validation failed: %d errors", len(ve.Errors))
}
```

## 実装例

### カスタムバリデータの追加例

```go
// src/validator/memo_validator.go (新規作成を推奨)
package validator

import (
    "regexp"
    "strings"
    "unicode/utf8"
)

type MemoValidator struct {
    categoryPattern *regexp.Regexp
}

func NewMemoValidator() *MemoValidator {
    return &MemoValidator{
        categoryPattern: regexp.MustCompile(`^[a-zA-Z0-9_-]+$`),
    }
}

func (v *MemoValidator) ValidateTitle(title string) error {
    title = strings.TrimSpace(title)
    if title == "" {
        return errors.New("title is required")
    }
    if utf8.RuneCountInString(title) > 200 {
        return errors.New("title must be 200 characters or less")
    }
    return nil
}

func (v *MemoValidator) ValidateCategory(category string) error {
    if category == "" {
        return nil // 任意フィールド
    }
    if !v.categoryPattern.MatchString(category) {
        return errors.New("category can only contain letters, numbers, hyphens, and underscores")
    }
    return nil
}
```

### 統合テストでのバリデーション検証

```go
// test/validation_test.go (新規作成を推奨)
func TestValidationIntegration(t *testing.T) {
    tests := []struct {
        name          string
        request       CreateMemoRequestDTO
        expectStatus  int
        expectError   string
    }{
        {
            name: "valid request",
            request: CreateMemoRequestDTO{
                Title:    "Valid Title",
                Content:  "Valid content",
                Priority: "medium",
            },
            expectStatus: http.StatusCreated,
        },
        {
            name: "title too long",
            request: CreateMemoRequestDTO{
                Title:   strings.Repeat("a", 201),
                Content: "Valid content",
            },
            expectStatus: http.StatusBadRequest,
            expectError:  "title",
        },
        // 他のケースも追加...
    }
}
```

## ベストプラクティスまとめ

### Do（推奨）
- ✅ 多層バリデーション（HTTP/Business/Domain）の維持
- ✅ DB制約による最後の防衛線の確保
- ✅ 明確なエラーメッセージの提供
- ✅ 入力値のサニタイゼーション
- ✅ 型安全性の活用
- ✅ バリデーションロジックのテスト

### Don't（非推奨）
- ❌ バリデーションの単一層への集約
- ❌ DB制約の省略
- ❌ 曖昧なエラーメッセージ
- ❌ ユーザー入力の直接信頼
- ❌ 文字列型での列挙値の扱い
- ❌ バリデーション失敗時の詳細な内部情報の露出

## 監視とメトリクス

バリデーションエラーの監視も重要です：

```go
// メトリクス例
var validationFailureCounter = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "validation_failures_total",
        Help: "Total number of validation failures",
    },
    []string{"field", "error_type"},
)
```

この文書は定期的に見直し、プロジェクトの成長に合わせて更新してください。
