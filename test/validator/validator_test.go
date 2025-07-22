package validator

import (
	"testing"

	"memo-app/src/validator"

	"github.com/stretchr/testify/assert"
)

func TestCustomValidator_Validate(t *testing.T) {
	v := validator.NewCustomValidator()

	t.Run("有効なDTO", func(t *testing.T) {
		type TestDTO struct {
			Title    string   `validate:"required,max=200,safe_text,no_sql_injection"`
			Content  string   `validate:"required,safe_text,no_sql_injection"`
			Category string   `validate:"omitempty,max=50,safe_category"`
			Tags     []string `validate:"omitempty,dive,max=30,safe_tag"`
			Priority string   `validate:"omitempty,oneof=low medium high"`
		}

		dto := TestDTO{
			Title:    "有効なタイトル",
			Content:  "有効なコンテンツです。",
			Category: "テスト",
			Tags:     []string{"タグ1", "タグ2"},
			Priority: "medium",
		}

		err := v.Validate(&dto)
		assert.NoError(t, err)
	})

	t.Run("SQLインジェクション試行", func(t *testing.T) {
		type TestDTO struct {
			Title   string `validate:"required,safe_text,no_sql_injection"`
			Content string `validate:"required,safe_text,no_sql_injection"`
		}

		maliciousCases := []TestDTO{
			{Title: "'; DROP TABLE memos; --", Content: "test"},
			{Title: "test", Content: "' OR 1=1 --"},
			{Title: "UNION SELECT * FROM users", Content: "test"},
			{Title: "test", Content: "<script>alert('xss')</script>"},
		}

		for _, testCase := range maliciousCases {
			err := v.Validate(&testCase)
			assert.Error(t, err, "悪意のある入力を検出できませんでした: %+v", testCase)
			
			if validationErrors, ok := err.(validator.ValidationErrors); ok {
				assert.NotEmpty(t, validationErrors.Errors)
			}
		}
	})

	t.Run("長すぎる入力", func(t *testing.T) {
		type TestDTO struct {
			Title string `validate:"required,max=200"`
		}

		dto := TestDTO{
			Title: string(make([]rune, 201)), // 201文字
		}

		err := v.Validate(&dto)
		assert.Error(t, err)
	})

	t.Run("無効な列挙値", func(t *testing.T) {
		type TestDTO struct {
			Priority string `validate:"omitempty,oneof=low medium high"`
		}

		dto := TestDTO{
			Priority: "invalid_priority",
		}

		err := v.Validate(&dto)
		assert.Error(t, err)
	})
}

func TestCustomValidator_SanitizeInput(t *testing.T) {
	v := validator.NewCustomValidator()

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
			name:     "HTMLエスケープ",
			input:    "<script>alert('xss')</script>",
			expected: "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;",
		},
		{
			name:     "連続する空白の正規化",
			input:    "Hello    World   Test",
			expected: "Hello World Test",
		},
		{
			name:     "日本語文字列",
			input:    "  こんにちは　世界  ",
			expected: "こんにちは　世界",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.SanitizeInput(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCustomValidator_SanitizeTags(t *testing.T) {
	v := validator.NewCustomValidator()

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "基本的なタグのサニタイズ",
			input:    []string{"tag1", "tag2", "tag1"}, // 重複含む
			expected: []string{"tag1", "tag2"},
		},
		{
			name:     "空文字列とスペースの除去",
			input:    []string{"", "  ", "tag1", "  tag2  "},
			expected: []string{"tag1", "tag2"},
		},
		{
			name:     "長すぎるタグの除去",
			input:    []string{"short", string(make([]rune, 31))},
			expected: []string{"short"},
		},
		{
			name:     "不正な文字を含むタグの除去",
			input:    []string{"valid_tag", "invalid<>tag", "日本語タグ"},
			expected: []string{"valid_tag", "日本語タグ"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.SanitizeTags(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCustomValidator_ValidateID(t *testing.T) {
	v := validator.NewCustomValidator()

	tests := []struct {
		name      string
		input     string
		expected  int
		shouldErr bool
	}{
		{
			name:      "有効なID",
			input:     "123",
			expected:  123,
			shouldErr: false,
		},
		{
			name:      "ゼロID",
			input:     "0",
			expected:  0,
			shouldErr: true,
		},
		{
			name:      "負の数",
			input:     "-1",
			expected:  0,
			shouldErr: true,
		},
		{
			name:      "非数値文字",
			input:     "abc",
			expected:  0,
			shouldErr: true,
		},
		{
			name:      "SQLインジェクション試行",
			input:     "1; DROP TABLE memos;",
			expected:  0,
			shouldErr: true,
		},
		{
			name:      "長すぎるID",
			input:     "12345678901",
			expected:  0,
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := v.ValidateID(tt.input)
			
			if tt.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
