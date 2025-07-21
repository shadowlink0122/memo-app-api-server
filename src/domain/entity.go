package domain

import (
	"time"
)

// Memo represents a memo domain entity
type Memo struct {
	ID          int
	Title       string
	Content     string
	Category    string
	Tags        []string
	Priority    Priority
	Status      Status
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt *time.Time
}

// Priority represents memo priority levels
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

// Status represents memo status
type Status string

const (
	StatusActive   Status = "active"
	StatusArchived Status = "archived"
)

// MemoFilter represents filter criteria for memo queries
type MemoFilter struct {
	Category string
	Status   Status
	Priority Priority
	Search   string
	Tags     []string
	Page     int
	Limit    int
}

// IsValid validates if the priority is valid
func (p Priority) IsValid() bool {
	switch p {
	case PriorityLow, PriorityMedium, PriorityHigh:
		return true
	default:
		return false
	}
}

// IsValid validates if the status is valid
func (s Status) IsValid() bool {
	switch s {
	case StatusActive, StatusArchived:
		return true
	default:
		return false
	}
}

// String returns string representation of Priority
func (p Priority) String() string {
	return string(p)
}

// String returns string representation of Status
func (s Status) String() string {
	return string(s)
}
