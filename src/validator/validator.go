package validator

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/go-playground/validator/v10"
)

// CustomValidator は拡張バリデーション機能を提供
type CustomValidator struct {
	validator           *validator.Validate
	categoryPattern     *regexp.Regexp
	tagPattern          *regexp.Regexp
	sqlInjectionPattern *regexp.Regexp
}

// ValidationError はバリデーションエラーの詳細情報
type ValidationError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Message string `json:"message"`
	Value   any    `json:"value,omitempty"`
}

// ValidationErrors は複数のバリデーションエラー
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

func (ve ValidationErrors) Error() string {
	return fmt.Sprintf("validation failed: %d errors", len(ve.Errors))
}

// NewCustomValidator creates a new custom validator instance
func NewCustomValidator() *CustomValidator {
	v := validator.New()
	cv := &CustomValidator{
		validator:           v,
		categoryPattern:     regexp.MustCompile(`^[a-zA-Z0-9_\-\x{3040}-\x{309F}\x{30A0}-\x{30FF}\x{4E00}-\x{9FAF}]+$`),   // 英数字、ひらがな、カタカナ、漢字
		tagPattern:          regexp.MustCompile(`^[a-zA-Z0-9_\-\x{3040}-\x{309F}\x{30A0}-\x{30FF}\x{4E00}-\x{9FAF}\s]+$`), // タグは空白も許可
		sqlInjectionPattern: regexp.MustCompile(`(?i)(\bunion\s+select\b|\bselect\s+.*\bfrom\b|\binsert\s+into\b|\bupdate\s+.*\bset\b|\bdelete\s+from\b|\bdrop\s+table\b|\bcreate\s+table\b|\balter\s+table\b|\bexec\s*\(|<script|</script>|onload\s*=|onerror\s*=|--|/\*|\*/|\|\||(\bor\b|\band\b)\s*(1\s*=\s*1|true|\d+\s*=\s*\d+))`),
	}

	// カスタムバリデーションルールを登録
	v.RegisterValidation("safe_text", cv.validateSafeText)
	v.RegisterValidation("safe_category", cv.validateSafeCategory)
	v.RegisterValidation("safe_tag", cv.validateSafeTag)
	v.RegisterValidation("no_sql_injection", cv.validateNoSQLInjection)

	return cv
}

// Validate validates a struct and returns detailed error information
func (cv *CustomValidator) Validate(s interface{}) error {
	if err := cv.validator.Struct(s); err != nil {
		var validationErrors []ValidationError

		for _, err := range err.(validator.ValidationErrors) {
			ve := ValidationError{
				Field: err.Field(),
				Tag:   err.Tag(),
				Value: err.Value(),
			}

			// カスタムエラーメッセージを生成
			ve.Message = cv.generateErrorMessage(err)
			validationErrors = append(validationErrors, ve)
		}

		return ValidationErrors{Errors: validationErrors}
	}
	return nil
}

// SanitizeInput sanitizes input data to prevent XSS and other attacks
func (cv *CustomValidator) SanitizeInput(input string) string {
	// HTMLエスケープ
	sanitized := html.EscapeString(input)

	// 前後の空白を除去
	sanitized = strings.TrimSpace(sanitized)

	// 連続する空白を単一の空白に変換
	sanitized = regexp.MustCompile(`\s+`).ReplaceAllString(sanitized, " ")

	return sanitized
}

// SanitizeTags sanitizes and normalizes tags
func (cv *CustomValidator) SanitizeTags(tags []string) []string {
	if len(tags) == 0 {
		return []string{}
	}

	seen := make(map[string]bool)
	result := make([]string, 0, len(tags))

	for _, tag := range tags {
		// サニタイズ
		sanitized := cv.SanitizeInput(tag)

		// 長さチェック
		if utf8.RuneCountInString(sanitized) > 30 {
			continue // 長すぎるタグは除外
		}

		// 重複チェック
		if sanitized != "" && !seen[sanitized] && cv.tagPattern.MatchString(sanitized) {
			seen[sanitized] = true
			result = append(result, sanitized)
		}
	}

	return result
}

// カスタムバリデーション関数

func (cv *CustomValidator) validateSafeText(fl validator.FieldLevel) bool {
	value := fl.Field().String()

	// SQLインジェクションパターンをチェック
	if cv.sqlInjectionPattern.MatchString(value) {
		return false
	}

	// 基本的な文字チェック（制御文字の排除）
	for _, r := range value {
		if r < 32 && r != 9 && r != 10 && r != 13 { // タブ、改行、復帰以外の制御文字を拒否
			return false
		}
	}

	return true
}

func (cv *CustomValidator) validateSafeCategory(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true // 任意フィールド
	}

	return cv.categoryPattern.MatchString(value) && !cv.sqlInjectionPattern.MatchString(value)
}

func (cv *CustomValidator) validateSafeTag(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true
	}

	return cv.tagPattern.MatchString(value) && !cv.sqlInjectionPattern.MatchString(value)
}

func (cv *CustomValidator) validateNoSQLInjection(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	return !cv.sqlInjectionPattern.MatchString(value)
}

// generateErrorMessage generates user-friendly error messages
func (cv *CustomValidator) generateErrorMessage(err validator.FieldError) string {
	field := err.Field()
	tag := err.Tag()
	value := err.Value()

	switch tag {
	case "required":
		return fmt.Sprintf("%s は必須項目です", field)
	case "max":
		return fmt.Sprintf("%s は %s 文字以下で入力してください", field, err.Param())
	case "min":
		return fmt.Sprintf("%s は %s 文字以上で入力してください", field, err.Param())
	case "oneof":
		return fmt.Sprintf("%s は有効な値を選択してください (許可された値: %s)", field, err.Param())
	case "safe_text":
		return fmt.Sprintf("%s に不正な文字が含まれています", field)
	case "safe_category":
		return fmt.Sprintf("%s は英数字、ひらがな、カタカナ、漢字、ハイフン、アンダースコアのみ使用できます", field)
	case "safe_tag":
		return fmt.Sprintf("%s は不正な文字が含まれています", field)
	case "no_sql_injection":
		return fmt.Sprintf("%s に危険なパターンが検出されました", field)
	default:
		return fmt.Sprintf("%s が無効です (値: %v)", field, value)
	}
}

// ValidateID validates ID parameters for SQL injection
func (cv *CustomValidator) ValidateID(idStr string) (int, error) {
	// 数値以外の文字をチェック
	if !regexp.MustCompile(`^\d+$`).MatchString(idStr) {
		return 0, fmt.Errorf("ID must be a positive integer")
	}

	// 長さチェック（異常に長いIDを防ぐ）
	if len(idStr) > 10 {
		return 0, fmt.Errorf("ID is too long")
	}

	// パースを試行
	var id int
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		return 0, fmt.Errorf("invalid ID format")
	}

	// 正の値チェック
	if id <= 0 {
		return 0, fmt.Errorf("ID must be positive")
	}

	return id, nil
}
