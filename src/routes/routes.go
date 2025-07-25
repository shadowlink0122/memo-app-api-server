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

	// TODO: 認証システムを完全に統合後に有効化
	// 認証関連のパブリックルート
	// auth := api.Group("/auth")
	// {
	//     auth.POST("/register", authHandler.Register)
	//     auth.POST("/login", authHandler.Login)
	//     auth.POST("/refresh", authHandler.RefreshToken)
	//     auth.GET("/github/url", authHandler.GetGitHubAuthURL)
	//     auth.GET("/github/callback", authHandler.GitHubCallback)
	// }

	// 一時的に認証なしでメモAPIを利用可能にする
	memos := api.Group("/memos")
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
