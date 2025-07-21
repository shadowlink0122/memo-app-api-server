package service

import (
	"context"

	"memo-app/src/models"
)

// MemoServiceInterface defines the interface for memo service
type MemoServiceInterface interface {
	CreateMemo(ctx context.Context, req *models.CreateMemoRequest) (*models.Memo, error)
	GetMemo(ctx context.Context, id int) (*models.Memo, error)
	ListMemos(ctx context.Context, filter *models.MemoFilter) (*models.MemoListResponse, error)
	UpdateMemo(ctx context.Context, id int, req *models.UpdateMemoRequest) (*models.Memo, error)
	DeleteMemo(ctx context.Context, id int) error
	ArchiveMemo(ctx context.Context, id int) (*models.Memo, error)
	RestoreMemo(ctx context.Context, id int) (*models.Memo, error)
	SearchMemos(ctx context.Context, query string, page, limit int) (*models.MemoListResponse, error)
}
