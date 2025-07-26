package routes

import (
	"memo-app/src/handlers"
	"memo-app/src/interface/handler"
	"memo-app/src/logger"
	"memo-app/src/middleware"
	"memo-app/src/repository"
	"memo-app/src/service"

	"github.com/gin-gonic/gin"
)

// SetupRoutes sets up all API routes
func SetupRoutes(r *gin.Engine, memoHandler *handler.MemoHandler, authHandler *handlers.AuthHandler, jwtService service.JWTService, userRepo repository.UserRepository) {
	// パブリックルートのグループ化
	api := r.Group("/api")
	api.Use(middleware.LoggerMiddleware())
	api.Use(middleware.CORSMiddleware())
	api.Use(middleware.RateLimitMiddleware())

	// 認証関連のパブリックルート
	auth := api.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.RefreshToken)
		auth.POST("/logout", authHandler.Logout)
		auth.GET("/github/url", authHandler.GetGitHubAuthURL)
		auth.GET("/github/callback", authHandler.GitHubCallback)
	}

	// 認証が必要なメモAPIエンドポイント
	memos := api.Group("/memos")
	memos.Use(middleware.AuthMiddleware(jwtService, userRepo))
	{
		// メモの基本CRUD操作
		memos.POST("", memoHandler.CreateMemo)   // POST /api/memos
		memos.GET("", memoHandler.ListMemos)     // GET /api/memos (activeのみ)
		memos.GET(":id", memoHandler.GetMemo)    // GET /api/memos/:id
		memos.PUT(":id", memoHandler.UpdateMemo) // PUT /api/memos/:id
		// DELETE /api/memos/:id
		// permanent=true で完全削除、falseまたは未指定でステージ削除
		// @Summary Delete memo (staged or permanent)
		// @Description Delete memo. If permanent=true, memo is permanently deleted. Otherwise, staged deletion.
		// @Tags memos
		// @Param id path int true "Memo ID"
		// @Param permanent query bool false "If true, permanently delete"
		// @Success 200 {object} models.Memo
		// @Failure 400,404,500 {object} ErrorResponse
		// @Security BearerAuth
		// @Router /api/memos/{id} [delete]
		memos.DELETE(":id", memoHandler.DeleteMemo)

		// アーカイブ関連の操作
		// memos.GET("/archive", memoHandler.ListArchivedMemos) // GET /api/memos/archive (archivedのみ) - TODO: 実装が必要
		memos.PATCH(":id/archive", memoHandler.ArchiveMemo) // PATCH /api/memos/:id/archive
		memos.PATCH(":id/restore", memoHandler.RestoreMemo) // PATCH /api/memos/:id/restore

		// 検索機能
		memos.GET("/search", memoHandler.SearchMemos) // GET /api/memos/search
	}
	// ルート登録直後にルート一覧をlogrusで出力
	for _, route := range r.Routes() {
		logger.Log.Infof("[ROUTE] %s %s -> %s", route.Method, route.Path, route.Handler)
	}
	logger.Log.Infof("[ROUTE] 全体: %+v", r.Routes())
}
