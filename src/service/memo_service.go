package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"memo-app/src/domain"
	"memo-app/src/models"
	"memo-app/src/usecase"

	"github.com/sirupsen/logrus"
)

// MemoService represents the memo service
type MemoService struct {
	repo   domain.MemoRepository
	logger *logrus.Logger
}

// NewMemoService creates a new memo service
func NewMemoService(repo domain.MemoRepository, logger *logrus.Logger) *MemoService {
	logger.WithField("repo_type", fmt.Sprintf("%T", repo)).Info("MemoService initialized with repository")
	return &MemoService{
		repo:   repo,
		logger: logger,
	}
}

// DeleteMemo deletes a memo
func (s *MemoService) DeleteMemo(ctx context.Context, userID int, id int) error {
	if id <= 0 {
		return fmt.Errorf("invalid memo ID: %d", id)
	}
	return s.repo.Delete(ctx, userID, id)
}

// CreateMemo creates a new memo
func (s *MemoService) CreateMemo(ctx context.Context, userID int, req usecase.CreateMemoRequest) (*domain.Memo, error) {
	memo := &domain.Memo{
		UserID:   userID,
		Title:    req.Title,
		Content:  req.Content,
		Category: req.Category,
		Tags:     req.Tags,
		Priority: domain.Priority(req.Priority),
		Status:   domain.StatusActive,
	}
	return s.repo.Create(ctx, memo)
}

// GetMemo retrieves a memo by ID
func (s *MemoService) GetMemo(ctx context.Context, userID int, id int) (*domain.Memo, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid memo ID: %d", id)
	}
	memo, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	return memo, nil
}

// ListMemos retrieves memos with filtering
func (s *MemoService) ListMemos(ctx context.Context, userID int, filter domain.MemoFilter) ([]domain.Memo, int, error) {
	memos, total, err := s.repo.List(ctx, userID, filter)
	if err != nil {
		return nil, 0, err
	}
	return memos, total, nil
}

// DeprecatedUpdateMemo is deprecated and should be removed or refactored
// func (s *MemoService) UpdateMemo(ctx context.Context, id int, userID int, memo *domain.Memo) (*domain.Memo, error) {
// 	// This method is deprecated and replaced by the new UpdateMemo method below.
// 	return nil, fmt.Errorf("DeprecatedUpdateMemo is deprecated, use the new UpdateMemo method")
// }

// ArchiveMemo archives a memo (sets status to archived)
func (s *MemoService) ArchiveMemo(ctx context.Context, userID int, id int) (*domain.Memo, error) {
	return s.repo.Archive(ctx, userID, id)
}

// RestoreMemo restores an archived memo (sets status to active)
func (s *MemoService) RestoreMemo(ctx context.Context, userID int, id int) (*domain.Memo, error) {
	return s.repo.Restore(ctx, userID, id)
}

// UpdateMemo updates a memo
func (s *MemoService) UpdateMemo(ctx context.Context, userID int, id int, req usecase.UpdateMemoRequest) (*domain.Memo, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid memo ID: %d", id)
	}
	existingMemo, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if req.Title != nil {
		existingMemo.Title = *req.Title
	}
	if req.Content != nil {
		existingMemo.Content = *req.Content
	}
	if req.Category != nil {
		existingMemo.Category = *req.Category
	}
	if len(req.Tags) > 0 {
		existingMemo.Tags = req.Tags
	}
	if req.Priority != nil {
		existingMemo.Priority = domain.Priority(*req.Priority)
	}
	if req.Status != nil {
		existingMemo.Status = domain.Status(*req.Status)
	}
	existingMemo.UpdatedAt = time.Now()
	return s.repo.Update(ctx, id, userID, existingMemo)
}

// SearchMemos searches memos by content
func (s *MemoService) SearchMemos(ctx context.Context, userID int, query string, filter domain.MemoFilter) ([]domain.Memo, int, error) {
	filter.Search = strings.TrimSpace(query)
	memos, total, err := s.repo.List(ctx, userID, filter)
	if err != nil {
		return nil, 0, err
	}
	return memos, total, nil
}

// validateCreateRequest validates the create memo request
func (s *MemoService) validateCreateRequest(req *models.CreateMemoRequest) error {
	if strings.TrimSpace(req.Title) == "" {
		return fmt.Errorf("title is required")
	}

	if strings.TrimSpace(req.Content) == "" {
		return fmt.Errorf("content is required")
	}

	if len(req.Title) > 200 {
		return fmt.Errorf("title must be at most 200 characters")
	}

	if req.Category != "" && len(req.Category) > 50 {
		return fmt.Errorf("category must be at most 50 characters")
	}

	if req.Priority != "" && !isValidPriority(req.Priority) {
		return fmt.Errorf("priority must be one of: low, medium, high")
	}

	return nil
}

// validateUpdateRequest validates the update memo request
func (s *MemoService) validateUpdateRequest(req *models.UpdateMemoRequest) error {
	if req.Title != nil {
		if strings.TrimSpace(*req.Title) == "" {
			return fmt.Errorf("title cannot be empty")
		}
		if len(*req.Title) > 200 {
			return fmt.Errorf("title must be at most 200 characters")
		}
	}

	if req.Content != nil && strings.TrimSpace(*req.Content) == "" {
		return fmt.Errorf("content cannot be empty")
	}

	if req.Category != nil && len(*req.Category) > 50 {
		return fmt.Errorf("category must be at most 50 characters")
	}

	if req.Priority != nil && !isValidPriority(*req.Priority) {
		return fmt.Errorf("priority must be one of: low, medium, high")
	}

	if req.Status != nil && !isValidStatus(*req.Status) {
		return fmt.Errorf("status must be one of: active, archived")
	}

	return nil
}

// validateAndNormalizeFilter validates and normalizes the filter
func (s *MemoService) validateAndNormalizeFilter(filter *models.MemoFilter) error {
	if filter.Page <= 0 {
		filter.Page = 1
	}

	if filter.Limit <= 0 {
		filter.Limit = 10
	}

	if filter.Limit > 100 {
		filter.Limit = 100
	}

	if filter.Priority != "" && !isValidPriority(filter.Priority) {
		return fmt.Errorf("priority must be one of: low, medium, high")
	}

	if filter.Status != "" && !isValidStatus(filter.Status) {
		return fmt.Errorf("status must be one of: active, archived")
	}

	// 検索クエリの正規化
	filter.Search = strings.TrimSpace(filter.Search)
	filter.Category = strings.TrimSpace(filter.Category)
	filter.Tags = strings.TrimSpace(filter.Tags)

	return nil
}

// normalizeTags normalizes tags (removes empty and duplicate tags)
func (s *MemoService) normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return tags
	}

	seen := make(map[string]bool)
	var normalized []string

	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" && !seen[tag] {
			normalized = append(normalized, tag)
			seen[tag] = true
		}
	}

	return normalized
}

// isValidPriority checks if priority is valid
func isValidPriority(priority string) bool {
	switch priority {
	case "low", "medium", "high":
		return true
	default:
		return false
	}
}

// isValidStatus checks if status is valid
func isValidStatus(status string) bool {
	switch status {
	case "active", "archived":
		return true
	default:
		return false
	}
}

// PermanentDeleteMemo permanently deletes a memo
func (s *MemoService) PermanentDeleteMemo(ctx context.Context, userID int, id int) error {
	if id <= 0 {
		return fmt.Errorf("invalid memo ID: %d", id)
	}
	return s.repo.PermanentDelete(ctx, userID, id)
}
