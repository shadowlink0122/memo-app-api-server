package repository

import (
	"context"

	"memo-app/src/models"
)

// MemoRepositoryInterface defines the interface for memo repository
type MemoRepositoryInterface interface {
	Create(ctx context.Context, req *models.CreateMemoRequest) (*models.Memo, error)
	GetByID(ctx context.Context, id int) (*models.Memo, error)
	List(ctx context.Context, filter *models.MemoFilter) (*models.MemoListResponse, error)
	Update(ctx context.Context, id int, req *models.UpdateMemoRequest) (*models.Memo, error)
	Delete(ctx context.Context, id int) error
	PermanentDelete(ctx context.Context, id int) error
	Archive(ctx context.Context, id int) error
	Restore(ctx context.Context, id int) error
	Search(ctx context.Context, query string, filter models.MemoFilter) ([]models.Memo, int, error)
}
