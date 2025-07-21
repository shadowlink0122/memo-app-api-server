package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"memo-app/src/handlers"
	"memo-app/src/logger"
	"memo-app/src/middleware"
	"memo-app/src/models"
	"memo-app/src/repository"
	"memo-app/src/service"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type MemoIntegrationTestSuite struct {
	suite.Suite
	router      *gin.Engine
	memoRepo    repository.MemoRepositoryInterface
	memoSvc     service.MemoServiceInterface
	memoHandler *handlers.MemoHandler
	testLogger  *logrus.Logger
}

func (suite *MemoIntegrationTestSuite) SetupSuite() {
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
	require.NoError(suite.T(), err)

	// テスト用ロガーを作成
	suite.testLogger = logrus.New()
	suite.testLogger.SetLevel(logrus.InfoLevel)

	// モックコンポーネントをセットアップ
	suite.setupMockComponents()

	// Ginルーターを設定
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()

	// ミドルウェアを適用
	suite.router.Use(middleware.LoggerMiddleware())
	suite.router.Use(middleware.CORSMiddleware())

	// ルートを設定（認証ミドルウェアなし）
	suite.setupTestRoutes()
}

func (suite *MemoIntegrationTestSuite) TearDownSuite() {
	logger.CloseLogger()
}

func (suite *MemoIntegrationTestSuite) SetupTest() {
	// 各テスト前にデータをクリーンアップ（モックを再初期化）
	suite.memoRepo = &mockMemoRepository{
		memos:  make(map[int]*models.Memo),
		nextID: 1,
	}
	suite.memoSvc = service.NewMemoService(suite.memoRepo, suite.testLogger)
	suite.memoHandler = handlers.NewMemoHandler(suite.memoSvc, suite.testLogger)

	// ルートも再設定
	suite.router = gin.New()
	suite.router.Use(middleware.LoggerMiddleware())
	suite.router.Use(middleware.CORSMiddleware())
	suite.setupTestRoutes()
}

func (suite *MemoIntegrationTestSuite) setupMockComponents() {
	// モックリポジトリを作成
	suite.memoRepo = &mockMemoRepository{
		memos:  make(map[int]*models.Memo),
		nextID: 1,
	}

	suite.memoSvc = service.NewMemoService(suite.memoRepo, suite.testLogger)
	suite.memoHandler = handlers.NewMemoHandler(suite.memoSvc, suite.testLogger)
}

// テスト専用のルート設定（認証ミドルウェアなし）
func (suite *MemoIntegrationTestSuite) setupTestRoutes() {
	// パブリックルートのグループ化
	api := suite.router.Group("/api")

	// 認証なしでメモAPIルートを設定
	memos := api.Group("/memos")
	{
		// メモの基本CRUD操作
		memos.POST("", suite.memoHandler.CreateMemo)       // POST /api/memos
		memos.GET("", suite.memoHandler.ListMemos)         // GET /api/memos
		memos.GET("/:id", suite.memoHandler.GetMemo)       // GET /api/memos/:id
		memos.PUT("/:id", suite.memoHandler.UpdateMemo)    // PUT /api/memos/:id
		memos.DELETE("/:id", suite.memoHandler.DeleteMemo) // DELETE /api/memos/:id
	}
}

// モックリポジトリの実装
type mockMemoRepository struct {
	memos  map[int]*models.Memo
	nextID int
}

func (m *mockMemoRepository) Create(ctx context.Context, req *models.CreateMemoRequest) (*models.Memo, error) {
	memo := &models.Memo{
		ID:        m.nextID,
		Title:     req.Title,
		Content:   req.Content,
		Category:  req.Category,
		Priority:  req.Priority,
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if req.Tags != nil {
		// タグを文字列に変換（実際の実装に合わせる）
		tagsBytes, _ := json.Marshal(req.Tags)
		memo.Tags = string(tagsBytes)
	}
	m.memos[m.nextID] = memo
	m.nextID++
	return memo, nil
}

func (m *mockMemoRepository) GetByID(ctx context.Context, id int) (*models.Memo, error) {
	memo, exists := m.memos[id]
	if !exists {
		return nil, fmt.Errorf("memo not found")
	}
	return memo, nil
}

func (m *mockMemoRepository) List(ctx context.Context, filter *models.MemoFilter) (*models.MemoListResponse, error) {
	var memos []models.Memo
	for _, memo := range m.memos {
		memos = append(memos, *memo)
	}

	// フィルターの適用は簡略化
	return &models.MemoListResponse{
		Memos:      memos,
		Total:      len(memos),
		Page:       1,
		Limit:      len(memos),
		TotalPages: 1,
	}, nil
}

func (m *mockMemoRepository) Update(ctx context.Context, id int, req *models.UpdateMemoRequest) (*models.Memo, error) {
	memo, exists := m.memos[id]
	if !exists {
		return nil, fmt.Errorf("memo not found")
	}

	if req.Title != nil {
		memo.Title = *req.Title
	}
	if req.Content != nil {
		memo.Content = *req.Content
	}
	if req.Category != nil {
		memo.Category = *req.Category
	}
	if req.Priority != nil {
		memo.Priority = *req.Priority
	}
	if req.Status != nil {
		memo.Status = *req.Status
	}
	if req.Tags != nil {
		tagsBytes, _ := json.Marshal(req.Tags)
		memo.Tags = string(tagsBytes)
	}

	memo.UpdatedAt = time.Now()
	return memo, nil
}

func (m *mockMemoRepository) Delete(ctx context.Context, id int) error {
	_, exists := m.memos[id]
	if !exists {
		return fmt.Errorf("memo not found")
	}
	delete(m.memos, id)
	return nil
}

// メモ一覧取得のテスト
func (suite *MemoIntegrationTestSuite) TestGetMemos() {
	// テストデータを準備
	ctx := context.Background()
	suite.memoRepo.Create(ctx, &models.CreateMemoRequest{
		Title:   "テストメモ1",
		Content: "これはテストメモ1の内容です",
	})
	suite.memoRepo.Create(ctx, &models.CreateMemoRequest{
		Title:   "テストメモ2",
		Content: "これはテストメモ2の内容です",
	})

	// リクエストを作成
	req, err := http.NewRequest("GET", "/api/memos", nil)
	require.NoError(suite.T(), err)

	// レスポンスを記録
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// ステータスコードを確認
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// レスポンスボディを解析（MemoListResponse）
	var response models.MemoListResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	// レスポンスの内容を確認
	assert.Len(suite.T(), response.Memos, 2)
	assert.Equal(suite.T(), 2, response.Total)
}

// メモ作成のテスト
func (suite *MemoIntegrationTestSuite) TestCreateMemo() {
	// リクエストボディを準備
	requestBody := map[string]interface{}{
		"title":    "新しいメモ",
		"content":  "これは新しいメモの内容です",
		"category": "テスト",
		"priority": "medium",
		"tags":     []string{"テスト", "統合テスト"},
	}
	jsonBody, err := json.Marshal(requestBody)
	require.NoError(suite.T(), err)

	// リクエストを作成
	req, err := http.NewRequest("POST", "/api/memos", bytes.NewBuffer(jsonBody))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	// レスポンスを記録
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// ステータスコードを確認
	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	// レスポンスボディを解析
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	// レスポンスの内容を確認（直接メモオブジェクト）
	var memo map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &memo)
	require.NoError(suite.T(), err)

	// メモの内容を確認
	assert.Equal(suite.T(), "新しいメモ", memo["title"])
	assert.Equal(suite.T(), "これは新しいメモの内容です", memo["content"])
	assert.Equal(suite.T(), "テスト", memo["category"])
	assert.Equal(suite.T(), "medium", memo["priority"])
	assert.Contains(suite.T(), memo, "id")
}

// メモ詳細取得のテスト
func (suite *MemoIntegrationTestSuite) TestGetMemoByID() {
	// テストデータを準備
	ctx := context.Background()
	memo, err := suite.memoRepo.Create(ctx, &models.CreateMemoRequest{
		Title:   "テストメモ",
		Content: "これはテストメモの内容です",
	})
	require.NoError(suite.T(), err)

	// リクエストを作成
	req, err := http.NewRequest("GET", "/api/memos/"+strconv.Itoa(memo.ID), nil)
	require.NoError(suite.T(), err)

	// レスポンスを記録
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// ステータスコードを確認
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// レスポンスボディを解析（Memo）
	var response models.Memo
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	// レスポンスの内容を確認
	assert.Equal(suite.T(), memo.ID, response.ID)
	assert.Equal(suite.T(), "テストメモ", response.Title)
	assert.Equal(suite.T(), "これはテストメモの内容です", response.Content)
}

// メモ更新のテスト
func (suite *MemoIntegrationTestSuite) TestUpdateMemo() {
	// テストデータを準備
	ctx := context.Background()
	memo, err := suite.memoRepo.Create(ctx, &models.CreateMemoRequest{
		Title:   "元のタイトル",
		Content: "元の内容",
	})
	require.NoError(suite.T(), err)

	// リクエストボディを準備
	requestBody := map[string]string{
		"title":   "更新されたタイトル",
		"content": "更新された内容",
	}
	jsonBody, err := json.Marshal(requestBody)
	require.NoError(suite.T(), err)

	// リクエストを作成
	req, err := http.NewRequest("PUT", "/api/memos/"+strconv.Itoa(memo.ID), bytes.NewBuffer(jsonBody))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	// レスポンスを記録
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// ステータスコードを確認
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// レスポンスボディを解析（Memo）
	var response models.Memo
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	// レスポンスの内容を確認
	assert.Equal(suite.T(), "更新されたタイトル", response.Title)
	assert.Equal(suite.T(), "更新された内容", response.Content)
}

// メモ削除のテスト
func (suite *MemoIntegrationTestSuite) TestDeleteMemo() {
	// テストデータを準備
	ctx := context.Background()
	memo, err := suite.memoRepo.Create(ctx, &models.CreateMemoRequest{
		Title:   "削除対象メモ",
		Content: "削除される内容",
	})
	require.NoError(suite.T(), err)

	// リクエストを作成
	req, err := http.NewRequest("DELETE", "/api/memos/"+strconv.Itoa(memo.ID), nil)
	require.NoError(suite.T(), err)

	// レスポンスを記録
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// ステータスコードを確認（204 No Content）
	assert.Equal(suite.T(), http.StatusNoContent, w.Code)

	// メモが実際に削除されていることを確認
	getReq, err := http.NewRequest("GET", "/api/memos/"+strconv.Itoa(memo.ID), nil)
	require.NoError(suite.T(), err)

	getW := httptest.NewRecorder()
	suite.router.ServeHTTP(getW, getReq)

	assert.Equal(suite.T(), http.StatusNotFound, getW.Code)
}

// 存在しないメモの取得テスト
func (suite *MemoIntegrationTestSuite) TestGetNonExistentMemo() {
	// 存在しないIDでリクエスト
	req, err := http.NewRequest("GET", "/api/memos/999", nil)
	require.NoError(suite.T(), err)

	// レスポンスを記録
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// ステータスコードを確認
	assert.Equal(suite.T(), http.StatusNotFound, w.Code)

	// レスポンスボディを解析
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	// レスポンスの内容を確認
	assert.Contains(suite.T(), response["error"], "not found")
}

// 無効なJSONでのメモ作成テスト
func (suite *MemoIntegrationTestSuite) TestCreateMemoInvalidJSON() {
	// 無効なJSONでリクエスト
	req, err := http.NewRequest("POST", "/api/memos", bytes.NewBufferString("invalid json"))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	// レスポンスを記録
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// ステータスコードを確認
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

// バリデーションエラーのテスト
func (suite *MemoIntegrationTestSuite) TestCreateMemoValidationError() {
	// 空のタイトルでリクエスト
	requestBody := map[string]string{
		"title":   "",
		"content": "内容だけあります",
	}
	jsonBody, err := json.Marshal(requestBody)
	require.NoError(suite.T(), err)

	req, err := http.NewRequest("POST", "/api/memos", bytes.NewBuffer(jsonBody))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	// レスポンスを記録
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// ステータスコードを確認
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

func TestMemoIntegrationSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("短いテストモードで統合テストをスキップ")
	}

	suite.Run(t, new(MemoIntegrationTestSuite))
}
