package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"memo-app/src/database"
	"memo-app/src/models"
	"memo-app/src/repository"
	"memo-app/src/security"

	"github.com/sirupsen/logrus"
)

// MemoRepository implements repository.MemoRepositoryInterface
type MemoRepository struct {
	db           *database.DB
	logger       *logrus.Logger
	sqlSanitizer *security.SQLSanitizer
}

// NewMemoRepository creates a new memo repository
func NewMemoRepository(db *database.DB, logger *logrus.Logger) repository.MemoRepositoryInterface {
	return &MemoRepository{
		db:           db,
		logger:       logger,
		sqlSanitizer: security.NewSQLSanitizer(),
	}
}

// Create creates a new memo
func (r *MemoRepository) Create(ctx context.Context, userID int, req *models.CreateMemoRequest) (*models.Memo, error) {
	// タグを JSON 文字列に変換
	tagsJSON, err := json.Marshal(req.Tags)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tags: %w", err)
	}

	now := time.Now()

	query := `
		INSERT INTO memos (user_id, title, content, category, tags, priority, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`

	var id int
	err = r.db.QueryRowContext(ctx, query,
		userID, req.Title, req.Content, req.Category, string(tagsJSON),
		req.Priority, "active", now, now,
	).Scan(&id)

	if err != nil {
		r.logger.WithError(err).Error("メモの作成に失敗")
		return nil, fmt.Errorf("failed to create memo: %w", err)
	}

	// 作成されたメモを返す
	newMemo := &models.Memo{
		ID:        id,
		UserID:    userID,
		Title:     req.Title,
		Content:   req.Content,
		Category:  req.Category,
		Tags:      string(tagsJSON),
		Priority:  req.Priority,
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}

	r.logger.WithField("memo_id", id).WithField("user_id", userID).WithField("returned_memo_user_id", newMemo.UserID).Info("メモを作成しました")
	return newMemo, nil
}

// GetByID retrieves a memo by ID for a specific user
func (r *MemoRepository) GetByID(ctx context.Context, id int, userID int) (*models.Memo, error) {
	query := `
		SELECT id, user_id, title, content, category, tags, priority, status, created_at, updated_at, completed_at
		FROM memos WHERE id = $1 AND user_id = $2`

	var memo models.Memo
	var tagsStr sql.NullString
	var completedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id, userID).Scan(
		&memo.ID, &memo.UserID, &memo.Title, &memo.Content, &memo.Category,
		&tagsStr, &memo.Priority, &memo.Status,
		&memo.CreatedAt, &memo.UpdatedAt, &completedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("memo not found")
		}
		r.logger.WithError(err).Error("Failed to get memo by ID")
		return nil, fmt.Errorf("failed to get memo: %w", err)
	}

	if tagsStr.Valid {
		memo.Tags = tagsStr.String
	}
	if completedAt.Valid {
		memo.CompletedAt = &completedAt.Time
	}

	return &memo, nil
}

// List retrieves memos for a user with filtering
func (r *MemoRepository) List(ctx context.Context, userID int, filter *models.MemoFilter) (*models.MemoListResponse, error) {
	query := `
		SELECT id, user_id, title, content, category, tags, priority, status, created_at, updated_at, completed_at
		FROM memos 
		WHERE user_id = $1`

	args := []interface{}{userID}
	argCount := 1

	// フィルタリング条件を追加
	if filter != nil {
		if filter.Status != "" {
			argCount++
			query += fmt.Sprintf(" AND status = $%d", argCount)
			args = append(args, filter.Status)
		}
		if filter.Category != "" {
			argCount++
			query += fmt.Sprintf(" AND category = $%d", argCount)
			args = append(args, filter.Category)
		}
		if filter.Priority != "" {
			argCount++
			query += fmt.Sprintf(" AND priority = $%d", argCount)
			args = append(args, filter.Priority)
		}
	}

	// 並び順
	query += " ORDER BY created_at DESC"

	// リミットとオフセット
	if filter != nil && filter.Limit > 0 {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)

		if filter.Page > 1 {
			offset := (filter.Page - 1) * filter.Limit
			argCount++
			query += fmt.Sprintf(" OFFSET $%d", argCount)
			args = append(args, offset)
		}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.WithError(err).Error("Failed to list memos")
		return nil, fmt.Errorf("failed to list memos: %w", err)
	}
	defer rows.Close()

	var memos []models.Memo
	for rows.Next() {
		var memo models.Memo
		var tagsStr sql.NullString
		var completedAt sql.NullTime

		err := rows.Scan(
			&memo.ID, &memo.UserID, &memo.Title, &memo.Content, &memo.Category,
			&tagsStr, &memo.Priority, &memo.Status,
			&memo.CreatedAt, &memo.UpdatedAt, &completedAt,
		)
		if err != nil {
			r.logger.WithError(err).Error("Failed to scan memo")
			return nil, fmt.Errorf("failed to scan memo: %w", err)
		}

		if tagsStr.Valid {
			memo.Tags = tagsStr.String
		}
		if completedAt.Valid {
			memo.CompletedAt = &completedAt.Time
		}

		memos = append(memos, memo)
	}

	// 総数を取得
	countQuery := `SELECT COUNT(*) FROM memos WHERE user_id = $1`
	countArgs := []interface{}{userID}

	if filter != nil {
		if filter.Status != "" {
			countQuery += " AND status = $2"
			countArgs = append(countArgs, filter.Status)
		}
		if filter.Category != "" {
			countQuery += " AND category = $" + fmt.Sprintf("%d", len(countArgs)+1)
			countArgs = append(countArgs, filter.Category)
		}
		if filter.Priority != "" {
			countQuery += " AND priority = $" + fmt.Sprintf("%d", len(countArgs)+1)
			countArgs = append(countArgs, filter.Priority)
		}
	}

	var total int
	err = r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		r.logger.WithError(err).Error("Failed to count memos")
		return nil, fmt.Errorf("failed to count memos: %w", err)
	}

	return &models.MemoListResponse{
		Memos: memos,
		Total: total,
	}, nil
}

// Update updates a memo
func (r *MemoRepository) Update(ctx context.Context, userID int, id int, req *models.UpdateMemoRequest) (*models.Memo, error) {
	// 既存メモ取得
	existing, err := r.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	// 更新値の決定
	title := existing.Title
	if req.Title != nil {
		title = *req.Title
	}
	content := existing.Content
	if req.Content != nil {
		content = *req.Content
	}
	category := existing.Category
	if req.Category != nil {
		category = *req.Category
	}
	tagsJSON := existing.Tags
	if req.Tags != nil {
		marshaled, err := json.Marshal(req.Tags)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal tags: %w", err)
		}
		tagsJSON = string(marshaled)
	}
	priority := existing.Priority
	if req.Priority != nil {
		priority = *req.Priority
	}
	status := existing.Status
	if req.Status != nil {
		status = *req.Status
	}

	now := time.Now()
	query := `
		UPDATE memos 
		SET title = $1, content = $2, category = $3, tags = $4, priority = $5, status = $6, updated_at = $7
		WHERE id = $8 AND user_id = $9
		RETURNING id, user_id, title, content, category, tags, priority, status, created_at, updated_at, completed_at`

	var memo models.Memo
	var tagsStr sql.NullString
	var completedAt sql.NullTime

	err = r.db.QueryRowContext(ctx, query,
		title, content, category, tagsJSON, priority, status, now, id, userID,
	).Scan(
		&memo.ID, &memo.UserID, &memo.Title, &memo.Content, &memo.Category,
		&tagsStr, &memo.Priority, &memo.Status,
		&memo.CreatedAt, &memo.UpdatedAt, &completedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("memo not found")
		}
		r.logger.WithError(err).Error("Failed to update memo")
		return nil, fmt.Errorf("failed to update memo: %w", err)
	}

	if tagsStr.Valid {
		memo.Tags = tagsStr.String
	}
	if completedAt.Valid {
		memo.CompletedAt = &completedAt.Time
	}

	return &memo, nil
}

// Delete soft deletes a memo
func (r *MemoRepository) Delete(ctx context.Context, userID int, id int) error {
	query := `UPDATE memos SET status = 'deleted', updated_at = $1 WHERE id = $2 AND user_id = $3`

	result, err := r.db.ExecContext(ctx, query, time.Now(), id, userID)
	if err != nil {
		r.logger.WithError(err).Error("Failed to delete memo")
		return fmt.Errorf("failed to delete memo: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("memo not found")
	}

	return nil
}

// PermanentDelete permanently deletes a memo
func (r *MemoRepository) PermanentDelete(ctx context.Context, userID int, id int) error {
	query := `DELETE FROM memos WHERE id = $1 AND user_id = $2`

	result, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		r.logger.WithError(err).Error("Failed to permanently delete memo")
		return fmt.Errorf("failed to permanently delete memo: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("memo not found")
	}

	return nil
}

// Archive archives a memo
func (r *MemoRepository) Archive(ctx context.Context, userID int, id int) error {
	now := time.Now()
	query := `UPDATE memos SET status = 'archived', completed_at = $1, updated_at = $2 WHERE id = $3 AND user_id = $4`

	result, err := r.db.ExecContext(ctx, query, now, now, id, userID)
	if err != nil {
		r.logger.WithError(err).Error("Failed to archive memo")
		return fmt.Errorf("failed to archive memo: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("memo not found")
	}

	return nil
}

// Restore restores an archived memo
func (r *MemoRepository) Restore(ctx context.Context, userID int, id int) error {
	query := `UPDATE memos SET status = 'active', completed_at = NULL, updated_at = $1 WHERE id = $2 AND user_id = $3`

	result, err := r.db.ExecContext(ctx, query, time.Now(), id, userID)
	if err != nil {
		r.logger.WithError(err).Error("Failed to restore memo")
		return fmt.Errorf("failed to restore memo: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("memo not found")
	}

	return nil
}

// Search searches for memos
func (r *MemoRepository) Search(ctx context.Context, userID int, query string, filter models.MemoFilter) ([]models.Memo, int, error) {
	// 簡単な検索実装（タイトルとコンテンツでの検索）
	searchQuery := `
		SELECT id, user_id, title, content, category, tags, priority, status, created_at, updated_at, completed_at
		FROM memos 
		WHERE user_id = $1 AND (title ILIKE $2 OR content ILIKE $2)`

	args := []interface{}{userID, "%" + query + "%"}
	argCount := 2

	// フィルタリング条件を追加
	if filter.Status != "" {
		argCount++
		searchQuery += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, filter.Status)
	}
	if filter.Category != "" {
		argCount++
		searchQuery += fmt.Sprintf(" AND category = $%d", argCount)
		args = append(args, filter.Category)
	}
	if filter.Priority != "" {
		argCount++
		searchQuery += fmt.Sprintf(" AND priority = $%d", argCount)
		args = append(args, filter.Priority)
	}

	searchQuery += " ORDER BY created_at DESC"

	// リミット
	if filter.Limit > 0 {
		argCount++
		searchQuery += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)

		if filter.Page > 1 {
			offset := (filter.Page - 1) * filter.Limit
			argCount++
			searchQuery += fmt.Sprintf(" OFFSET $%d", argCount)
			args = append(args, offset)
		}
	}

	rows, err := r.db.QueryContext(ctx, searchQuery, args...)
	if err != nil {
		r.logger.WithError(err).Error("Failed to search memos")
		return nil, 0, fmt.Errorf("failed to search memos: %w", err)
	}
	defer rows.Close()

	var memos []models.Memo
	for rows.Next() {
		var memo models.Memo
		var tagsStr sql.NullString
		var completedAt sql.NullTime

		err := rows.Scan(
			&memo.ID, &memo.UserID, &memo.Title, &memo.Content, &memo.Category,
			&tagsStr, &memo.Priority, &memo.Status,
			&memo.CreatedAt, &memo.UpdatedAt, &completedAt,
		)
		if err != nil {
			r.logger.WithError(err).Error("Failed to scan memo")
			return nil, 0, fmt.Errorf("failed to scan memo: %w", err)
		}

		if tagsStr.Valid {
			memo.Tags = tagsStr.String
		}
		if completedAt.Valid {
			memo.CompletedAt = &completedAt.Time
		}

		memos = append(memos, memo)
	}

	// 総数を取得
	countQuery := `SELECT COUNT(*) FROM memos WHERE user_id = $1 AND (title ILIKE $2 OR content ILIKE $2)`
	countArgs := []interface{}{userID, "%" + query + "%"}

	if filter.Status != "" {
		countQuery += " AND status = $3"
		countArgs = append(countArgs, filter.Status)
	}
	if filter.Category != "" {
		countQuery += " AND category = $" + fmt.Sprintf("%d", len(countArgs)+1)
		countArgs = append(countArgs, filter.Category)
	}
	if filter.Priority != "" {
		countQuery += " AND priority = $" + fmt.Sprintf("%d", len(countArgs)+1)
		countArgs = append(countArgs, filter.Priority)
	}

	var total int
	err = r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		r.logger.WithError(err).Error("Failed to count search results")
		return nil, 0, fmt.Errorf("failed to count search results: %w", err)
	}

	return memos, total, nil
}
