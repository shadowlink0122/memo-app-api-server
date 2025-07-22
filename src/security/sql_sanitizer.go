package security

import (
	"fmt"
	"regexp"
	"strings"
)

// SQLSanitizer provides SQL injection protection utilities
type SQLSanitizer struct {
	// 危険なSQLキーワードのパターン
	dangerousPatterns []*regexp.Regexp
}

// NewSQLSanitizer creates a new SQL sanitizer
func NewSQLSanitizer() *SQLSanitizer {
	patterns := []*regexp.Regexp{
		// SQLインジェクション攻撃パターン
		regexp.MustCompile(`(?i)(^|\s)(union|select|insert|update|delete|drop|create|alter|exec|execute|declare|grant|revoke|truncate|show|describe)\s`),
		regexp.MustCompile(`(?i)(--|/\*|\*/|;|'|"|\||&|\+|<|>|=|\(|\))`),
		regexp.MustCompile(`(?i)(script|javascript|vbscript|onload|onerror|alert|document|window|eval|expression)`),
		regexp.MustCompile(`(?i)(xp_|sp_|sys\.|information_schema|pg_|mysql\.)`),
	}

	return &SQLSanitizer{
		dangerousPatterns: patterns,
	}
}

// ValidateSearchQuery validates and sanitizes search queries
func (s *SQLSanitizer) ValidateSearchQuery(query string) error {
	if query == "" {
		return nil
	}

	// 長さチェック
	if len(query) > 500 {
		return fmt.Errorf("search query too long (max: 500 characters)")
	}

	// 危険なパターンをチェック
	for _, pattern := range s.dangerousPatterns {
		if pattern.MatchString(query) {
			return fmt.Errorf("potentially dangerous pattern detected in search query")
		}
	}

	return nil
}

// SanitizeSearchQuery sanitizes search query for safe database operations
func (s *SQLSanitizer) SanitizeSearchQuery(query string) string {
	if query == "" {
		return ""
	}

	// 基本的なサニタイゼーション
	sanitized := strings.TrimSpace(query)
	
	// 特殊文字の除去（検索クエリとして安全な文字のみ許可）
	// アルファベット、数字、ひらがな、カタカナ、漢字、空白のみ許可
	safeChars := regexp.MustCompile(`[^a-zA-Z0-9\s\x{3040}-\x{309F}\x{30A0}-\x{30FF}\x{4E00}-\x{9FAF}]`)
	sanitized = safeChars.ReplaceAllString(sanitized, "")
	
	// 連続する空白を単一の空白に変換
	sanitized = regexp.MustCompile(`\s+`).ReplaceAllString(sanitized, " ")
	
	// 前後の空白を除去
	sanitized = strings.TrimSpace(sanitized)
	
	return sanitized
}

// ValidateLimitOffset validates pagination parameters to prevent resource exhaustion
func (s *SQLSanitizer) ValidateLimitOffset(limit, offset int) error {
	if limit < 1 {
		return fmt.Errorf("limit must be positive")
	}
	if limit > 1000 {
		return fmt.Errorf("limit too large (max: 1000)")
	}
	if offset < 0 {
		return fmt.Errorf("offset must be non-negative")
	}
	if offset > 100000 {
		return fmt.Errorf("offset too large (max: 100000)")
	}
	return nil
}

// ValidateOrderBy validates ORDER BY clauses to prevent SQL injection
func (s *SQLSanitizer) ValidateOrderBy(orderBy string) error {
	if orderBy == "" {
		return nil
	}

	// 許可されたカラム名のみを受け入れる（ホワイトリスト方式）
	allowedColumns := map[string]bool{
		"id":         true,
		"title":      true,
		"content":    true,
		"category":   true,
		"priority":   true,
		"status":     true,
		"created_at": true,
		"updated_at": true,
	}

	// ORDER BY句を解析
	parts := strings.Fields(strings.ToLower(orderBy))
	if len(parts) == 0 || len(parts) > 2 {
		return fmt.Errorf("invalid order by format")
	}

	column := parts[0]
	if !allowedColumns[column] {
		return fmt.Errorf("invalid column for ordering: %s", column)
	}

	if len(parts) == 2 {
		direction := parts[1]
		if direction != "asc" && direction != "desc" {
			return fmt.Errorf("invalid order direction: %s", direction)
		}
	}

	return nil
}

// EscapeForLike escapes special characters in LIKE patterns
func (s *SQLSanitizer) EscapeForLike(pattern string) string {
	// PostgreSQLのLIKE演算子用にエスケープ
	replacer := strings.NewReplacer(
		"\\", "\\\\", // バックスラッシュ
		"%", "\\%",   // パーセント
		"_", "\\_",   // アンダースコア
	)
	return replacer.Replace(pattern)
}
