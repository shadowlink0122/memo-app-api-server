package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"memo-app/src/database"
	"memo-app/src/domain"

	"memo-app/src/usecase"

	"github.com/sirupsen/logrus"
)

// import文の後
type MemoRepository struct {
	db     *database.DB
	logger *logrus.Logger
}

func NewMemoRepository(db *database.DB, logger *logrus.Logger) *MemoRepository {
	return &MemoRepository{
		db:     db,
		logger: logger,
	}
}

// ...既存のコード...

// Create creates a new memo (domain interface)
func (r *MemoRepository) Create(ctx context.Context, memo *domain.Memo) (*domain.Memo, error) {
	tagsJSON, err := json.Marshal(memo.Tags)
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
		memo.UserID, memo.Title, memo.Content, memo.Category, string(tagsJSON),
		memo.Priority, memo.Status, now, now,
	).Scan(&id)
	if err != nil {
		r.logger.WithError(err).Error("メモの作成に失敗")
		return nil, fmt.Errorf("failed to create memo: %w", err)
	}
	memo.ID = id
	memo.CreatedAt = now
	memo.UpdatedAt = now
	r.logger.WithField("memo_id", id).WithField("user_id", memo.UserID).Info("メモを作成しました")
	return memo, nil
}

// GetByID retrieves a memo by ID for a specific user
func (r *MemoRepository) GetByID(ctx context.Context, id int, userID int) (*domain.Memo, error) {
	query := `
		SELECT id, user_id, title, content, category, tags, priority, status, created_at, updated_at, completed_at
		FROM memos WHERE id = $1 AND user_id = $2`
	var memo domain.Memo
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
		// domain.Memo.Tags は []string 型
		var tags []string
		_ = json.Unmarshal([]byte(tagsStr.String), &tags)
		memo.Tags = tags
	}
	if completedAt.Valid {
		memo.CompletedAt = &completedAt.Time
	}
	return &memo, nil
}

// List retrieves memos for a user with filtering (domain interface)
func (r *MemoRepository) List(ctx context.Context, userID int, filter domain.MemoFilter) ([]domain.Memo, int, error) {
	query := `
		SELECT id, user_id, title, content, category, tags, priority, status, created_at, updated_at, completed_at
		FROM memos 
		WHERE user_id = $1`
	args := []interface{}{userID}
	argCount := 1
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
	query += " ORDER BY created_at DESC"
	if filter.Limit > 0 {
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
		return nil, 0, fmt.Errorf("failed to list memos: %w", err)
	}
	defer rows.Close()
	var memos []domain.Memo
	for rows.Next() {
		var memo domain.Memo
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
			var tags []string
			_ = json.Unmarshal([]byte(tagsStr.String), &tags)
			memo.Tags = tags
		}
		if completedAt.Valid {
			memo.CompletedAt = &completedAt.Time
		}
		memos = append(memos, memo)
	}
	// 総数を取得
	countQuery := `SELECT COUNT(*) FROM memos WHERE user_id = $1`
	countArgs := []interface{}{userID}
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
	var total int
	err = r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		r.logger.WithError(err).Error("Failed to count memos")
		return nil, 0, fmt.Errorf("failed to count memos: %w", err)
	}
	return memos, total, nil
}

// Update updates a memo (domain interface)
func (r *MemoRepository) Update(ctx context.Context, id int, userID int, memo *domain.Memo) (*domain.Memo, error) {
	// 既存メモ取得
	existing, err := r.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	// 更新値の決定
	title := existing.Title
	if memo.Title != "" {
		title = memo.Title
	}
	content := existing.Content
	if memo.Content != "" {
		content = memo.Content
	}
	category := existing.Category
	if memo.Category != "" {
		category = memo.Category
	}
	tagsJSON, _ := json.Marshal(existing.Tags)
	if len(memo.Tags) > 0 {
		tagsJSON, _ = json.Marshal(memo.Tags)
	}
	priority := existing.Priority
	if memo.Priority != "" {
		priority = memo.Priority
	}
	status := existing.Status
	// statusは必ず"active"か"archived"のみセット可能
	if string(memo.Status) == string(domain.StatusActive) {
		status = domain.StatusActive
	} else if string(memo.Status) == string(domain.StatusArchived) {
		status = domain.StatusArchived
	}
	now := time.Now()
	query := `
		UPDATE memos 
		SET title = $1, content = $2, category = $3, tags = $4, priority = $5, status = $6, updated_at = $7
		WHERE id = $8 AND user_id = $9
		RETURNING id, user_id, title, content, category, tags, priority, status, created_at, updated_at, completed_at`
	var updated domain.Memo
	var tagsStr sql.NullString
	var completedAt sql.NullTime
	err = r.db.QueryRowContext(ctx, query,
		title, content, category, string(tagsJSON), priority, status, now, id, userID,
	).Scan(
		&updated.ID, &updated.UserID, &updated.Title, &updated.Content, &updated.Category,
		&tagsStr, &updated.Priority, &updated.Status,
		&updated.CreatedAt, &updated.UpdatedAt, &completedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("memo not found")
		}
		r.logger.WithError(err).Error("Failed to update memo")
		return nil, fmt.Errorf("failed to update memo: %w", err)
	}
	if tagsStr.Valid {
		var tags []string
		_ = json.Unmarshal([]byte(tagsStr.String), &tags)
		updated.Tags = tags
	}
	if completedAt.Valid {
		updated.CompletedAt = &completedAt.Time
	}
	return &updated, nil
}

// Delete soft deletes a memo
func (r *MemoRepository) Delete(ctx context.Context, userID int, id int) error {
	// まずメモの現在のstatusを取得
	getQuery := `SELECT status FROM memos WHERE id = $1 AND user_id = $2`
	var status string
	err := r.db.QueryRowContext(ctx, getQuery, id, userID).Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			return usecase.ErrMemoNotFound
		}
		r.logger.WithError(err).Error("Failed to get memo status")
		return fmt.Errorf("failed to get memo status: %w", err)
	}

	if status == "active" {
		// activeならarchivedへ更新
		updateQuery := `UPDATE memos SET status = 'archived', completed_at = $1, updated_at = $2 WHERE id = $3 AND user_id = $4`
		result, err := r.db.ExecContext(ctx, updateQuery, time.Now(), time.Now(), id, userID)
		if err != nil {
			r.logger.WithError(err).Error("Failed to archive memo (delete)")
			return fmt.Errorf("failed to archive memo: %w", err)
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get affected rows: %w", err)
		}
		if rowsAffected == 0 {
			return usecase.ErrMemoNotFound
		}
		return nil
	} else if status == "archived" {
		// archivedなら物理削除
		deleteQuery := `DELETE FROM memos WHERE id = $1 AND user_id = $2`
		result, err := r.db.ExecContext(ctx, deleteQuery, id, userID)
		if err != nil {
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get affected rows: %w", err)
		}
		if rowsAffected == 0 {
			return usecase.ErrMemoNotFound
		}
		return nil
	}
	// 不正なstatus
	return usecase.ErrInvalidStatus
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
		return usecase.ErrMemoNotFound
	}

	return nil
}

// Archive archives a memo
func (r *MemoRepository) Archive(ctx context.Context, id int, userID int) (*domain.Memo, error) {
	now := time.Now()
	query := `UPDATE memos SET status = 'archived', completed_at = $1, updated_at = $2 WHERE id = $3 AND user_id = $4`

	result, err := r.db.ExecContext(ctx, query, now, now, id, userID)
	if err != nil {
		r.logger.WithError(err).Error("Failed to archive memo")
		return nil, fmt.Errorf("failed to archive memo: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return nil, fmt.Errorf("memo not found")
	}

	// アーカイブ後のメモ情報を取得して返す
	memo, err := r.GetByID(ctx, id, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch archived memo: %w", err)
	}
	return memo, nil
}

// Restore restores an archived memo
func (r *MemoRepository) Restore(ctx context.Context, userID int, id int) (*domain.Memo, error) {
	query := `UPDATE memos SET status = 'active', completed_at = NULL, updated_at = $1 WHERE id = $2 AND user_id = $3`

	result, err := r.db.ExecContext(ctx, query, time.Now(), id, userID)
	if err != nil {
		r.logger.WithError(err).Error("Failed to restore memo")
		return nil, fmt.Errorf("failed to restore memo: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		r.logger.WithFields(logrus.Fields{
			"id":      id,
			"user_id": userID,
			"error":   err,
		}).Error("[Delete] Memo not found or error on status fetch")
		return nil, fmt.Errorf("memo not found")
	}

	// 復元後のメモ情報を取得して返す
	memo, err := r.GetByID(ctx, id, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch restored memo: %w", err)
	}

	return memo, nil
}

// Search searches for memos (domain interface)
func (r *MemoRepository) Search(ctx context.Context, userID int, query string, filter domain.MemoFilter) ([]domain.Memo, int, error) {
	searchQuery := `SELECT id, user_id, title, content, category, tags, priority, status, created_at, updated_at, completed_at
		FROM memos 
		WHERE user_id = $1 AND (title ILIKE $2 OR content ILIKE $2)`
	args := []interface{}{userID, "%" + query + "%"}
	argCount := 2
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
	var memos []domain.Memo
	for rows.Next() {
		var memo domain.Memo
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
			var tags []string
			_ = json.Unmarshal([]byte(tagsStr.String), &tags)
			memo.Tags = tags
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
