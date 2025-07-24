package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"memo-app/src/database"
	"memo-app/src/domain"
	"memo-app/src/security"

	"github.com/sirupsen/logrus"
)

// MemoRepository implements domain.MemoRepository
type MemoRepository struct {
	db           *database.DB
	logger       *logrus.Logger
	sqlSanitizer *security.SQLSanitizer
}

// NewMemoRepository creates a new memo repository
func NewMemoRepository(db *database.DB, logger *logrus.Logger) domain.MemoRepository {
	return &MemoRepository{
		db:           db,
		logger:       logger,
		sqlSanitizer: security.NewSQLSanitizer(),
	}
}

// Create creates a new memo
func (r *MemoRepository) Create(ctx context.Context, memo *domain.Memo) (*domain.Memo, error) {
	// タグを JSON 文字列に変換
	tagsJSON, err := json.Marshal(memo.Tags)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tags: %w", err)
	}

	now := time.Now()
	newMemo := &domain.Memo{
		Title:     memo.Title,
		Content:   memo.Content,
		Category:  memo.Category,
		Tags:      memo.Tags,
		Priority:  memo.Priority,
		Status:    domain.StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}

	query := `
		INSERT INTO memos (title, content, category, tags, priority, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`

	err = r.db.QueryRowContext(ctx, query,
		newMemo.Title, newMemo.Content, newMemo.Category, string(tagsJSON),
		string(newMemo.Priority), string(newMemo.Status), newMemo.CreatedAt, newMemo.UpdatedAt,
	).Scan(&newMemo.ID)

	if err != nil {
		r.logger.WithError(err).Error("メモの作成に失敗")
		return nil, fmt.Errorf("failed to create memo: %w", err)
	}

	r.logger.WithField("memo_id", newMemo.ID).Info("メモを作成しました")
	return newMemo, nil
}

// GetByID retrieves a memo by ID
func (r *MemoRepository) GetByID(ctx context.Context, id int) (*domain.Memo, error) {
	query := `
		SELECT id, title, content, category, tags, priority, status, created_at, updated_at, completed_at
		FROM memos WHERE id = $1`

	var memo domain.Memo
	var tagsJSON string
	var priorityStr string
	var statusStr string
	var completedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&memo.ID, &memo.Title, &memo.Content, &memo.Category, &tagsJSON,
		&priorityStr, &statusStr, &memo.CreatedAt, &memo.UpdatedAt, &completedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("memo not found")
		}
		r.logger.WithError(err).WithField("memo_id", id).Error("メモの取得に失敗")
		return nil, fmt.Errorf("failed to get memo: %w", err)
	}

	// JSON文字列からタグを復元
	if err := json.Unmarshal([]byte(tagsJSON), &memo.Tags); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
	}

	memo.Priority = domain.Priority(priorityStr)
	memo.Status = domain.Status(statusStr)
	if completedAt.Valid {
		memo.CompletedAt = &completedAt.Time
	}

	return &memo, nil
}

// List retrieves memos with filtering
func (r *MemoRepository) List(ctx context.Context, filter domain.MemoFilter) ([]domain.Memo, int, error) {
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
		args = append(args, string(filter.Status))
		argIndex++
	}

	if filter.Priority != "" {
		baseQuery += fmt.Sprintf(" AND priority = $%d", argIndex)
		args = append(args, string(filter.Priority))
		argIndex++
	}

	if filter.Search != "" {
		baseQuery += fmt.Sprintf(" AND (title ILIKE $%d OR content ILIKE $%d)", argIndex, argIndex)
		// LIKE演算子用のエスケープ処理
		escapedSearch := r.sqlSanitizer.EscapeForLike(filter.Search)
		args = append(args, "%"+escapedSearch+"%")
		argIndex++
	}

	if len(filter.Tags) > 0 {
		for _, tag := range filter.Tags {
			baseQuery += fmt.Sprintf(" AND tags::text ILIKE $%d", argIndex)
			// タグもエスケープ処理
			escapedTag := r.sqlSanitizer.EscapeForLike(tag)
			args = append(args, "%"+escapedTag+"%")
			argIndex++
		}
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
		return nil, 0, fmt.Errorf("failed to count memos: %w", err)
	}

	// ページネーションを追加
	selectQuery += " ORDER BY updated_at DESC"
	selectQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)

	// メモを取得
	rows, err := r.db.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		r.logger.WithError(err).Error("メモリストの取得に失敗")
		return nil, 0, fmt.Errorf("failed to get memos: %w", err)
	}
	defer rows.Close()

	var memos []domain.Memo
	for rows.Next() {
		var memo domain.Memo
		var tagsJSON string
		var priorityStr string
		var statusStr string
		var completedAt sql.NullTime

		err := rows.Scan(
			&memo.ID, &memo.Title, &memo.Content, &memo.Category, &tagsJSON,
			&priorityStr, &statusStr, &memo.CreatedAt, &memo.UpdatedAt, &completedAt,
		)
		if err != nil {
			r.logger.WithError(err).Error("メモのスキャンに失敗")
			return nil, 0, fmt.Errorf("failed to scan memo: %w", err)
		}

		// JSON文字列からタグを復元
		if err := json.Unmarshal([]byte(tagsJSON), &memo.Tags); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal tags: %w", err)
		}

		memo.Priority = domain.Priority(priorityStr)
		memo.Status = domain.Status(statusStr)
		if completedAt.Valid {
			memo.CompletedAt = &completedAt.Time
		}

		memos = append(memos, memo)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}

	return memos, total, nil
}

// Update updates a memo
func (r *MemoRepository) Update(ctx context.Context, id int, memo *domain.Memo) (*domain.Memo, error) {
	// タグを JSON 文字列に変換
	tagsJSON, err := json.Marshal(memo.Tags)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tags: %w", err)
	}

	now := time.Now()
	memo.UpdatedAt = now

	// ステータスがarchivedの場合、完了日時を設定
	if memo.Status == domain.StatusArchived && memo.CompletedAt == nil {
		memo.CompletedAt = &now
	}

	query := `
		UPDATE memos SET 
			title = $2, 
			content = $3, 
			category = $4, 
			tags = $5, 
			priority = $6, 
			status = $7, 
			updated_at = $8, 
			completed_at = $9
		WHERE id = $1
		RETURNING id, title, content, category, tags, priority, status, created_at, updated_at, completed_at`

	var updatedMemo domain.Memo
	var tagsJSONResult string
	var priorityStr string
	var statusStr string
	var completedAt sql.NullTime

	err = r.db.QueryRowContext(ctx, query,
		id, memo.Title, memo.Content, memo.Category, string(tagsJSON),
		string(memo.Priority), string(memo.Status), memo.UpdatedAt, memo.CompletedAt,
	).Scan(
		&updatedMemo.ID, &updatedMemo.Title, &updatedMemo.Content, &updatedMemo.Category, &tagsJSONResult,
		&priorityStr, &statusStr, &updatedMemo.CreatedAt, &updatedMemo.UpdatedAt, &completedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("memo not found")
		}
		r.logger.WithError(err).WithField("memo_id", id).Error("メモの更新に失敗")
		return nil, fmt.Errorf("failed to update memo: %w", err)
	}

	// JSON文字列からタグを復元
	if err := json.Unmarshal([]byte(tagsJSONResult), &updatedMemo.Tags); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
	}

	updatedMemo.Priority = domain.Priority(priorityStr)
	updatedMemo.Status = domain.Status(statusStr)
	if completedAt.Valid {
		updatedMemo.CompletedAt = &completedAt.Time
	}

	r.logger.WithField("memo_id", id).Info("メモを更新しました")
	return &updatedMemo, nil
}

// Delete handles memo deletion with staged approach
func (r *MemoRepository) Delete(ctx context.Context, id int) error {
	r.logger.WithField("memo_id", id).Info("=== インフラストラクチャリポジトリのDeleteメソッドが呼ばれました ===")

	// まず、メモの現在の状態を確認
	memo, err := r.GetByID(ctx, id)
	if err != nil {
		r.logger.WithError(err).WithField("memo_id", id).Error("メモの取得に失敗")
		return err
	}

	r.logger.WithField("memo_id", id).WithField("current_status", memo.Status).Info("メモの現在のステータス")

	// すでにアーカイブ済みの場合は完全削除
	if memo.Status == domain.StatusArchived {
		r.logger.WithField("memo_id", id).Info("アーカイブ済みメモを完全削除します")
		return r.PermanentDelete(ctx, id)
	}

	// アクティブなメモの場合はアーカイブに移動
	r.logger.WithField("memo_id", id).Info("メモをアーカイブに移動します")
	return r.Archive(ctx, id)
}

// PermanentDelete permanently deletes a memo from database
func (r *MemoRepository) PermanentDelete(ctx context.Context, id int) error {
	query := `DELETE FROM memos WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.WithError(err).WithField("memo_id", id).Error("メモの完全削除に失敗")
		return fmt.Errorf("failed to permanently delete memo: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("memo not found")
	}

	r.logger.WithField("memo_id", id).Info("メモを完全削除しました")
	return nil
}

// Archive archives a memo
func (r *MemoRepository) Archive(ctx context.Context, id int) error {
	r.logger.WithField("memo_id", id).Info("=== Archiveメソッドが呼ばれました ===")

	memo, err := r.GetByID(ctx, id)
	if err != nil {
		r.logger.WithError(err).WithField("memo_id", id).Error("アーカイブ対象メモの取得に失敗")
		return err
	}

	r.logger.WithField("memo_id", id).WithField("before_status", memo.Status).Info("アーカイブ前のステータス")

	memo.Status = domain.StatusArchived
	now := time.Now()
	memo.CompletedAt = &now

	updatedMemo, err := r.Update(ctx, id, memo)
	if err != nil {
		r.logger.WithError(err).WithField("memo_id", id).Error("アーカイブ更新に失敗")
		return err
	}

	r.logger.WithField("memo_id", id).WithField("new_status", updatedMemo.Status).Info("メモをアーカイブしました")
	return nil
}

// Restore restores an archived memo
func (r *MemoRepository) Restore(ctx context.Context, id int) error {
	memo, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	memo.Status = domain.StatusActive
	memo.CompletedAt = nil

	_, err = r.Update(ctx, id, memo)
	return err
}

// Search searches memos by query
func (r *MemoRepository) Search(ctx context.Context, query string, filter domain.MemoFilter) ([]domain.Memo, int, error) {
	// 検索クエリのバリデーションとサニタイゼーション
	if err := r.sqlSanitizer.ValidateSearchQuery(query); err != nil {
		r.logger.WithError(err).WithField("query", query).Error("危険な検索クエリが検出されました")
		return nil, 0, fmt.Errorf("invalid search query: %w", err)
	}

	// 検索クエリをサニタイズ
	sanitizedQuery := r.sqlSanitizer.SanitizeSearchQuery(query)

	// ページネーションパラメータのバリデーション
	offset := (filter.Page - 1) * filter.Limit
	if err := r.sqlSanitizer.ValidateLimitOffset(filter.Limit, offset); err != nil {
		r.logger.WithError(err).Error("無効なページネーションパラメータ")
		return nil, 0, fmt.Errorf("invalid pagination: %w", err)
	}

	// サニタイズされた検索クエリをフィルターに設定
	filter.Search = sanitizedQuery
	return r.List(ctx, filter)
}
