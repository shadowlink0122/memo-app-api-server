package repository

import (
	"context"

	"memo-app/src/models"
)

// MemoRepositoryInterface defines the interface for memo repository
type MemoRepositoryInterface interface {
	Create(ctx context.Context, userID int, req *models.CreateMemoRequest) (*models.Memo, error)
	GetByID(ctx context.Context, id int, userID int) (*models.Memo, error)
	List(ctx context.Context, userID int, filter *models.MemoFilter) (*models.MemoListResponse, error)
	Update(ctx context.Context, userID int, id int, req *models.UpdateMemoRequest) (*models.Memo, error)
	Delete(ctx context.Context, userID int, id int) error
	PermanentDelete(ctx context.Context, userID int, id int) error
	Archive(ctx context.Context, userID int, id int) error
	Restore(ctx context.Context, userID int, id int) error
	Search(ctx context.Context, userID int, query string, filter models.MemoFilter) ([]models.Memo, int, error)
}
