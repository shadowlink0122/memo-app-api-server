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
func (r *MemoRepository) Create(ctx context.Context, req *models.CreateMemoRequest) (*models.Memo, error) {
	// タグを JSON 文字列に変換
	tagsJSON, err := json.Marshal(req.Tags)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tags: %w", err)
	}

	now := time.Now()
	memo := &models.Memo{
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
		INSERT INTO memos (title, content, category, tags, priority, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`

	err = r.db.QueryRowContext(ctx, query,
		memo.Title, memo.Content, memo.Category, memo.Tags,
		memo.Priority, memo.Status, memo.CreatedAt, memo.UpdatedAt,
	).Scan(&memo.ID)

	if err != nil {
		r.logger.WithError(err).Error("メモの作成に失敗")
		return nil, fmt.Errorf("failed to create memo: %w", err)
	}

	r.logger.WithField("memo_id", memo.ID).Info("メモを作成しました")
	return memo, nil
}

// GetByID retrieves a memo by ID
func (r *MemoRepository) GetByID(ctx context.Context, id int) (*models.Memo, error) {
	query := `
		SELECT id, title, content, category, tags, priority, status, created_at, updated_at, completed_at
		FROM memos WHERE id = $1`

	memo := &models.Memo{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&memo.ID, &memo.Title, &memo.Content, &memo.Category, &memo.Tags,
		&memo.Priority, &memo.Status, &memo.CreatedAt, &memo.UpdatedAt, &memo.CompletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("memo not found")
		}
		r.logger.WithError(err).WithField("memo_id", id).Error("メモの取得に失敗")
		return nil, fmt.Errorf("failed to get memo: %w", err)
	}

	return memo, nil
}

// List retrieves memos with filtering
func (r *MemoRepository) List(ctx context.Context, filter *models.MemoFilter) (*models.MemoListResponse, error) {
	// ベースクエリ
	baseQuery := `FROM memos WHERE 1=1`
	countQuery := `SELECT COUNT(*) ` + baseQuery
	selectQuery := `
		SELECT id, title, content, category, tags, priority, status, created_at, updated_at, completed_at
		` + baseQuery

	var args []interface{}
	argIndex := 1

	// フィルター条件を追加
	if filter.Category != "" {
		baseQuery += fmt.Sprintf(" AND category = $%d", argIndex)
		args = append(args, filter.Category)
		argIndex++
	}

	if filter.Status != "" {
		baseQuery += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, filter.Status)
		argIndex++
	}

	if filter.Priority != "" {
		baseQuery += fmt.Sprintf(" AND priority = $%d", argIndex)
		args = append(args, filter.Priority)
		argIndex++
	}

	if filter.Search != "" {
		baseQuery += fmt.Sprintf(" AND (title ILIKE $%d OR content ILIKE $%d)", argIndex, argIndex)
		args = append(args, "%"+filter.Search+"%")
		argIndex++
	}

	if filter.Tags != "" {
		baseQuery += fmt.Sprintf(" AND tags::text ILIKE $%d", argIndex)
		args = append(args, "%"+filter.Tags+"%")
		argIndex++
	}

	// 更新されたクエリ
	countQuery = `SELECT COUNT(*) ` + baseQuery
	selectQuery = `
		SELECT id, title, content, category, tags, priority, status, created_at, updated_at, completed_at
		` + baseQuery

	// 総数を取得
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		r.logger.WithError(err).Error("メモ総数の取得に失敗")
		return nil, fmt.Errorf("failed to count memos: %w", err)
	}

	// ページネーションを追加
	selectQuery += " ORDER BY updated_at DESC"
	selectQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)

	// メモを取得
	rows, err := r.db.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		r.logger.WithError(err).Error("メモリストの取得に失敗")
		return nil, fmt.Errorf("failed to get memos: %w", err)
	}
	defer rows.Close()

	var memos []models.Memo
	for rows.Next() {
		var memo models.Memo
		err := rows.Scan(
			&memo.ID, &memo.Title, &memo.Content, &memo.Category, &memo.Tags,
			&memo.Priority, &memo.Status, &memo.CreatedAt, &memo.UpdatedAt, &memo.CompletedAt,
		)
		if err != nil {
			r.logger.WithError(err).Error("メモのスキャンに失敗")
			return nil, fmt.Errorf("failed to scan memo: %w", err)
		}
		memos = append(memos, memo)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	totalPages := (total + filter.Limit - 1) / filter.Limit

	return &models.MemoListResponse{
		Memos:      memos,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

// Update updates a memo
func (r *MemoRepository) Update(ctx context.Context, id int, req *models.UpdateMemoRequest) (*models.Memo, error) {
	// 既存のメモを取得
	existing, err := r.GetByID(ctx, id)
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
		WHERE id = $%d
		RETURNING id, title, content, category, tags, priority, status, created_at, updated_at, completed_at`,
		strings.Join(setParts, ", "), argIndex)

	memo := &models.Memo{}
	err = r.db.QueryRowContext(ctx, query, args...).Scan(
		&memo.ID, &memo.Title, &memo.Content, &memo.Category, &memo.Tags,
		&memo.Priority, &memo.Status, &memo.CreatedAt, &memo.UpdatedAt, &memo.CompletedAt,
	)

	if err != nil {
		r.logger.WithError(err).WithField("memo_id", id).Error("メモの更新に失敗")
		return nil, fmt.Errorf("failed to update memo: %w", err)
	}

	r.logger.WithField("memo_id", id).Info("メモを更新しました")
	return memo, nil
}

// Delete deletes a memo
func (r *MemoRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM memos WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.WithError(err).WithField("memo_id", id).Error("メモの削除に失敗")
		return fmt.Errorf("failed to delete memo: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("memo not found")
	}

	r.logger.WithField("memo_id", id).Info("メモを削除しました")
	return nil
}
