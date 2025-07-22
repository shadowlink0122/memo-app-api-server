package security

import (
	"testing"

	"memo-app/src/security"

	"github.com/stretchr/testify/assert"
)

func TestSQLSanitizer_ValidateSearchQuery(t *testing.T) {
	sanitizer := security.NewSQLSanitizer()

	tests := []struct {
		name      string
		query     string
		shouldErr bool
	}{
		{
			name:      "正常な検索クエリ",
			query:     "正常な検索テキスト",
			shouldErr: false,
		},
		{
			name:      "空の検索クエリ",
			query:     "",
			shouldErr: false,
		},
		{
			name:      "SQLインジェクション試行 - UNION",
			query:     "test UNION SELECT * FROM users",
			shouldErr: true,
		},
		{
			name:      "SQLインジェクション試行 - DROP",
			query:     "'; DROP TABLE memos; --",
			shouldErr: true,
		},
		{
			name:      "SQLインジェクション試行 - コメント",
			query:     "test -- comment",
			shouldErr: true,
		},
		{
			name:      "XSS試行",
			query:     "<script>alert('xss')</script>",
			shouldErr: true,
		},
		{
			name:      "長すぎるクエリ",
			query:     string(make([]rune, 501)),
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sanitizer.ValidateSearchQuery(tt.query)
			
			if tt.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSQLSanitizer_SanitizeSearchQuery(t *testing.T) {
	sanitizer := security.NewSQLSanitizer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "基本的なサニタイゼーション",
			input:    "  Hello World  ",
			expected: "Hello World",
		},
		{
			name:     "特殊文字の除去",
			input:    "test<>\"'&|+",
			expected: "test",
		},
		{
			name:     "日本語文字の保持",
			input:    "こんにちは 世界",
			expected: "こんにちは 世界",
		},
		{
			name:     "連続する空白の正規化",
			input:    "test    multiple   spaces",
			expected: "test multiple spaces",
		},
		{
			name:     "空文字列",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.SanitizeSearchQuery(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSQLSanitizer_ValidateLimitOffset(t *testing.T) {
	sanitizer := security.NewSQLSanitizer()

	tests := []struct {
		name      string
		limit     int
		offset    int
		shouldErr bool
	}{
		{
			name:      "正常な値",
			limit:     10,
			offset:    0,
			shouldErr: false,
		},
		{
			name:      "最大限界値",
			limit:     1000,
			offset:    100000,
			shouldErr: false,
		},
		{
			name:      "limit が小さすぎる",
			limit:     0,
			offset:    0,
			shouldErr: true,
		},
		{
			name:      "limit が大きすぎる",
			limit:     1001,
			offset:    0,
			shouldErr: true,
		},
		{
			name:      "offset が負の値",
			limit:     10,
			offset:    -1,
			shouldErr: true,
		},
		{
			name:      "offset が大きすぎる",
			limit:     10,
			offset:    100001,
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sanitizer.ValidateLimitOffset(tt.limit, tt.offset)
			
			if tt.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSQLSanitizer_ValidateOrderBy(t *testing.T) {
	sanitizer := security.NewSQLSanitizer()

	tests := []struct {
		name      string
		orderBy   string
		shouldErr bool
	}{
		{
			name:      "有効なカラム名",
			orderBy:   "title",
			shouldErr: false,
		},
		{
			name:      "有効なカラム名とDESC",
			orderBy:   "created_at desc",
			shouldErr: false,
		},
		{
			name:      "有効なカラム名とASC",
			orderBy:   "id asc",
			shouldErr: false,
		},
		{
			name:      "空文字列",
			orderBy:   "",
			shouldErr: false,
		},
		{
			name:      "無効なカラム名",
			orderBy:   "invalid_column",
			shouldErr: true,
		},
		{
			name:      "無効な方向",
			orderBy:   "title invalid_direction",
			shouldErr: true,
		},
		{
			name:      "SQLインジェクション試行",
			orderBy:   "title; DROP TABLE memos;",
			shouldErr: true,
		},
		{
			name:      "複雑な式",
			orderBy:   "title asc, id desc",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sanitizer.ValidateOrderBy(tt.orderBy)
			
			if tt.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSQLSanitizer_EscapeForLike(t *testing.T) {
	sanitizer := security.NewSQLSanitizer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "通常の文字列",
			input:    "normal text",
			expected: "normal text",
		},
		{
			name:     "パーセント文字のエスケープ",
			input:    "test%pattern",
			expected: "test\\%pattern",
		},
		{
			name:     "アンダースコア文字のエスケープ",
			input:    "test_pattern",
			expected: "test\\_pattern",
		},
		{
			name:     "バックスラッシュのエスケープ",
			input:    "test\\pattern",
			expected: "test\\\\pattern",
		},
		{
			name:     "複数の特殊文字",
			input:    "test%_\\pattern",
			expected: "test\\%\\_\\\\pattern",
		},
		{
			name:     "日本語文字列",
			input:    "テスト%パターン",
			expected: "テスト\\%パターン",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.EscapeForLike(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
