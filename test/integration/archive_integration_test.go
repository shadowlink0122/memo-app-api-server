package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"memo-app/src/domain"
	"memo-app/src/interface/handler"
	"memo-app/src/logger"
	"memo-app/src/middleware"
	"memo-app/src/usecase"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
)

type ArchiveTestSuite struct {
	suite.Suite
	ctx         context.Context
	router      *gin.Engine
	memoRepo    domain.MemoRepository
	memoUsecase usecase.MemoUsecase
	memoHandler *handler.MemoHandler
	testLogger  *logrus.Logger
}

func (s *ArchiveTestSuite) SetupSuite() {
	// テスト用の環境変数を設定
	os.Setenv("LOG_DIRECTORY", "../../logs/test")
	os.Setenv("LOG_MAX_SIZE", "1")
	os.Setenv("LOG_MAX_BACKUPS", "3")
	os.Setenv("LOG_MAX_AGE", "1")
	os.Setenv("LOG_COMPRESS", "false")
	os.Setenv("RATE_LIMIT_RPS", "100")
	os.Setenv("RATE_LIMIT_BURST", "200")

	// ロガーを初期化
	err := logger.InitLogger()
	s.Require().NoError(err)

	s.ctx = context.Background()
	s.testLogger = logger.Log
	s.setupMockComponents()

	// Ginのテストモードを設定
	gin.SetMode(gin.TestMode)
	s.router = gin.New()
	s.router.Use(middleware.LoggerMiddleware())
	s.router.Use(middleware.CORSMiddleware())
	s.setupTestRoutes()
}

func (s *ArchiveTestSuite) SetupTest() {
	// 各テスト前にデータをクリーンアップ（モックを再初期化）
	s.memoRepo = &mockMemoRepository{
		memos:  make(map[int]*domain.Memo),
		nextID: 1,
	}
	s.memoUsecase = usecase.NewMemoUsecase(s.memoRepo)
	s.memoHandler = handler.NewMemoHandler(s.memoUsecase, s.testLogger)

	// ルートも再設定
	s.router = gin.New()
	s.router.Use(middleware.LoggerMiddleware())
	s.router.Use(middleware.CORSMiddleware())
	s.setupTestRoutes()
}

func (s *ArchiveTestSuite) setupMockComponents() {
	// モックリポジトリを作成
	s.memoRepo = &mockMemoRepository{
		memos:  make(map[int]*domain.Memo),
		nextID: 1,
	}

	s.memoUsecase = usecase.NewMemoUsecase(s.memoRepo)
	s.memoHandler = handler.NewMemoHandler(s.memoUsecase, s.testLogger)
}

// テスト専用のルート設定
func (s *ArchiveTestSuite) setupTestRoutes() {
	// パブリックルートのグループ化
	api := s.router.Group("/api")

	// 認証なしでメモAPIルートを設定
	memos := api.Group("/memos")
	{
		memos.POST("", s.memoHandler.CreateMemo)
		memos.GET("", s.memoHandler.ListMemos)
		memos.GET("/archive", s.memoHandler.ListArchivedMemos)
		memos.GET("/:id", s.memoHandler.GetMemo)
		memos.PUT("/:id", s.memoHandler.UpdateMemo)
		memos.DELETE("/:id", s.memoHandler.DeleteMemo)
		memos.PATCH("/:id/archive", s.memoHandler.ArchiveMemo)
		memos.PATCH("/:id/restore", s.memoHandler.RestoreMemo)
		memos.DELETE("/:id/permanent", s.memoHandler.PermanentDeleteMemo)
		memos.GET("/search", s.memoHandler.SearchMemos)
	}
}

func (s *ArchiveTestSuite) TestArchiveExclusionFromRegularList() {
	// 1. アクティブなメモを作成
	activeReq := usecase.CreateMemoRequest{
		Title:    "Active Memo",
		Content:  "This is an active memo",
		Category: "test",
		Priority: "medium",
	}
	activeMemo, err := s.memoUsecase.CreateMemo(s.ctx, activeReq)
	s.Require().NoError(err)
	s.Require().NotNil(activeMemo)

	// 2. アーカイブするメモを作成
	archiveReq := usecase.CreateMemoRequest{
		Title:    "Archive Memo",
		Content:  "This memo will be archived",
		Category: "test",
		Priority: "medium",
	}
	archiveMemo, err := s.memoUsecase.CreateMemo(s.ctx, archiveReq)
	s.Require().NoError(err)
	s.Require().NotNil(archiveMemo)

	// 3. メモをアーカイブ
	err = s.memoUsecase.ArchiveMemo(s.ctx, archiveMemo.ID)
	s.Require().NoError(err)

	// 4. 通常のメモ一覧でアーカイブされたメモが表示されないことを確認
	req := httptest.NewRequest(http.MethodGet, "/api/memos", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	var response handler.MemoListResponseDTO
	err = json.Unmarshal(w.Body.Bytes(), &response)
	s.Require().NoError(err)

	// アクティブなメモだけが含まれていることを確認
	foundActive := false
	foundArchived := false
	for _, memo := range response.Memos {
		if memo.ID == activeMemo.ID {
			foundActive = true
			s.Equal("active", memo.Status)
		}
		if memo.ID == archiveMemo.ID {
			foundArchived = true
		}
	}

	s.True(foundActive, "アクティブなメモが表示されるべき")
	s.False(foundArchived, "アーカイブされたメモは表示されるべきではない")
}

func (s *ArchiveTestSuite) TestArchiveEndpointShowsOnlyArchivedMemos() {
	// 1. アクティブなメモを作成
	activeReq := usecase.CreateMemoRequest{
		Title:    "Active Memo for Archive Test",
		Content:  "This is an active memo",
		Category: "test",
		Priority: "medium",
	}
	activeMemo, err := s.memoUsecase.CreateMemo(s.ctx, activeReq)
	s.Require().NoError(err)

	// 2. アーカイブするメモを作成
	archiveReq := usecase.CreateMemoRequest{
		Title:    "Archive Memo for Archive Test",
		Content:  "This memo will be archived",
		Category: "test",
		Priority: "medium",
	}
	archiveMemo, err := s.memoUsecase.CreateMemo(s.ctx, archiveReq)
	s.Require().NoError(err)

	// 3. メモをアーカイブ
	err = s.memoUsecase.ArchiveMemo(s.ctx, archiveMemo.ID)
	s.Require().NoError(err)

	// 4. アーカイブエンドポイントでアーカイブされたメモのみが表示されることを確認
	req := httptest.NewRequest(http.MethodGet, "/api/memos/archive", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	var response handler.MemoListResponseDTO
	err = json.Unmarshal(w.Body.Bytes(), &response)
	s.Require().NoError(err)

	// アーカイブされたメモだけが含まれていることを確認
	foundActive := false
	foundArchived := false
	for _, memo := range response.Memos {
		if memo.ID == activeMemo.ID {
			foundActive = true
		}
		if memo.ID == archiveMemo.ID {
			foundArchived = true
			s.Equal("archived", memo.Status)
		}
	}

	s.False(foundActive, "アクティブなメモはアーカイブエンドポイントに表示されるべきではない")
	s.True(foundArchived, "アーカイブされたメモが表示されるべき")
}

func (s *ArchiveTestSuite) TestSearchExcludesArchivedMemos() {
	// 1. 検索対象のアクティブメモを作成
	activeReq := usecase.CreateMemoRequest{
		Title:    "Searchable Active Memo",
		Content:  "This active memo contains searchable content",
		Category: "test",
		Priority: "medium",
	}
	activeMemo, err := s.memoUsecase.CreateMemo(s.ctx, activeReq)
	s.Require().NoError(err)

	// 2. 検索対象のアーカイブメモを作成
	archiveReq := usecase.CreateMemoRequest{
		Title:    "Searchable Archive Memo",
		Content:  "This archived memo contains searchable content",
		Category: "test",
		Priority: "medium",
	}
	archiveMemo, err := s.memoUsecase.CreateMemo(s.ctx, archiveReq)
	s.Require().NoError(err)

	// 3. メモをアーカイブ
	err = s.memoUsecase.ArchiveMemo(s.ctx, archiveMemo.ID)
	s.Require().NoError(err)

	// 4. 検索でアーカイブされたメモが除外されることを確認
	req := httptest.NewRequest(http.MethodGet, "/api/memos/search?search=searchable", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	var response handler.MemoListResponseDTO
	err = json.Unmarshal(w.Body.Bytes(), &response)
	s.Require().NoError(err)

	// アクティブなメモだけが検索結果に含まれていることを確認
	foundActive := false
	foundArchived := false
	for _, memo := range response.Memos {
		if memo.ID == activeMemo.ID {
			foundActive = true
			s.Equal("active", memo.Status)
		}
		if memo.ID == archiveMemo.ID {
			foundArchived = true
		}
	}

	s.True(foundActive, "検索結果にアクティブなメモが含まれるべき")
	s.False(foundArchived, "検索結果にアーカイブされたメモは含まれるべきではない")
}

func (s *ArchiveTestSuite) TestIndividualMemoAccessStillWorks() {
	// 1. メモを作成
	req := usecase.CreateMemoRequest{
		Title:    "Individual Access Test Memo",
		Content:  "This memo tests individual access",
		Category: "test",
		Priority: "medium",
	}
	memo, err := s.memoUsecase.CreateMemo(s.ctx, req)
	s.Require().NoError(err)

	// 2. メモをアーカイブ
	err = s.memoUsecase.ArchiveMemo(s.ctx, memo.ID)
	s.Require().NoError(err)

	// 3. 個別IDでのアクセスは依然として可能であることを確認
	reqHTTP := httptest.NewRequest(http.MethodGet, "/api/memos/"+string(rune(memo.ID+'0')), nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, reqHTTP)

	// 個別アクセスは成功するべき（アーカイブ後でも）
	s.Equal(http.StatusOK, w.Code)

	var response handler.MemoResponseDTO
	err = json.Unmarshal(w.Body.Bytes(), &response)
	s.Require().NoError(err)
	s.Equal(memo.ID, response.ID)
	s.Equal("archived", response.Status)
}

func TestArchiveTestSuite(t *testing.T) {
	suite.Run(t, new(ArchiveTestSuite))
}
