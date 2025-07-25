package models

import (
	"time"
)

// Memo represents a memo item
type Memo struct {
	ID          int        `json:"id" db:"id"`
	UserID      int        `json:"user_id" db:"user_id"`
	Title       string     `json:"title" db:"title" binding:"required,max=200"`
	Content     string     `json:"content" db:"content" binding:"required"`
	Category    string     `json:"category" db:"category" binding:"max=50"`
	Tags        string     `json:"tags" db:"tags"` // JSON文字列として保存
	Priority    string     `json:"priority" db:"priority" binding:"oneof=low medium high"`
	Status      string     `json:"status" db:"status" binding:"oneof=active archived"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`
}

// CreateMemoRequest represents the request payload for creating a memo
type CreateMemoRequest struct {
	Title    string   `json:"title" binding:"required,max=200"`
	Content  string   `json:"content" binding:"required"`
	Category string   `json:"category" binding:"max=50"`
	Tags     []string `json:"tags"`
	Priority string   `json:"priority" binding:"oneof=low medium high"`
}

// UpdateMemoRequest represents the request payload for updating a memo
type UpdateMemoRequest struct {
	Title    *string  `json:"title,omitempty" binding:"omitempty,max=200"`
	Content  *string  `json:"content,omitempty"`
	Category *string  `json:"category,omitempty" binding:"omitempty,max=50"`
	Tags     []string `json:"tags,omitempty"`
	Priority *string  `json:"priority,omitempty" binding:"omitempty,oneof=low medium high"`
	Status   *string  `json:"status,omitempty" binding:"omitempty,oneof=active archived"`
}

// MemoListResponse represents the response for memo list
type MemoListResponse struct {
	Memos      []Memo `json:"memos"`
	Total      int    `json:"total"`
	Page       int    `json:"page"`
	Limit      int    `json:"limit"`
	TotalPages int    `json:"total_pages"`
}

// MemoFilter represents filter options for memo queries
type MemoFilter struct {
	Category string `form:"category"`
	Status   string `form:"status" binding:"omitempty,oneof=active archived"`
	Priority string `form:"priority" binding:"omitempty,oneof=low medium high"`
	Search   string `form:"search"`
	Tags     string `form:"tags"`
	Page     int    `form:"page,default=1" binding:"min=1"`
	Limit    int    `form:"limit,default=10" binding:"min=1,max=100"`
}
