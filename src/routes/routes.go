package routes

import (
	"memo-app/src/interface/handler"
	"memo-app/src/middleware"

	"github.com/gin-gonic/gin"
)

// SetupRoutes sets up all API routes
func SetupRoutes(r *gin.Engine, memoHandler *handler.MemoHandler) {
	// パブリックルートのグループ化
	api := r.Group("/api")
	api.Use(middleware.LoggerMiddleware())
	api.Use(middleware.CORSMiddleware())
	api.Use(middleware.RateLimitMiddleware())

	// 認証が必要なメモAPIルート
	memos := api.Group("/memos")
	memos.Use(middleware.AuthMiddleware())
	{
		// メモの基本CRUD操作
		memos.POST("", memoHandler.CreateMemo)       // POST /api/memos
		memos.GET("", memoHandler.ListMemos)         // GET /api/memos
		memos.GET("/:id", memoHandler.GetMemo)       // GET /api/memos/:id
		memos.PUT("/:id", memoHandler.UpdateMemo)    // PUT /api/memos/:id
		memos.DELETE("/:id", memoHandler.DeleteMemo) // DELETE /api/memos/:id

		// メモの特別な操作
		memos.PATCH("/:id/archive", memoHandler.ArchiveMemo) // PATCH /api/memos/:id/archive
		memos.PATCH("/:id/restore", memoHandler.RestoreMemo) // PATCH /api/memos/:id/restore

		// 検索機能
		memos.GET("/search", memoHandler.SearchMemos) // GET /api/memos/search
	}
}
