package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"memo-app/src/domain"
	"memo-app/src/logger"
)

var (
	ErrMemoNotFound    = errors.New("memo not found")
	ErrInvalidTitle    = errors.New("title is required and must be less than 200 characters")
	ErrInvalidContent  = errors.New("content is required")
	ErrInvalidPriority = errors.New("priority must be low, medium, or high")
	ErrInvalidStatus   = errors.New("status must be active or archived")
	ErrInvalidPage     = errors.New("page must be greater than 0")
	ErrInvalidLimit    = errors.New("limit must be between 1 and 100")
)

// CreateMemoRequest represents input for creating a memo
type CreateMemoRequest struct {
	Title    string
	Content  string
	Category string
	Tags     []string
	Priority string
}

// UpdateMemoRequest represents input for updating a memo
type UpdateMemoRequest struct {
	Title    *string
	Content  *string
	Category *string
	Tags     []string
	Priority *string
	Status   *string
}

// MemoUsecase defines the interface for memo business logic
type MemoUsecase interface {
	CreateMemo(ctx context.Context, userID int, req CreateMemoRequest) (*domain.Memo, error)
	GetMemo(ctx context.Context, userID int, id int) (*domain.Memo, error)
	ListMemos(ctx context.Context, userID int, filter domain.MemoFilter) ([]domain.Memo, int, error)
	UpdateMemo(ctx context.Context, userID int, id int, req UpdateMemoRequest) (*domain.Memo, error)
	DeleteMemo(ctx context.Context, userID int, id int) error
	PermanentDeleteMemo(ctx context.Context, userID int, id int) error
	ArchiveMemo(ctx context.Context, userID int, id int) (*domain.Memo, error)
	RestoreMemo(ctx context.Context, userID int, id int) (*domain.Memo, error)
	SearchMemos(ctx context.Context, userID int, query string, filter domain.MemoFilter) ([]domain.Memo, int, error)
}

type memoUsecase struct {
	memoRepo domain.MemoRepository
}

// NewMemoUsecase creates a new memo usecase
func NewMemoUsecase(memoRepo domain.MemoRepository) MemoUsecase {
	return &memoUsecase{
		memoRepo: memoRepo,
	}
}

// CreateMemo creates a new memo for a specific user
func (u *memoUsecase) CreateMemo(ctx context.Context, userID int, req CreateMemoRequest) (*domain.Memo, error) {
	if err := u.validateCreateRequest(req); err != nil {
		return nil, err
	}

	priority := domain.Priority(req.Priority)
	if req.Priority == "" {
		priority = domain.PriorityMedium // デフォルト値
	}

	memo := &domain.Memo{
		UserID:    userID,
		Title:     req.Title,
		Content:   req.Content,
		Category:  req.Category,
		Tags:      u.normalizeTags(req.Tags),
		Priority:  priority,
		Status:    domain.StatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return u.memoRepo.Create(ctx, memo)
}

// GetMemo retrieves a memo by ID for a specific user
func (u *memoUsecase) GetMemo(ctx context.Context, userID int, id int) (*domain.Memo, error) {
	memo, err := u.memoRepo.GetByID(ctx, id, userID)
	if err != nil {
		if strings.Contains(err.Error(), "memo not found") || strings.Contains(err.Error(), "access denied") {
			return nil, ErrMemoNotFound
		}
		return nil, err
	}
	return memo, nil
}

// ListMemos retrieves memos with filtering for a specific user
func (u *memoUsecase) ListMemos(ctx context.Context, userID int, filter domain.MemoFilter) ([]domain.Memo, int, error) {
	// デバッグログ
	logger.Log.WithField("userID", userID).Error("=== USECASE ListMemos called with userID")

	if err := u.validateAndNormalizeFilter(&filter); err != nil {
		return nil, 0, err
	}

	logger.Log.Error("=== USECASE calling repo.List ===")
	return u.memoRepo.List(ctx, userID, filter)
}

// UpdateMemo updates an existing memo for a specific user
func (u *memoUsecase) UpdateMemo(ctx context.Context, userID int, id int, req UpdateMemoRequest) (*domain.Memo, error) {
	if err := u.validateUpdateRequest(req); err != nil {
		return nil, err
	}

	// 既存のメモを取得（ユーザー固有）
	existingMemo, err := u.memoRepo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	// 更新フィールドを適用
	updatedMemo := *existingMemo

	if req.Title != nil {
		updatedMemo.Title = *req.Title
	}
	if req.Content != nil {
		updatedMemo.Content = *req.Content
	}
	if req.Category != nil {
		updatedMemo.Category = *req.Category
	}
	if req.Tags != nil {
		updatedMemo.Tags = u.normalizeTags(req.Tags)
	}
	if req.Priority != nil {
		updatedMemo.Priority = domain.Priority(*req.Priority)
	}
	if req.Status != nil {
		updatedMemo.Status = domain.Status(*req.Status)
	}

	updatedMemo.UpdatedAt = time.Now()

	return u.memoRepo.Update(ctx, id, userID, &updatedMemo)
}

// DeleteMemo handles memo deletion (archives active memos, permanently deletes archived ones) for a specific user
func (u *memoUsecase) DeleteMemo(ctx context.Context, userID int, id int) error {
	return u.memoRepo.Delete(ctx, id, userID)
}

// PermanentDeleteMemo permanently deletes a memo from database for a specific user
func (u *memoUsecase) PermanentDeleteMemo(ctx context.Context, userID int, id int) error {
	// まずメモの現在のstatusを取得
	memo, err := u.memoRepo.GetByID(ctx, id, userID)
	if err != nil {
		if err == ErrMemoNotFound {
			return ErrMemoNotFound
		}
		return err
	}
	if memo.Status != domain.StatusArchived {
		// archived以外はエラー
		return ErrInvalidStatus
	}
	// archivedなら物理削除
	return u.memoRepo.PermanentDelete(ctx, id, userID)
}

// ArchiveMemo archives a memo for a specific user
func (u *memoUsecase) ArchiveMemo(ctx context.Context, userID int, id int) (*domain.Memo, error) {
	memo, err := u.memoRepo.Archive(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	return memo, nil
}

// RestoreMemo restores an archived memo for a specific user
func (u *memoUsecase) RestoreMemo(ctx context.Context, userID int, id int) (*domain.Memo, error) {
	memo, err := u.memoRepo.Restore(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	return memo, nil
}

// SearchMemos searches memos for a specific user
func (u *memoUsecase) SearchMemos(ctx context.Context, userID int, query string, filter domain.MemoFilter) ([]domain.Memo, int, error) {
	if err := u.validateAndNormalizeFilter(&filter); err != nil {
		return nil, 0, err
	}

	return u.memoRepo.Search(ctx, userID, query, filter)
}

// validateCreateRequest validates create memo request
func (u *memoUsecase) validateCreateRequest(req CreateMemoRequest) error {
	if req.Title == "" || len(req.Title) > 200 {
		return ErrInvalidTitle
	}
	if req.Content == "" {
		return ErrInvalidContent
	}
	if req.Priority != "" && !domain.Priority(req.Priority).IsValid() {
		return ErrInvalidPriority
	}
	return nil
}

// validateUpdateRequest validates update memo request
func (u *memoUsecase) validateUpdateRequest(req UpdateMemoRequest) error {
	if req.Title != nil && (*req.Title == "" || len(*req.Title) > 200) {
		return ErrInvalidTitle
	}
	if req.Content != nil && *req.Content == "" {
		return ErrInvalidContent
	}
	if req.Priority != nil && !domain.Priority(*req.Priority).IsValid() {
		return ErrInvalidPriority
	}
	if req.Status != nil && !domain.Status(*req.Status).IsValid() {
		return ErrInvalidStatus
	}
	return nil
}

// validateAndNormalizeFilter validates and normalizes filter
func (u *memoUsecase) validateAndNormalizeFilter(filter *domain.MemoFilter) error {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 {
		filter.Limit = 10
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}

	if filter.Status != "" && !filter.Status.IsValid() {
		return ErrInvalidStatus
	}
	if filter.Priority != "" && !filter.Priority.IsValid() {
		return ErrInvalidPriority
	}

	return nil
}

// normalizeTags normalizes tags by removing empty ones and duplicates
func (u *memoUsecase) normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return []string{}
	}

	seen := make(map[string]bool)
	result := make([]string, 0, len(tags))

	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed != "" && !seen[trimmed] {
			seen[trimmed] = true
			result = append(result, trimmed)
		}
	}

	return result
}
