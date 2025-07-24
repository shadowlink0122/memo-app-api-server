package domain

import "context"

// MemoRepository defines the interface for memo data operations
type MemoRepository interface {
	Create(ctx context.Context, memo *Memo) (*Memo, error)
	GetByID(ctx context.Context, id int) (*Memo, error)
	List(ctx context.Context, filter MemoFilter) ([]Memo, int, error)
	Update(ctx context.Context, id int, memo *Memo) (*Memo, error)
	Delete(ctx context.Context, id int) error
	PermanentDelete(ctx context.Context, id int) error
	Archive(ctx context.Context, id int) error
	Restore(ctx context.Context, id int) error
	Search(ctx context.Context, query string, filter MemoFilter) ([]Memo, int, error)
}
