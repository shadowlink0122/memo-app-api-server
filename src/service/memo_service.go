package service

import (
	"context"
	"fmt"
	"strings"

	"memo-app/src/models"
	"memo-app/src/repository"

	"github.com/sirupsen/logrus"
)

// MemoService represents the memo service
type MemoService struct {
	repo   repository.MemoRepositoryInterface
	logger *logrus.Logger
}

// NewMemoService creates a new memo service
func NewMemoService(repo repository.MemoRepositoryInterface, logger *logrus.Logger) *MemoService {
	logger.WithField("repo_type", fmt.Sprintf("%T", repo)).Info("MemoService initialized with repository")
	return &MemoService{
		repo:   repo,
		logger: logger,
	}
}

// CreateMemo creates a new memo
func (s *MemoService) CreateMemo(ctx context.Context, userID int, req *models.CreateMemoRequest) (*models.Memo, error) {
	// バリデーション
	if err := s.validateCreateRequest(req); err != nil {
		return nil, err
	}

	// デフォルト値の設定
	if req.Priority == "" {
		req.Priority = "medium"
	}

	// タグの正規化
	req.Tags = s.normalizeTags(req.Tags)

	return s.repo.Create(ctx, userID, req)
}

// GetMemo retrieves a memo by ID
func (s *MemoService) GetMemo(ctx context.Context, userID int, id int) (*models.Memo, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid memo ID: %d", id)
	}

	return s.repo.GetByID(ctx, id, userID)
}

// ListMemos retrieves memos with filtering
func (s *MemoService) ListMemos(ctx context.Context, userID int, filter *models.MemoFilter) (*models.MemoListResponse, error) {
	// フィルターのバリデーションとデフォルト値設定
	if err := s.validateAndNormalizeFilter(filter); err != nil {
		return nil, err
	}

	return s.repo.List(ctx, userID, filter)
}

// UpdateMemo updates a memo
func (s *MemoService) UpdateMemo(ctx context.Context, userID int, id int, req *models.UpdateMemoRequest) (*models.Memo, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid memo ID: %d", id)
	}

	// バリデーション
	if err := s.validateUpdateRequest(req); err != nil {
		return nil, err
	}

	// タグの正規化
	if req.Tags != nil {
		normalizedTags := s.normalizeTags(req.Tags)
		req.Tags = normalizedTags
	}

	return s.repo.Update(ctx, userID, id, req)
}

// DeleteMemo deletes a memo
func (s *MemoService) DeleteMemo(ctx context.Context, userID int, id int) error {
	if id <= 0 {
		return fmt.Errorf("invalid memo ID: %d", id)
	}

	return s.repo.Delete(ctx, userID, id)
}

// ArchiveMemo archives a memo (sets status to archived)
func (s *MemoService) ArchiveMemo(ctx context.Context, userID int, id int) (*models.Memo, error) {
	// statusのみを安全に更新する
	err := s.repo.Archive(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id, userID)
}

// RestoreMemo restores an archived memo (sets status to active)
func (s *MemoService) RestoreMemo(ctx context.Context, userID int, id int) (*models.Memo, error) {
	status := "active"
	req := &models.UpdateMemoRequest{
		Status: &status,
	}

	return s.UpdateMemo(ctx, userID, id, req)
}

// SearchMemos searches memos by content
func (s *MemoService) SearchMemos(ctx context.Context, userID int, query string, page, limit int) (*models.MemoListResponse, error) {
	filter := &models.MemoFilter{
		Search: strings.TrimSpace(query),
		Page:   page,
		Limit:  limit,
	}

	if err := s.validateAndNormalizeFilter(filter); err != nil {
		return nil, err
	}

	return s.repo.List(ctx, userID, filter)
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
