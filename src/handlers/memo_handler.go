package handlers

import (
	"net/http"
	"strconv"

	"memo-app/src/models"
	"memo-app/src/service"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// MemoHandler represents the memo handler
type MemoHandler struct {
	service service.MemoServiceInterface
	logger  *logrus.Logger
}

// NewMemoHandler creates a new memo handler
func NewMemoHandler(service service.MemoServiceInterface, logger *logrus.Logger) *MemoHandler {
	return &MemoHandler{
		service: service,
		logger:  logger,
	}
}

// CreateMemo creates a new memo
// @Summary Create a new memo
// @Description Create a new memo with title, content, category, tags, and priority
// @Tags memos
// @Accept json
// @Produce json
// @Param memo body models.CreateMemoRequest true "Memo data"
// @Success 201 {object} models.Memo
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/memos [post]
func (h *MemoHandler) CreateMemo(c *gin.Context) {
	// コンテキストからユーザーIDを取得
	userID, exists := c.Get("user_id")
	if !exists {
		h.logger.Error("ユーザーIDがコンテキストに設定されていません")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req models.CreateMemoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("リクエストのバインドに失敗")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	memo, err := h.service.CreateMemo(c.Request.Context(), userID.(int), &req)
	if err != nil {
		h.logger.WithError(err).Error("メモの作成に失敗")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create memo", "details": err.Error()})
		return
	}

	h.logger.WithField("memo_id", memo.ID).WithField("returned_memo_user_id", memo.UserID).Info("メモを作成しました")
	c.JSON(http.StatusCreated, memo)
}

// GetMemo retrieves a memo by ID
// @Summary Get a memo by ID
// @Description Get a specific memo by its ID
// @Tags memos
// @Produce json
// @Param id path int true "Memo ID"
// @Success 200 {object} models.Memo
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/memos/{id} [get]
func (h *MemoHandler) GetMemo(c *gin.Context) {
	// コンテキストからユーザーIDを取得
	userID, exists := c.Get("user_id")
	if !exists {
		h.logger.Error("ユーザーIDがコンテキストに設定されていません")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid memo ID"})
		return
	}

	memo, err := h.service.GetMemo(c.Request.Context(), userID.(int), id)
	if err != nil {
		h.logger.WithError(err).WithField("memo_id", id).Error("メモの取得に失敗")
		if err.Error() == "memo not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Memo not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get memo", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, memo)
}

// ListMemos retrieves memos with filtering
// @Summary List memos with filtering
// @Description Get a list of memos with optional filtering by category, status, priority, search term, and tags
// @Tags memos
// @Produce json
// @Param category query string false "Category filter"
// @Param status query string false "Status filter (active, archived)"
// @Param priority query string false "Priority filter (low, medium, high)"
// @Param search query string false "Search in title and content"
// @Param tags query string false "Tags filter"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10, max: 100)"
// @Success 200 {object} models.MemoListResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/memos [get]
func (h *MemoHandler) ListMemos(c *gin.Context) {
	// コンテキストからユーザーIDを取得
	userID, exists := c.Get("user_id")
	if !exists {
		h.logger.Error("ユーザーIDがコンテキストに設定されていません")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var filter models.MemoFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.logger.WithError(err).Error("クエリパラメータのバインドに失敗")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters", "details": err.Error()})
		return
	}

	result, err := h.service.ListMemos(c.Request.Context(), userID.(int), &filter)
	if err != nil {
		h.logger.WithError(err).Error("メモリストの取得に失敗")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list memos", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// UpdateMemo updates a memo
// @Summary Update a memo
// @Description Update an existing memo's fields
// @Tags memos
// @Accept json
// @Produce json
// @Param id path int true "Memo ID"
// @Param memo body models.UpdateMemoRequest true "Updated memo data"
// @Success 200 {object} models.Memo
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/memos/{id} [put]
func (h *MemoHandler) UpdateMemo(c *gin.Context) {
	// コンテキストからユーザーIDを取得
	userID, exists := c.Get("user_id")
	if !exists {
		h.logger.Error("ユーザーIDがコンテキストに設定されていません")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid memo ID"})
		return
	}

	var req models.UpdateMemoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("リクエストのバインドに失敗")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	memo, err := h.service.UpdateMemo(c.Request.Context(), userID.(int), id, &req)
	if err != nil {
		h.logger.WithError(err).WithField("memo_id", id).Error("メモの更新に失敗")
		if err.Error() == "memo not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Memo not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update memo", "details": err.Error()})
		return
	}

	h.logger.WithField("memo_id", id).Info("メモを更新しました")
	c.JSON(http.StatusOK, memo)
}

// DeleteMemo deletes a memo
// @Summary Delete a memo
// @Description Delete a memo by its ID
// @Tags memos
// @Param id path int true "Memo ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/memos/{id} [delete]
func (h *MemoHandler) DeleteMemo(c *gin.Context) {
	// コンテキストからユーザーIDを取得
	userID, exists := c.Get("user_id")
	if !exists {
		h.logger.Error("ユーザーIDがコンテキストに設定されていません")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid memo ID"})
		return
	}

	err = h.service.DeleteMemo(c.Request.Context(), userID.(int), id)
	if err != nil {
		h.logger.WithError(err).WithField("memo_id", id).Error("メモの削除に失敗")
		if err.Error() == "memo not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Memo not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete memo", "details": err.Error()})
		return
	}

	h.logger.WithField("memo_id", id).Info("メモを削除しました")
	c.Status(http.StatusNoContent)
}

// ArchiveMemo archives a memo
// @Summary Archive a memo
// @Description Set a memo's status to archived
// @Tags memos
// @Param id path int true "Memo ID"
// @Success 200 {object} models.Memo
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/memos/{id}/archive [patch]
func (h *MemoHandler) ArchiveMemo(c *gin.Context) {
	// コンテキストからユーザーIDを取得
	userID, exists := c.Get("user_id")
	if !exists {
		h.logger.Error("ユーザーIDがコンテキストに設定されていません")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid memo ID"})
		return
	}

	memo, err := h.service.ArchiveMemo(c.Request.Context(), userID.(int), id)
	if err != nil {
		h.logger.WithError(err).WithField("memo_id", id).Error("メモのアーカイブに失敗")
		if err.Error() == "memo not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Memo not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to archive memo", "details": err.Error()})
		return
	}

	h.logger.WithField("memo_id", id).Info("メモをアーカイブしました")
	c.JSON(http.StatusOK, memo)
}

// RestoreMemo restores an archived memo
// @Summary Restore an archived memo
// @Description Set an archived memo's status back to active
// @Tags memos
// @Param id path int true "Memo ID"
// @Success 200 {object} models.Memo
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/memos/{id}/restore [patch]
func (h *MemoHandler) RestoreMemo(c *gin.Context) {
	// コンテキストからユーザーIDを取得
	userID, exists := c.Get("user_id")
	if !exists {
		h.logger.Error("ユーザーIDがコンテキストに設定されていません")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid memo ID"})
		return
	}

	memo, err := h.service.RestoreMemo(c.Request.Context(), userID.(int), id)
	if err != nil {
		h.logger.WithError(err).WithField("memo_id", id).Error("メモの復元に失敗")
		if err.Error() == "memo not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Memo not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to restore memo", "details": err.Error()})
		return
	}

	h.logger.WithField("memo_id", id).Info("メモを復元しました")
	c.JSON(http.StatusOK, memo)
}

// SearchMemos searches memos
// @Summary Search memos
// @Description Search memos by content
// @Tags memos
// @Param q query string true "Search query"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10, max: 100)"
// @Success 200 {object} models.MemoListResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/memos/search [get]
func (h *MemoHandler) SearchMemos(c *gin.Context) {
	// コンテキストからユーザーIDを取得
	userID, exists := c.Get("user_id")
	if !exists {
		h.logger.Error("ユーザーIDがコンテキストに設定されていません")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query is required"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	result, err := h.service.SearchMemos(c.Request.Context(), userID.(int), query, page, limit)
	if err != nil {
		h.logger.WithError(err).Error("メモ検索に失敗")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search memos", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
