package handler

import (
	"time"
)

// CreateMemoRequestDTO represents HTTP request for creating a memo
type CreateMemoRequestDTO struct {
	Title    string   `json:"title" binding:"required,max=200,min=1" validate:"required,max=200,min=1,safe_text,no_sql_injection"`
	Content  string   `json:"content" binding:"required" validate:"required,min=1,safe_text,no_sql_injection"`
	Category string   `json:"category" binding:"max=50" validate:"omitempty,max=50,safe_category"`
	Tags     []string `json:"tags" validate:"omitempty,dive,max=30,safe_tag"`
	Priority string   `json:"priority" binding:"omitempty,oneof=low medium high" validate:"omitempty,oneof=low medium high"`
}

// UpdateMemoRequestDTO represents HTTP request for updating a memo
type UpdateMemoRequestDTO struct {
	Title    *string  `json:"title,omitempty" binding:"omitempty,max=200" validate:"omitempty,max=200,min=1,safe_text,no_sql_injection"`
	Content  *string  `json:"content,omitempty" validate:"omitempty,min=1,safe_text,no_sql_injection"`
	Category *string  `json:"category,omitempty" binding:"omitempty,max=50" validate:"omitempty,max=50,safe_category"`
	Tags     []string `json:"tags,omitempty" validate:"omitempty,dive,max=30,safe_tag"`
	Priority *string  `json:"priority,omitempty" binding:"omitempty,oneof=low medium high" validate:"omitempty,oneof=low medium high"`
	Status   *string  `json:"status,omitempty" binding:"omitempty,oneof=active archived" validate:"omitempty,oneof=active archived"`
}

// MemoResponseDTO represents HTTP response for a memo
type MemoResponseDTO struct {
	ID          int        `json:"id"`
	Title       string     `json:"title"`
	Content     string     `json:"content"`
	Category    string     `json:"category"`
	Tags        []string   `json:"tags"`
	Priority    string     `json:"priority"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// MemoListResponseDTO represents HTTP response for memo list
type MemoListResponseDTO struct {
	Memos      []MemoResponseDTO `json:"memos"`
	Total      int               `json:"total"`
	Page       int               `json:"page"`
	Limit      int               `json:"limit"`
	TotalPages int               `json:"total_pages"`
}

// MemoFilterDTO represents HTTP query parameters for filtering memos
type MemoFilterDTO struct {
	Category string `form:"category" validate:"omitempty,max=50,safe_category"`
	Status   string `form:"status" binding:"omitempty,oneof=active archived" validate:"omitempty,oneof=active archived"`
	Priority string `form:"priority" binding:"omitempty,oneof=low medium high" validate:"omitempty,oneof=low medium high"`
	Search   string `form:"search" validate:"omitempty,max=200,safe_text,no_sql_injection"`
	Tags     string `form:"tags" validate:"omitempty,max=200"`
	Page     int    `form:"page,default=1" binding:"min=1" validate:"min=1,max=1000"`
	Limit    int    `form:"limit,default=10" binding:"min=1,max=100" validate:"min=1,max=100"`
}

// ErrorResponseDTO represents HTTP error response
type ErrorResponseDTO struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
