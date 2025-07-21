package handler

import (
	"net/http"
	"strconv"
	"strings"

	"memo-app/src/domain"
	"memo-app/src/usecase"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// MemoHandler handles HTTP requests for memo operations
type MemoHandler struct {
	memoUsecase usecase.MemoUsecase
	logger      *logrus.Logger
}

// NewMemoHandler creates a new memo handler
func NewMemoHandler(memoUsecase usecase.MemoUsecase, logger *logrus.Logger) *MemoHandler {
	return &MemoHandler{
		memoUsecase: memoUsecase,
		logger:      logger,
	}
}

// CreateMemo creates a new memo
func (h *MemoHandler) CreateMemo(c *gin.Context) {
	var req CreateMemoRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("リクエストのバインドに失敗")
		c.JSON(http.StatusBadRequest, ErrorResponseDTO{
			Error:   "Invalid request format",
			Message: err.Error(),
		})
		return
	}

	usecaseReq := usecase.CreateMemoRequest{
		Title:    req.Title,
		Content:  req.Content,
		Category: req.Category,
		Tags:     req.Tags,
		Priority: req.Priority,
	}

	memo, err := h.memoUsecase.CreateMemo(c.Request.Context(), usecaseReq)
	if err != nil {
		h.logger.WithError(err).Error("メモの作成に失敗")

		status := http.StatusInternalServerError
		if err == usecase.ErrInvalidTitle || err == usecase.ErrInvalidContent || err == usecase.ErrInvalidPriority {
			status = http.StatusBadRequest
		}

		c.JSON(status, ErrorResponseDTO{
			Error:   "Failed to create memo",
			Message: err.Error(),
		})
		return
	}

	h.logger.WithField("memo_id", memo.ID).Info("メモを作成しました")
	c.JSON(http.StatusCreated, h.toMemoResponseDTO(memo))
}

// GetMemo retrieves a memo by ID
func (h *MemoHandler) GetMemo(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponseDTO{
			Error:   "Invalid memo ID",
			Message: "Memo ID must be a number",
		})
		return
	}

	memo, err := h.memoUsecase.GetMemo(c.Request.Context(), id)
	if err != nil {
		h.logger.WithError(err).WithField("memo_id", id).Error("メモの取得に失敗")

		status := http.StatusInternalServerError
		if err == usecase.ErrMemoNotFound {
			status = http.StatusNotFound
		}

		c.JSON(status, ErrorResponseDTO{
			Error: "Failed to get memo",
		})
		return
	}

	c.JSON(http.StatusOK, h.toMemoResponseDTO(memo))
}

// ListMemos retrieves memos with filtering
func (h *MemoHandler) ListMemos(c *gin.Context) {
	var filterDTO MemoFilterDTO
	if err := c.ShouldBindQuery(&filterDTO); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponseDTO{
			Error:   "Invalid query parameters",
			Message: err.Error(),
		})
		return
	}

	filter := h.toDomainFilter(filterDTO)

	memos, total, err := h.memoUsecase.ListMemos(c.Request.Context(), filter)
	if err != nil {
		h.logger.WithError(err).Error("メモリストの取得に失敗")

		status := http.StatusInternalServerError
		if err == usecase.ErrInvalidPage || err == usecase.ErrInvalidLimit {
			status = http.StatusBadRequest
		}

		c.JSON(status, ErrorResponseDTO{
			Error:   "Failed to get memos",
			Message: err.Error(),
		})
		return
	}

	response := MemoListResponseDTO{
		Memos:      h.toMemoResponseDTOs(memos),
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: (total + filter.Limit - 1) / filter.Limit,
	}

	c.JSON(http.StatusOK, response)
}

// UpdateMemo updates an existing memo
func (h *MemoHandler) UpdateMemo(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponseDTO{
			Error:   "Invalid memo ID",
			Message: "Memo ID must be a number",
		})
		return
	}

	var req UpdateMemoRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("リクエストのバインドに失敗")
		c.JSON(http.StatusBadRequest, ErrorResponseDTO{
			Error:   "Invalid request format",
			Message: err.Error(),
		})
		return
	}

	usecaseReq := usecase.UpdateMemoRequest{
		Title:    req.Title,
		Content:  req.Content,
		Category: req.Category,
		Tags:     req.Tags,
		Priority: req.Priority,
		Status:   req.Status,
	}

	memo, err := h.memoUsecase.UpdateMemo(c.Request.Context(), id, usecaseReq)
	if err != nil {
		h.logger.WithError(err).WithField("memo_id", id).Error("メモの更新に失敗")

		status := http.StatusInternalServerError
		if err == usecase.ErrMemoNotFound {
			status = http.StatusNotFound
		} else if err == usecase.ErrInvalidTitle || err == usecase.ErrInvalidContent ||
			err == usecase.ErrInvalidPriority || err == usecase.ErrInvalidStatus {
			status = http.StatusBadRequest
		}

		c.JSON(status, ErrorResponseDTO{
			Error:   "Failed to update memo",
			Message: err.Error(),
		})
		return
	}

	h.logger.WithField("memo_id", id).Info("メモを更新しました")
	c.JSON(http.StatusOK, h.toMemoResponseDTO(memo))
}

// DeleteMemo deletes a memo
func (h *MemoHandler) DeleteMemo(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponseDTO{
			Error:   "Invalid memo ID",
			Message: "Memo ID must be a number",
		})
		return
	}

	err = h.memoUsecase.DeleteMemo(c.Request.Context(), id)
	if err != nil {
		h.logger.WithError(err).WithField("memo_id", id).Error("メモの削除に失敗")

		status := http.StatusInternalServerError
		if err == usecase.ErrMemoNotFound {
			status = http.StatusNotFound
		}

		c.JSON(status, ErrorResponseDTO{
			Error: "Failed to delete memo",
		})
		return
	}

	h.logger.WithField("memo_id", id).Info("メモを削除しました")
	c.Status(http.StatusNoContent)
}

// ArchiveMemo archives a memo
func (h *MemoHandler) ArchiveMemo(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponseDTO{
			Error:   "Invalid memo ID",
			Message: "Memo ID must be a number",
		})
		return
	}

	err = h.memoUsecase.ArchiveMemo(c.Request.Context(), id)
	if err != nil {
		h.logger.WithError(err).WithField("memo_id", id).Error("メモのアーカイブに失敗")

		status := http.StatusInternalServerError
		if err == usecase.ErrMemoNotFound {
			status = http.StatusNotFound
		}

		c.JSON(status, ErrorResponseDTO{
			Error: "Failed to archive memo",
		})
		return
	}

	h.logger.WithField("memo_id", id).Info("メモをアーカイブしました")
	c.Status(http.StatusNoContent)
}

// RestoreMemo restores an archived memo
func (h *MemoHandler) RestoreMemo(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponseDTO{
			Error:   "Invalid memo ID",
			Message: "Memo ID must be a number",
		})
		return
	}

	err = h.memoUsecase.RestoreMemo(c.Request.Context(), id)
	if err != nil {
		h.logger.WithError(err).WithField("memo_id", id).Error("メモの復元に失敗")

		status := http.StatusInternalServerError
		if err == usecase.ErrMemoNotFound {
			status = http.StatusNotFound
		}

		c.JSON(status, ErrorResponseDTO{
			Error: "Failed to restore memo",
		})
		return
	}

	h.logger.WithField("memo_id", id).Info("メモを復元しました")
	c.Status(http.StatusNoContent)
}

// SearchMemos searches memos
func (h *MemoHandler) SearchMemos(c *gin.Context) {
	var filterDTO MemoFilterDTO
	if err := c.ShouldBindQuery(&filterDTO); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponseDTO{
			Error:   "Invalid query parameters",
			Message: err.Error(),
		})
		return
	}

	query := filterDTO.Search
	filter := h.toDomainFilter(filterDTO)

	memos, total, err := h.memoUsecase.SearchMemos(c.Request.Context(), query, filter)
	if err != nil {
		h.logger.WithError(err).Error("メモ検索に失敗")

		status := http.StatusInternalServerError
		if err == usecase.ErrInvalidPage || err == usecase.ErrInvalidLimit {
			status = http.StatusBadRequest
		}

		c.JSON(status, ErrorResponseDTO{
			Error:   "Failed to search memos",
			Message: err.Error(),
		})
		return
	}

	response := MemoListResponseDTO{
		Memos:      h.toMemoResponseDTOs(memos),
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: (total + filter.Limit - 1) / filter.Limit,
	}

	c.JSON(http.StatusOK, response)
}

// Helper methods for conversion

func (h *MemoHandler) toMemoResponseDTO(memo *domain.Memo) MemoResponseDTO {
	return MemoResponseDTO{
		ID:          memo.ID,
		Title:       memo.Title,
		Content:     memo.Content,
		Category:    memo.Category,
		Tags:        memo.Tags,
		Priority:    memo.Priority.String(),
		Status:      memo.Status.String(),
		CreatedAt:   memo.CreatedAt,
		UpdatedAt:   memo.UpdatedAt,
		CompletedAt: memo.CompletedAt,
	}
}

func (h *MemoHandler) toMemoResponseDTOs(memos []domain.Memo) []MemoResponseDTO {
	result := make([]MemoResponseDTO, len(memos))
	for i, memo := range memos {
		result[i] = h.toMemoResponseDTO(&memo)
	}
	return result
}

func (h *MemoHandler) toDomainFilter(dto MemoFilterDTO) domain.MemoFilter {
	var tags []string
	if dto.Tags != "" {
		tags = strings.Split(dto.Tags, ",")
		for i := range tags {
			tags[i] = strings.TrimSpace(tags[i])
		}
	}

	return domain.MemoFilter{
		Category: dto.Category,
		Status:   domain.Status(dto.Status),
		Priority: domain.Priority(dto.Priority),
		Search:   dto.Search,
		Tags:     tags,
		Page:     dto.Page,
		Limit:    dto.Limit,
	}
}
