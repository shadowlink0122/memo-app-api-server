package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"memo-app/src/database"
	"memo-app/src/models"

	"github.com/sirupsen/logrus"
)

// MemoRepository represents the memo repository
type MemoRepository struct {
	db     *database.DB
	logger *logrus.Logger
}

// NewMemoRepository creates a new memo repository
func NewMemoRepository(db *database.DB, logger *logrus.Logger) *MemoRepository {
	return &MemoRepository{
		db:     db,
		logger: logger,
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
	memo := &models.Memo{
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

	query := `
		INSERT INTO memos (user_id, title, content, category, tags, priority, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`

	err = r.db.QueryRowContext(ctx, query,
		memo.UserID, memo.Title, memo.Content, memo.Category, memo.Tags,
		memo.Priority, memo.Status, memo.CreatedAt, memo.UpdatedAt,
	).Scan(&memo.ID)

	if err != nil {
		r.logger.WithError(err).Error("メモの作成に失敗")
		return nil, fmt.Errorf("failed to create memo: %w", err)
	}

	r.logger.WithField("memo_id", memo.ID).Info("メモを作成しました")
	return memo, nil
}

// GetByID retrieves a memo by ID for a specific user
func (r *MemoRepository) GetByID(ctx context.Context, id int, userID int) (*models.Memo, error) {
	query := `
		SELECT id, user_id, title, content, category, tags, priority, status, created_at, updated_at, completed_at
		FROM memos WHERE id = $1 AND user_id = $2`

	memo := &models.Memo{}
	err := r.db.QueryRowContext(ctx, query, id, userID).Scan(
		&memo.ID, &memo.UserID, &memo.Title, &memo.Content, &memo.Category, &memo.Tags,
		&memo.Priority, &memo.Status, &memo.CreatedAt, &memo.UpdatedAt, &memo.CompletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("memo not found or access denied")
		}
		r.logger.WithError(err).WithField("memo_id", id).WithField("user_id", userID).Error("メモの取得に失敗")
		return nil, fmt.Errorf("failed to get memo: %w", err)
	}

	return memo, nil
}

// List retrieves all memos with optional filtering
func (r *MemoRepository) List(ctx context.Context, userID int, filter models.MemoFilter) ([]models.Memo, int, error) {
	// この古いリポジトリは使用されていないはずです
	return nil, 0, fmt.Errorf("OLD REPOSITORY SHOULD NOT BE USED")
}

// Update updates a memo for a specific user
func (r *MemoRepository) Update(ctx context.Context, userID int, id int, req *models.UpdateMemoRequest) (*models.Memo, error) {
	// 既存のメモを取得（ユーザー固有）
	existing, err := r.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	// 更新するフィールドを構築
	var setParts []string
	var args []interface{}
	argIndex := 1

	if req.Title != nil {
		setParts = append(setParts, fmt.Sprintf("title = $%d", argIndex))
		args = append(args, *req.Title)
		argIndex++
	}

	if req.Content != nil {
		setParts = append(setParts, fmt.Sprintf("content = $%d", argIndex))
		args = append(args, *req.Content)
		argIndex++
	}

	if req.Category != nil {
		setParts = append(setParts, fmt.Sprintf("category = $%d", argIndex))
		args = append(args, *req.Category)
		argIndex++
	}

	if req.Tags != nil {
		tagsJSON, err := json.Marshal(req.Tags)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal tags: %w", err)
		}
		setParts = append(setParts, fmt.Sprintf("tags = $%d", argIndex))
		args = append(args, string(tagsJSON))
		argIndex++
	}

	if req.Priority != nil {
		setParts = append(setParts, fmt.Sprintf("priority = $%d", argIndex))
		args = append(args, *req.Priority)
		argIndex++
	}

	if req.Status != nil {
		setParts = append(setParts, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *req.Status)
		argIndex++

		// ステータスがarchivedの場合、完了日時を設定
		if *req.Status == "archived" {
			setParts = append(setParts, fmt.Sprintf("completed_at = $%d", argIndex))
			args = append(args, time.Now())
			argIndex++
		}
	}

	if len(setParts) == 0 {
		return existing, nil // 更新するフィールドがない場合は既存のメモを返す
	}

	// updated_atを常に更新
	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	// IDを最後の引数として追加
	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE memos SET %s
		WHERE id = $%d AND user_id = $%d
		RETURNING id, user_id, title, content, category, tags, priority, status, created_at, updated_at, completed_at`,
		strings.Join(setParts, ", "), argIndex, argIndex+1)

	// ユーザーIDを追加
	args = append(args, userID)

	memo := &models.Memo{}
	err = r.db.QueryRowContext(ctx, query, args...).Scan(
		&memo.ID, &memo.UserID, &memo.Title, &memo.Content, &memo.Category, &memo.Tags,
		&memo.Priority, &memo.Status, &memo.CreatedAt, &memo.UpdatedAt, &memo.CompletedAt,
	)

	if err != nil {
		r.logger.WithError(err).WithField("memo_id", id).Error("メモの更新に失敗")
		return nil, fmt.Errorf("failed to update memo: %w", err)
	}

	r.logger.WithField("memo_id", id).Info("メモを更新しました")
	return memo, nil
}

// Delete handles memo deletion with staged approach for a specific user
func (r *MemoRepository) Delete(ctx context.Context, userID int, id int) error {
	r.logger.WithField("memo_id", id).WithField("user_id", userID).Info("=== レガシーリポジトリのDeleteメソッドが呼ばれました ===")

	// まず、メモの現在の状態を確認（ユーザー固有）
	memo, err := r.GetByID(ctx, id, userID)
	if err != nil {
		r.logger.WithError(err).WithField("memo_id", id).WithField("user_id", userID).Error("メモの取得に失敗")
		return err
	}

	r.logger.WithField("memo_id", id).WithField("current_status", memo.Status).Info("メモの現在のステータス")

	// すでにアーカイブ済みの場合は完全削除
	if memo.Status == "archived" {
		r.logger.WithField("memo_id", id).WithField("user_id", userID).Info("アーカイブ済みメモを完全削除します")
		return r.PermanentDelete(ctx, id, userID)
	}

	// アクティブなメモの場合はアーカイブに移動
	r.logger.WithField("memo_id", id).WithField("user_id", userID).Info("メモをアーカイブに移動します")
	status := "archived"
	updateReq := &models.UpdateMemoRequest{
		Status: &status,
	}
	updatedMemo, err := r.Update(ctx, id, userID, updateReq)
	if err != nil {
		r.logger.WithError(err).WithField("memo_id", id).WithField("user_id", userID).Error("アーカイブ更新に失敗")
		return err
	}

	r.logger.WithField("memo_id", id).WithField("new_status", updatedMemo.Status).Info("メモをアーカイブしました")
	return nil
}

// PermanentDelete permanently deletes a memo from database for a specific user
func (r *MemoRepository) PermanentDelete(ctx context.Context, userID int, id int) error {
	query := `DELETE FROM memos WHERE id = $1 AND user_id = $2`

	result, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		r.logger.WithError(err).WithField("memo_id", id).WithField("user_id", userID).Error("メモの完全削除に失敗")
		return fmt.Errorf("failed to permanently delete memo: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("memo not found or access denied")
	}

	r.logger.WithField("memo_id", id).WithField("user_id", userID).Info("メモを完全削除しました")
	return nil
}

// Archive archives a memo for a specific user
func (r *MemoRepository) Archive(ctx context.Context, userID int, id int) error {
	query := `UPDATE memos SET status = 'archived', updated_at = $1, completed_at = $1 WHERE id = $2 AND user_id = $3`

	result, err := r.db.ExecContext(ctx, query, time.Now(), id, userID)
	if err != nil {
		r.logger.WithError(err).WithField("memo_id", id).WithField("user_id", userID).Error("メモのアーカイブに失敗")
		return fmt.Errorf("failed to archive memo: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("memo not found or access denied")
	}

	r.logger.WithField("memo_id", id).WithField("user_id", userID).Info("メモをアーカイブしました")
	return nil
}

// Restore restores an archived memo for a specific user
func (r *MemoRepository) Restore(ctx context.Context, userID int, id int) error {
	query := `UPDATE memos SET status = 'active', updated_at = $1, completed_at = NULL WHERE id = $2 AND user_id = $3`

	result, err := r.db.ExecContext(ctx, query, time.Now(), id, userID)
	if err != nil {
		r.logger.WithError(err).WithField("memo_id", id).WithField("user_id", userID).Error("メモの復元に失敗")
		return fmt.Errorf("failed to restore memo: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("memo not found or access denied")
	}

	r.logger.WithField("memo_id", id).WithField("user_id", userID).Info("メモを復元しました")
	return nil
}

// Search searches for memos for a specific user
func (r *MemoRepository) Search(ctx context.Context, userID int, query string, filter models.MemoFilter) ([]models.Memo, int, error) {
	// 基本的なクエリ構築（ユーザー固有）
	baseQuery := `
		SELECT id, user_id, title, content, category, tags, priority, status, created_at, updated_at, completed_at
		FROM memos
		WHERE user_id = $1 AND (title ILIKE $2 OR content ILIKE $2)`

	args := []interface{}{userID, "%" + query + "%"}
	argIndex := 3

	// フィルター条件を追加
	if filter.Status != "" {
		baseQuery += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, filter.Status)
		argIndex++
	}

	if filter.Category != "" {
		baseQuery += fmt.Sprintf(" AND category = $%d", argIndex)
		args = append(args, filter.Category)
		argIndex++
	}

	if filter.Priority != "" {
		baseQuery += fmt.Sprintf(" AND priority = $%d", argIndex)
		args = append(args, filter.Priority)
		argIndex++
	}

	// タグフィルター
	if len(filter.Tags) > 0 {
		placeholders := make([]string, len(filter.Tags))
		for i, tag := range filter.Tags {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, tag)
			argIndex++
		}
		baseQuery += fmt.Sprintf(" AND tags && ARRAY[%s]", strings.Join(placeholders, ","))
	}

	// 順序とページネーション
	baseQuery += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		offset := (filter.Page - 1) * filter.Limit
		baseQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
		args = append(args, filter.Limit, offset)
	}

	rows, err := r.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		r.logger.WithError(err).Error("メモ検索クエリの実行に失敗")
		return nil, 0, fmt.Errorf("failed to search memos: %w", err)
	}
	defer rows.Close()

	var memos []models.Memo
	for rows.Next() {
		var memo models.Memo
		err := rows.Scan(
			&memo.ID, &memo.UserID, &memo.Title, &memo.Content, &memo.Category, &memo.Tags,
			&memo.Priority, &memo.Status, &memo.CreatedAt, &memo.UpdatedAt, &memo.CompletedAt,
		)
		if err != nil {
			r.logger.WithError(err).Error("メモデータのスキャンに失敗")
			continue
		}
		memos = append(memos, memo)
	}

	// 総件数を取得（ページネーション用）
	countQuery := `
		SELECT COUNT(*)
		FROM memos
		WHERE user_id = $1 AND (title ILIKE $2 OR content ILIKE $2)`

	countArgs := []interface{}{userID, "%" + query + "%"}
	countArgIndex := 3

	if filter.Status != "" {
		countQuery += fmt.Sprintf(" AND status = $%d", countArgIndex)
		countArgs = append(countArgs, filter.Status)
		countArgIndex++
	}

	if filter.Category != "" {
		countQuery += fmt.Sprintf(" AND category = $%d", countArgIndex)
		countArgs = append(countArgs, filter.Category)
		countArgIndex++
	}

	if filter.Priority != "" {
		countQuery += fmt.Sprintf(" AND priority = $%d", countArgIndex)
		countArgs = append(countArgs, filter.Priority)
		countArgIndex++
	}

	if len(filter.Tags) > 0 {
		placeholders := make([]string, len(filter.Tags))
		for i, tag := range filter.Tags {
			placeholders[i] = fmt.Sprintf("$%d", countArgIndex)
			countArgs = append(countArgs, tag)
			countArgIndex++
		}
		countQuery += fmt.Sprintf(" AND tags && ARRAY[%s]", strings.Join(placeholders, ","))
	}

	var total int
	err = r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		r.logger.WithError(err).Error("メモ検索総件数の取得に失敗")
		return memos, 0, fmt.Errorf("failed to get search count: %w", err)
	}

	return memos, total, nil
}
