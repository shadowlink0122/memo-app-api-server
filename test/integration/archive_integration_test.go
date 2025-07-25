package integration

import (
	"context"
	"encoding/json"
	"fmt"
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

// archiveMockRepository は、アーカイブテスト用のモックリポジトリ
type archiveMockRepository struct {
	userMemos map[int]map[int]*domain.Memo
	nextID    int
}

func (m *archiveMockRepository) CreateMemo(ctx context.Context, userID int, memo *domain.Memo) error {
	if m.userMemos[userID] == nil {
		m.userMemos[userID] = make(map[int]*domain.Memo)
	}
	memo.ID = m.nextID
	m.nextID++
	m.userMemos[userID][memo.ID] = memo
	return nil
}

func (m *archiveMockRepository) GetMemoByID(ctx context.Context, userID, id int) (*domain.Memo, error) {
	if userMemos, exists := m.userMemos[userID]; exists {
		if memo, exists := userMemos[id]; exists {
			return memo, nil
		}
	}
	return nil, fmt.Errorf("memo not found")
}

func (m *archiveMockRepository) ListMemos(ctx context.Context, userID int) ([]*domain.Memo, error) {
	var memos []*domain.Memo
	if userMemos, exists := m.userMemos[userID]; exists {
		for _, memo := range userMemos {
			memos = append(memos, memo)
		}
	}
	return memos, nil
}

func (m *archiveMockRepository) UpdateMemo(ctx context.Context, userID int, memo *domain.Memo) error {
	if userMemos, exists := m.userMemos[userID]; exists {
		if _, exists := userMemos[memo.ID]; exists {
			userMemos[memo.ID] = memo
			return nil
		}
	}
	return fmt.Errorf("memo not found")
}

func (m *archiveMockRepository) DeleteMemo(ctx context.Context, userID, id int) error {
	if userMemos, exists := m.userMemos[userID]; exists {
		if _, exists := userMemos[id]; exists {
			delete(userMemos, id)
			return nil
		}
	}
	return fmt.Errorf("memo not found")
}

func (m *archiveMockRepository) SearchMemos(ctx context.Context, userID int, searchTerm string) ([]*domain.Memo, error) {
	var memos []*domain.Memo
	if userMemos, exists := m.userMemos[userID]; exists {
		for _, memo := range userMemos {
			memos = append(memos, memo)
		}
	}
	return memos, nil
}

func (m *archiveMockRepository) GetArchivedMemos(ctx context.Context, userID int) ([]*domain.Memo, error) {
	var memos []*domain.Memo
	if userMemos, exists := m.userMemos[userID]; exists {
		for _, memo := range userMemos {
			if memo.Status == "archived" {
				memos = append(memos, memo)
			}
		}
	}
	return memos, nil
}

func (m *archiveMockRepository) GetUserMemos(ctx context.Context, userID int) ([]*domain.Memo, error) {
	return m.ListMemos(ctx, userID)
}

// domain.MemoRepositoryインターフェースを実装するためのメソッド群
func (m *archiveMockRepository) Create(ctx context.Context, memo *domain.Memo) (*domain.Memo, error) {
	if m.userMemos[memo.UserID] == nil {
		m.userMemos[memo.UserID] = make(map[int]*domain.Memo)
	}
	memo.ID = m.nextID
	m.nextID++

	// デフォルトでアクティブなステータスを設定
	if memo.Status == "" {
		memo.Status = "active"
	}

	m.userMemos[memo.UserID][memo.ID] = memo
	return memo, nil
}

func (m *archiveMockRepository) GetByID(ctx context.Context, id int, userID int) (*domain.Memo, error) {
	return m.GetMemoByID(ctx, userID, id)
}

func (m *archiveMockRepository) List(ctx context.Context, userID int, filter domain.MemoFilter) ([]domain.Memo, int, error) {
	memos, err := m.ListMemos(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	// フィルターに基づいてメモをフィルタリング
	var filteredMemos []domain.Memo
	for _, memo := range memos {
		// ステータスフィルター
		if filter.Status != "" {
			if string(memo.Status) != string(filter.Status) {
				continue
			}
		} else {
			// ステータスが指定されていない場合は、アクティブなメモのみを返す
			if memo.Status != "active" && memo.Status != "" {
				continue
			}
		}

		filteredMemos = append(filteredMemos, *memo)
	}
	return filteredMemos, len(filteredMemos), nil
}

func (m *archiveMockRepository) Update(ctx context.Context, id int, userID int, memo *domain.Memo) (*domain.Memo, error) {
	memo.ID = id
	return memo, m.UpdateMemo(ctx, userID, memo)
}

func (m *archiveMockRepository) Delete(ctx context.Context, id int, userID int) error {
	return m.DeleteMemo(ctx, userID, id)
}

func (m *archiveMockRepository) PermanentDelete(ctx context.Context, id int, userID int) error {
	return m.DeleteMemo(ctx, userID, id)
}

func (m *archiveMockRepository) Archive(ctx context.Context, id int, userID int) error {
	if userMemos, exists := m.userMemos[userID]; exists {
		if memo, exists := userMemos[id]; exists {
			memo.Status = "archived"
			return nil
		}
	}
	return fmt.Errorf("memo not found")
}

func (m *archiveMockRepository) Restore(ctx context.Context, id int, userID int) error {
	if userMemos, exists := m.userMemos[userID]; exists {
		if memo, exists := userMemos[id]; exists {
			memo.Status = "active"
			return nil
		}
	}
	return fmt.Errorf("memo not found")
}

func (m *archiveMockRepository) Search(ctx context.Context, userID int, query string, filter domain.MemoFilter) ([]domain.Memo, int, error) {
	memos, err := m.SearchMemos(ctx, userID, query)
	if err != nil {
		return nil, 0, err
	}

	// フィルターに基づいてメモをフィルタリング
	var filteredMemos []domain.Memo
	for _, memo := range memos {
		// ステータスフィルターが明示的に指定されている場合のみフィルタリング
		if filter.Status != "" {
			if string(memo.Status) != string(filter.Status) {
				continue
			}
		}
		// ステータスが指定されていない場合は、すべてのメモ（アクティブとアーカイブ両方）を含める

		filteredMemos = append(filteredMemos, *memo)
	}
	return filteredMemos, len(filteredMemos), nil
}

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

	// Ginのテストモードを設定
	gin.SetMode(gin.TestMode)
	s.router = gin.New()
	s.router.Use(middleware.LoggerMiddleware())
	s.router.Use(middleware.CORSMiddleware())
	s.setupTestRoutes()
}

func (s *ArchiveTestSuite) SetupTest() {
	// 各テスト前にデータをクリーンアップ（モックを再初期化）
	s.memoRepo = &archiveMockRepository{
		userMemos: make(map[int]map[int]*domain.Memo),
		nextID:    1,
	}
	s.memoUsecase = usecase.NewMemoUsecase(s.memoRepo)
	s.memoHandler = handler.NewMemoHandler(s.memoUsecase, s.testLogger)

	// ルートも再設定
	s.router = gin.New()
	s.router.Use(middleware.LoggerMiddleware())
	s.router.Use(middleware.CORSMiddleware())
	s.setupTestRoutes()
}

// テスト専用のルート設定
func (s *ArchiveTestSuite) setupTestRoutes() {
	// パブリックルートのグループ化
	api := s.router.Group("/api")

	// 認証付きでメモAPIルートを設定
	memos := api.Group("/memos")
	memos.Use(s.mockAuthMiddleware())
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

// モック認証ミドルウェア - テスト用
func (s *ArchiveTestSuite) mockAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// テスト用の固定ユーザーID
		c.Set("user_id", 1)
		c.Next()
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
	activeMemo, err := s.memoUsecase.CreateMemo(s.ctx, 1, activeReq)
	s.Require().NoError(err)
	s.Require().NotNil(activeMemo)

	// 2. アーカイブするメモを作成
	archiveReq := usecase.CreateMemoRequest{
		Title:    "Archive Memo",
		Content:  "This memo will be archived",
		Category: "test",
		Priority: "medium",
	}
	archiveMemo, err := s.memoUsecase.CreateMemo(s.ctx, 1, archiveReq)
	s.Require().NoError(err)
	s.Require().NotNil(archiveMemo)

	// 3. メモをアーカイブ
	err = s.memoUsecase.ArchiveMemo(s.ctx, 1, archiveMemo.ID)
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
	activeMemo, err := s.memoUsecase.CreateMemo(s.ctx, 1, activeReq)
	s.Require().NoError(err)

	// 2. アーカイブするメモを作成
	archiveReq := usecase.CreateMemoRequest{
		Title:    "Archive Memo for Archive Test",
		Content:  "This memo will be archived",
		Category: "test",
		Priority: "medium",
	}
	archiveMemo, err := s.memoUsecase.CreateMemo(s.ctx, 1, archiveReq)
	s.Require().NoError(err)

	// 3. メモをアーカイブ
	err = s.memoUsecase.ArchiveMemo(s.ctx, 1, archiveMemo.ID)
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

func (s *ArchiveTestSuite) TestSearchIncludesArchivedMemos() {
	// 1. 検索対象のアクティブメモを作成
	activeReq := usecase.CreateMemoRequest{
		Title:    "Searchable Active Memo",
		Content:  "This active memo contains searchable content",
		Category: "test",
		Priority: "medium",
	}
	activeMemo, err := s.memoUsecase.CreateMemo(s.ctx, 1, activeReq)
	s.Require().NoError(err)

	// 2. 検索対象のアーカイブメモを作成
	archiveReq := usecase.CreateMemoRequest{
		Title:    "Searchable Archive Memo",
		Content:  "This archived memo contains searchable content",
		Category: "test",
		Priority: "medium",
	}
	archiveMemo, err := s.memoUsecase.CreateMemo(s.ctx, 1, archiveReq)
	s.Require().NoError(err)

	// 3. メモをアーカイブ
	err = s.memoUsecase.ArchiveMemo(s.ctx, 1, archiveMemo.ID)
	s.Require().NoError(err)

	// 4. 検索でアーカイブされたメモも含まれることを確認
	req := httptest.NewRequest(http.MethodGet, "/api/memos/search?search=searchable", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	var response handler.MemoListResponseDTO
	err = json.Unmarshal(w.Body.Bytes(), &response)
	s.Require().NoError(err)

	// アクティブなメモとアーカイブされたメモの両方が検索結果に含まれていることを確認
	foundActive := false
	foundArchived := false
	for _, memo := range response.Memos {
		if memo.ID == activeMemo.ID {
			foundActive = true
			s.Equal("active", memo.Status)
		}
		if memo.ID == archiveMemo.ID {
			foundArchived = true
			s.Equal("archived", memo.Status)
		}
	}

	s.True(foundActive, "検索結果にアクティブなメモが含まれるべき")
	s.True(foundArchived, "検索結果にアーカイブされたメモも含まれるべき")
}

func (s *ArchiveTestSuite) TestSearchWithStatusFilterOnlyArchivedMemos() {
	// 1. 検索対象のアクティブメモを作成
	activeReq := usecase.CreateMemoRequest{
		Title:    "Searchable Active Memo",
		Content:  "This active memo contains searchable content",
		Category: "test",
		Priority: "medium",
	}
	activeMemo, err := s.memoUsecase.CreateMemo(s.ctx, 1, activeReq)
	s.Require().NoError(err)

	// 2. 検索対象のアーカイブメモを作成
	archiveReq := usecase.CreateMemoRequest{
		Title:    "Searchable Archive Memo",
		Content:  "This archived memo contains searchable content",
		Category: "test",
		Priority: "medium",
	}
	archiveMemo, err := s.memoUsecase.CreateMemo(s.ctx, 1, archiveReq)
	s.Require().NoError(err)

	// 3. メモをアーカイブ
	err = s.memoUsecase.ArchiveMemo(s.ctx, 1, archiveMemo.ID)
	s.Require().NoError(err)

	// 4. ステータスフィルターでアーカイブのみを検索
	req := httptest.NewRequest(http.MethodGet, "/api/memos/search?search=searchable&status=archived", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	var response handler.MemoListResponseDTO
	err = json.Unmarshal(w.Body.Bytes(), &response)
	s.Require().NoError(err)

	// アーカイブされたメモのみが検索結果に含まれていることを確認
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

	s.False(foundActive, "ステータスフィルターでアクティブなメモは除外されるべき")
	s.True(foundArchived, "ステータスフィルターでアーカイブされたメモが含まれるべき")
}

func (s *ArchiveTestSuite) TestIndividualMemoAccessStillWorks() {
	// 1. メモを作成
	req := usecase.CreateMemoRequest{
		Title:    "Individual Access Test Memo",
		Content:  "This memo tests individual access",
		Category: "test",
		Priority: "medium",
	}
	memo, err := s.memoUsecase.CreateMemo(s.ctx, 1, req)
	s.Require().NoError(err)

	// 2. メモをアーカイブ
	err = s.memoUsecase.ArchiveMemo(s.ctx, 1, memo.ID)
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
