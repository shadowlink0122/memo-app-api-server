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
	"strings"
	"sync"
	"testing"
	"time"

	"memo-app/src/domain"
	"memo-app/src/interface/handler"
	"memo-app/src/logger"
	"memo-app/src/middleware"
	"memo-app/src/usecase"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type MemoIntegrationTestSuite struct {
	suite.Suite
	router      *gin.Engine
	memoRepo    domain.MemoRepository
	memoUsecase usecase.MemoUsecase
	memoHandler *handler.MemoHandler
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
	if err != nil {
		suite.T().Fatalf("Failed to initialize logger: %v", err)
	}

	suite.testLogger = logger.Log
	suite.setupMockComponents()

	// Ginのテストモードを設定
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()
	suite.router.Use(middleware.LoggerMiddleware())
	suite.router.Use(middleware.CORSMiddleware())
	suite.setupTestRoutes()
}

func (suite *MemoIntegrationTestSuite) SetupTest() {
	// 各テスト前にデータをクリーンアップ（モックを再初期化）
	suite.memoRepo = &memoIntegrationMockRepository{
		userMemos: make(map[int]map[int]*domain.Memo),
		nextID:    1,
	}
	suite.memoUsecase = usecase.NewMemoUsecase(suite.memoRepo)
	suite.memoHandler = handler.NewMemoHandler(suite.memoUsecase, suite.testLogger)

	// ルートも再設定
	suite.router = gin.New()
	suite.router.Use(middleware.LoggerMiddleware())
	suite.router.Use(middleware.CORSMiddleware())
	suite.setupTestRoutes()
}

func (suite *MemoIntegrationTestSuite) setupMockComponents() {
	// モックリポジトリを作成
	suite.memoRepo = &memoIntegrationMockRepository{
		userMemos: make(map[int]map[int]*domain.Memo),
		nextID:    1,
	}

	suite.memoUsecase = usecase.NewMemoUsecase(suite.memoRepo)
	suite.memoHandler = handler.NewMemoHandler(suite.memoUsecase, suite.testLogger)
}

// テスト専用のルート設定（認証ミドルウェアあり）
func (suite *MemoIntegrationTestSuite) setupTestRoutes() {
	// パブリックルートのグループ化
	api := suite.router.Group("/api")

	// 認証付きでメモAPIルートを設定（固定ユーザーID=1）
	memos := api.Group("/memos")
	memos.Use(suite.simpleAuthMiddleware())
	{
		// メモの基本CRUD操作
		memos.POST("", suite.memoHandler.CreateMemo)                          // POST /api/memos
		memos.GET("", suite.memoHandler.ListMemos)                            // GET /api/memos
		memos.GET("/:id", suite.memoHandler.GetMemo)                          // GET /api/memos/:id
		memos.PUT("/:id", suite.memoHandler.UpdateMemo)                       // PUT /api/memos/:id
		memos.DELETE("/:id", suite.memoHandler.DeleteMemo)                    // DELETE /api/memos/:id (段階的削除)
		memos.DELETE("/:id/permanent", suite.memoHandler.PermanentDeleteMemo) // DELETE /api/memos/:id/permanent (完全削除)
	}
}

// モックリポジトリの実装（ユーザー分離対応）
type memoIntegrationMockRepository struct {
	userMemos map[int]map[int]*domain.Memo // userID -> memoID -> memo
	nextID    int
	mutex     sync.RWMutex
}

func (m *memoIntegrationMockRepository) getUserMemos(userID int) map[int]*domain.Memo {
	if _, exists := m.userMemos[userID]; !exists {
		m.userMemos[userID] = make(map[int]*domain.Memo)
	}
	return m.userMemos[userID]
}

// Removed duplicate Create method to resolve redeclaration error.

func (m *memoIntegrationMockRepository) Create(ctx context.Context, memo *domain.Memo) (*domain.Memo, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// userIDが設定されていない場合は、デフォルトでユーザー1を使用
	if memo.UserID == 0 {
		memo.UserID = 1
	}

	memos := m.getUserMemos(memo.UserID)
	memo.ID = m.nextID
	m.nextID++
	memo.CreatedAt = time.Now()
	memo.UpdatedAt = time.Now()

	// デフォルトでアクティブステータスを設定
	if memo.Status == "" {
		memo.Status = domain.StatusActive
	}

	memos[memo.ID] = memo
	return memo, nil
}

func (m *memoIntegrationMockRepository) GetByID(ctx context.Context, id int, userID int) (*domain.Memo, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	memos := m.getUserMemos(userID)
	memo, exists := memos[id]
	if !exists {
		return nil, usecase.ErrMemoNotFound
	}
	return memo, nil
}

func (m *memoIntegrationMockRepository) List(ctx context.Context, userID int, filter domain.MemoFilter) ([]domain.Memo, int, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	memos := m.getUserMemos(userID)
	var result []domain.Memo
	for _, memo := range memos {
		// フィルタリング（status, categoryなどの条件）
		if filter.Status != "" && memo.Status != filter.Status {
			continue
		}
		if filter.Category != "" && memo.Category != filter.Category {
			continue
		}
		result = append(result, *memo)
	}
	return result, len(result), nil
}

func (m *memoIntegrationMockRepository) Update(ctx context.Context, id int, userID int, memo *domain.Memo) (*domain.Memo, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	memos := m.getUserMemos(userID)
	existingMemo, exists := memos[id]
	if !exists {
		return nil, usecase.ErrMemoNotFound
	}

	// 更新フィールドのみを更新
	if memo.Title != "" {
		existingMemo.Title = memo.Title
	}
	if memo.Content != "" {
		existingMemo.Content = memo.Content
	}
	if memo.Category != "" {
		existingMemo.Category = memo.Category
	}
	if memo.Priority != "" {
		existingMemo.Priority = memo.Priority
	}
	if memo.Status != "" {
		existingMemo.Status = memo.Status
	}
	if memo.Tags != nil {
		existingMemo.Tags = memo.Tags
	}

	existingMemo.UpdatedAt = time.Now()
	return existingMemo, nil
}

func (m *memoIntegrationMockRepository) Delete(ctx context.Context, id int, userID int) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	memos := m.getUserMemos(userID)
	memo, exists := memos[id]
	if !exists {
		return usecase.ErrMemoNotFound
	}

	// 段階的削除の実装
	if memo.Status == domain.StatusActive {
		// アクティブメモをアーカイブに移動
		memo.Status = domain.StatusArchived
		memo.UpdatedAt = time.Now()
		completedAt := time.Now()
		memo.CompletedAt = &completedAt
	} else if memo.Status == domain.StatusArchived {
		// アーカイブ済みメモを完全削除
		delete(memos, id)
	}
	return nil
}

func (m *memoIntegrationMockRepository) PermanentDelete(ctx context.Context, id int, userID int) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	memos := m.getUserMemos(userID)
	_, exists := memos[id]
	if !exists {
		return usecase.ErrMemoNotFound
	}
	delete(memos, id)
	return nil
}

func (m *memoIntegrationMockRepository) Archive(ctx context.Context, id int, userID int) (*domain.Memo, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	memos := m.getUserMemos(userID)
	memo, exists := memos[id]
	if !exists {
		return nil, usecase.ErrMemoNotFound
	}
	memo.Status = domain.StatusArchived
	memo.UpdatedAt = time.Now()
	completedAt := time.Now()
	memo.CompletedAt = &completedAt
	return memo, nil
}

func (m *memoIntegrationMockRepository) Restore(ctx context.Context, id int, userID int) (*domain.Memo, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	memos := m.getUserMemos(userID)
	memo, exists := memos[id]
	if !exists {
		return nil, usecase.ErrMemoNotFound
	}
	memo.Status = domain.StatusActive
	memo.UpdatedAt = time.Now()
	memo.CompletedAt = nil
	return memo, nil
}

func (m *memoIntegrationMockRepository) Search(ctx context.Context, userID int, query string, filter domain.MemoFilter) ([]domain.Memo, int, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	memos := m.getUserMemos(userID)
	var result []domain.Memo
	for _, memo := range memos {
		// ステータスフィルタ
		if filter.Status != "" && memo.Status != filter.Status {
			continue
		}

		// 簡単な検索実装（タイトルと内容をチェック）
		if strings.Contains(strings.ToLower(memo.Title), strings.ToLower(query)) ||
			strings.Contains(strings.ToLower(memo.Content), strings.ToLower(query)) {
			result = append(result, *memo)
		}
	}
	return result, len(result), nil
}

// メモ一覧取得のテスト
func (suite *MemoIntegrationTestSuite) TestGetMemos() {
	// テストデータを準備（HTTP経由）
	memo1 := map[string]interface{}{
		"title":   "テストメモ1",
		"content": "これはテストメモです",
	}
	memo2 := map[string]interface{}{
		"title":   "テストメモ2",
		"content": "これは2番目のテストメモです",
	}

	// メモ1作成
	memo1JSON, _ := json.Marshal(memo1)
	req1 := httptest.NewRequest("POST", "/api/memos", bytes.NewReader(memo1JSON))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	suite.router.ServeHTTP(w1, req1)

	// メモ2作成
	memo2JSON, _ := json.Marshal(memo2)
	req2 := httptest.NewRequest("POST", "/api/memos", bytes.NewReader(memo2JSON))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	suite.router.ServeHTTP(w2, req2)

	// メモ一覧を取得
	req := httptest.NewRequest("GET", "/api/memos", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// レスポンスを検証
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response struct {
		Memos []domain.Memo `json:"memos"`
		Total int           `json:"total"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), 2, response.Total)
	assert.Len(suite.T(), response.Memos, 2)
}

// メモ作成のテスト
func (suite *MemoIntegrationTestSuite) TestCreateMemo() {
	// リクエストボディを準備
	requestBody := map[string]interface{}{
		"title":   "新しいメモ",
		"content": "これは新しいメモの内容です",
	}

	jsonData, err := json.Marshal(requestBody)
	require.NoError(suite.T(), err)

	// HTTPリクエストを作成
	req := httptest.NewRequest("POST", "/api/memos", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// レスポンスレコーダーを作成
	w := httptest.NewRecorder()

	// ルーターでリクエストを処理
	suite.router.ServeHTTP(w, req)

	// レスポンスを検証
	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response domain.Memo
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "新しいメモ", response.Title)
	assert.Equal(suite.T(), "これは新しいメモの内容です", response.Content)
	assert.NotZero(suite.T(), response.ID)
}

// メモ更新のテスト
func (suite *MemoIntegrationTestSuite) TestUpdateMemo() {
	// テストメモを作成
	memo := map[string]interface{}{
		"title":   "更新前のメモ",
		"content": "更新前の内容",
	}
	memoJSON, _ := json.Marshal(memo)
	createReq := httptest.NewRequest("POST", "/api/memos", bytes.NewReader(memoJSON))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	suite.router.ServeHTTP(createW, createReq)

	var createdMemo domain.Memo
	json.Unmarshal(createW.Body.Bytes(), &createdMemo)

	// 更新リクエストを準備
	updateBody := map[string]interface{}{
		"title":   "更新後のメモ",
		"content": "更新後の内容",
	}

	jsonData, err := json.Marshal(updateBody)
	require.NoError(suite.T(), err)

	// HTTPリクエストを作成
	req := httptest.NewRequest("PUT", "/api/memos/"+strconv.Itoa(createdMemo.ID), bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// レスポンスレコーダーを作成
	w := httptest.NewRecorder()

	// ルーターでリクエストを処理
	suite.router.ServeHTTP(w, req)

	// レスポンスを検証
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response domain.Memo
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "更新後のメモ", response.Title)
	assert.Equal(suite.T(), "更新後の内容", response.Content)
	assert.Equal(suite.T(), createdMemo.ID, response.ID)
}

// メモ削除のテスト（段階的削除）
func (suite *MemoIntegrationTestSuite) TestDeleteMemo() {
	// テストメモを作成
	memo := map[string]interface{}{
		"title":   "削除テストメモ",
		"content": "削除テスト用の内容",
	}
	memoJSON, _ := json.Marshal(memo)
	createReq := httptest.NewRequest("POST", "/api/memos", bytes.NewReader(memoJSON))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	suite.router.ServeHTTP(createW, createReq)

	var createdMemo domain.Memo
	json.Unmarshal(createW.Body.Bytes(), &createdMemo)

	// 第1段階: アクティブ → アーカイブ
	deleteReq1 := httptest.NewRequest("DELETE", "/api/memos/"+strconv.Itoa(createdMemo.ID), nil)
	deleteW1 := httptest.NewRecorder()
	suite.router.ServeHTTP(deleteW1, deleteReq1)

	assert.Equal(suite.T(), http.StatusNoContent, deleteW1.Code)

	// メモを取得してアーカイブ状態を確認
	getReq1 := httptest.NewRequest("GET", "/api/memos/"+strconv.Itoa(createdMemo.ID), nil)
	getW1 := httptest.NewRecorder()
	suite.router.ServeHTTP(getW1, getReq1)

	var archivedMemo domain.Memo
	json.Unmarshal(getW1.Body.Bytes(), &archivedMemo)
	assert.Equal(suite.T(), domain.StatusArchived, archivedMemo.Status)

	// 第2段階: アーカイブ → 永続削除
	deleteReq2 := httptest.NewRequest("DELETE", "/api/memos/"+strconv.Itoa(createdMemo.ID), nil)
	deleteW2 := httptest.NewRecorder()
	suite.router.ServeHTTP(deleteW2, deleteReq2)

	assert.Equal(suite.T(), http.StatusNoContent, deleteW2.Code)

	// メモが完全に削除されたことを確認
	getReq2 := httptest.NewRequest("GET", "/api/memos/"+strconv.Itoa(createdMemo.ID), nil)
	getW2 := httptest.NewRecorder()
	suite.router.ServeHTTP(getW2, getReq2)

	assert.Equal(suite.T(), http.StatusNotFound, getW2.Code)
}

// メモフィルタリングのテスト
func (suite *MemoIntegrationTestSuite) TestFilterMemos() {
	// 異なるステータスのメモを作成
	activeMemo := map[string]interface{}{
		"title":   "アクティブメモ",
		"content": "アクティブなメモ",
	}
	activeMemoJSON, _ := json.Marshal(activeMemo)
	createReq1 := httptest.NewRequest("POST", "/api/memos", bytes.NewReader(activeMemoJSON))
	createReq1.Header.Set("Content-Type", "application/json")
	createW1 := httptest.NewRecorder()
	suite.router.ServeHTTP(createW1, createReq1)

	archivedMemo := map[string]interface{}{
		"title":   "アーカイブメモ",
		"content": "アーカイブ予定のメモ",
	}
	archivedMemoJSON, _ := json.Marshal(archivedMemo)
	createReq2 := httptest.NewRequest("POST", "/api/memos", bytes.NewReader(archivedMemoJSON))
	createReq2.Header.Set("Content-Type", "application/json")
	createW2 := httptest.NewRecorder()
	suite.router.ServeHTTP(createW2, createReq2)

	var archivedMemoResponse domain.Memo
	json.Unmarshal(createW2.Body.Bytes(), &archivedMemoResponse)

	// 1つのメモをアーカイブする
	deleteReq := httptest.NewRequest("DELETE", "/api/memos/"+strconv.Itoa(archivedMemoResponse.ID), nil)
	deleteW := httptest.NewRecorder()
	suite.router.ServeHTTP(deleteW, deleteReq)

	// アクティブメモのみをフィルタリング
	req := httptest.NewRequest("GET", "/api/memos?status=active", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	var activeResponse struct {
		Memos []domain.Memo `json:"memos"`
		Total int           `json:"total"`
	}
	json.Unmarshal(w.Body.Bytes(), &activeResponse)

	assert.Equal(suite.T(), 1, activeResponse.Total)
	assert.Equal(suite.T(), "アクティブメモ", activeResponse.Memos[0].Title)

	// アーカイブメモのフィルタリング
	req2 := httptest.NewRequest("GET", "/api/memos?status=archived", nil)
	w2 := httptest.NewRecorder()
	suite.router.ServeHTTP(w2, req2)

	var archivedResponse struct {
		Memos []domain.Memo `json:"memos"`
		Total int           `json:"total"`
	}
	json.Unmarshal(w2.Body.Bytes(), &archivedResponse)

	assert.Equal(suite.T(), 1, archivedResponse.Total)
	assert.Equal(suite.T(), "アーカイブメモ", archivedResponse.Memos[0].Title)
}

// TestUserIsolation_MemoCreation - ユーザーがメモを作成し、他のユーザーからは見えないことを確認
func (suite *MemoIntegrationTestSuite) TestUserIsolation_MemoCreation() {
	// 新しいルーターに認証ミドルウェアを追加
	router := gin.New()
	router.Use(middleware.LoggerMiddleware())
	router.Use(middleware.CORSMiddleware())

	api := router.Group("/api")
	memos := api.Group("/memos")
	memos.Use(suite.mockAuthMiddleware())
	{
		memos.POST("", suite.memoHandler.CreateMemo)
		memos.GET("", suite.memoHandler.ListMemos)
		memos.GET("/:id", suite.memoHandler.GetMemo)
	}

	// ユーザー1がメモを作成
	createReq := map[string]interface{}{
		"title":    "ユーザー1のプライベートメモ",
		"content":  "これはユーザー1だけが見られるメモです",
		"category": "個人",
		"priority": "high",
	}

	reqBody, _ := json.Marshal(createReq)
	req1 := httptest.NewRequest("POST", "/api/memos", bytes.NewReader(reqBody))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("X-User-ID", "1")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	assert.Equal(suite.T(), http.StatusCreated, w1.Code)

	var createResp map[string]interface{}
	err := json.Unmarshal(w1.Body.Bytes(), &createResp)
	require.NoError(suite.T(), err)

	// メモIDを取得
	memoIDFloat, ok := createResp["id"].(float64)
	require.True(suite.T(), ok)
	memoID := int(memoIDFloat)

	// ユーザー1は自分のメモが見える
	req2 := httptest.NewRequest("GET", "/api/memos", nil)
	req2.Header.Set("X-User-ID", "1")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(suite.T(), http.StatusOK, w2.Code)

	var listResp map[string]interface{}
	err = json.Unmarshal(w2.Body.Bytes(), &listResp)
	require.NoError(suite.T(), err)

	totalFloat, ok := listResp["total"].(float64)
	require.True(suite.T(), ok)
	assert.Equal(suite.T(), 1, int(totalFloat))

	// ユーザー2は他人のメモが見えない
	req3 := httptest.NewRequest("GET", "/api/memos", nil)
	req3.Header.Set("X-User-ID", "2")
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)

	assert.Equal(suite.T(), http.StatusOK, w3.Code)

	err = json.Unmarshal(w3.Body.Bytes(), &listResp)
	require.NoError(suite.T(), err)

	totalFloat, ok = listResp["total"].(float64)
	require.True(suite.T(), ok)
	assert.Equal(suite.T(), 0, int(totalFloat))

	// ユーザー2は他人のメモIDを直接指定してもアクセスできない
	req4 := httptest.NewRequest("GET", fmt.Sprintf("/api/memos/%d", memoID), nil)
	req4.Header.Set("X-User-ID", "2")
	w4 := httptest.NewRecorder()
	router.ServeHTTP(w4, req4)

	assert.Equal(suite.T(), http.StatusNotFound, w4.Code)
}

// TestUserIsolation_UnauthorizedAccess - 認証なしでのメモアクセスが拒否されることを確認
func (suite *MemoIntegrationTestSuite) TestUserIsolation_UnauthorizedAccess() {
	// 新しいルーターに認証ミドルウェアを追加
	router := gin.New()
	router.Use(middleware.LoggerMiddleware())
	router.Use(middleware.CORSMiddleware())

	api := router.Group("/api")
	memos := api.Group("/memos")
	memos.Use(suite.mockAuthMiddleware())
	{
		memos.POST("", suite.memoHandler.CreateMemo)
		memos.GET("", suite.memoHandler.ListMemos)
	}

	// 認証なしでメモ一覧取得
	req1 := httptest.NewRequest("GET", "/api/memos", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	assert.Equal(suite.T(), http.StatusUnauthorized, w1.Code)

	// 認証なしでメモ作成
	createReq := map[string]interface{}{
		"title":    "未認証メモ",
		"content":  "認証なしで作成されるべきではない",
		"priority": "low",
	}

	reqBody, _ := json.Marshal(createReq)
	req2 := httptest.NewRequest("POST", "/api/memos", bytes.NewReader(reqBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(suite.T(), http.StatusUnauthorized, w2.Code)
}

// TestUserIsolation_UpdateDeleteAccess - 他のユーザーのメモを更新・削除できないことを確認
func (suite *MemoIntegrationTestSuite) TestUserIsolation_UpdateDeleteAccess() {
	// 新しいルーターに認証ミドルウェアを追加
	router := gin.New()
	router.Use(middleware.LoggerMiddleware())
	router.Use(middleware.CORSMiddleware())

	api := router.Group("/api")
	memos := api.Group("/memos")
	memos.Use(suite.mockAuthMiddleware())
	{
		memos.POST("", suite.memoHandler.CreateMemo)
		memos.GET("/:id", suite.memoHandler.GetMemo)
		memos.PUT("/:id", suite.memoHandler.UpdateMemo)
		memos.DELETE("/:id", suite.memoHandler.DeleteMemo)
	}

	// ユーザー1がメモを作成
	createReq := map[string]interface{}{
		"title":    "元のタイトル",
		"content":  "元の内容",
		"category": "元のカテゴリ",
		"priority": "medium",
	}

	reqBody, _ := json.Marshal(createReq)
	req1 := httptest.NewRequest("POST", "/api/memos", bytes.NewReader(reqBody))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("X-User-ID", "1")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	assert.Equal(suite.T(), http.StatusCreated, w1.Code)

	var createResp map[string]interface{}
	err := json.Unmarshal(w1.Body.Bytes(), &createResp)
	require.NoError(suite.T(), err)

	memoIDFloat, ok := createResp["id"].(float64)
	require.True(suite.T(), ok)
	memoID := int(memoIDFloat)

	// ユーザー2が他人のメモを更新しようとする
	updateReq := map[string]interface{}{
		"title":    "悪意のあるタイトル",
		"content":  "悪意のある内容",
		"category": "悪意のあるカテゴリ",
		"priority": "low",
	}

	updateBody, _ := json.Marshal(updateReq)
	req2 := httptest.NewRequest("PUT", fmt.Sprintf("/api/memos/%d", memoID), bytes.NewReader(updateBody))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-User-ID", "2")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(suite.T(), http.StatusNotFound, w2.Code)

	// ユーザー1のメモが変更されていないことを確認
	req3 := httptest.NewRequest("GET", fmt.Sprintf("/api/memos/%d", memoID), nil)
	req3.Header.Set("X-User-ID", "1")
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)

	assert.Equal(suite.T(), http.StatusOK, w3.Code)

	var getResp map[string]interface{}
	err = json.Unmarshal(w3.Body.Bytes(), &getResp)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "元のタイトル", getResp["title"])
	assert.Equal(suite.T(), "元の内容", getResp["content"])
	assert.Equal(suite.T(), "元のカテゴリ", getResp["category"])

	// ユーザー2が他人のメモを削除しようとする
	req4 := httptest.NewRequest("DELETE", fmt.Sprintf("/api/memos/%d", memoID), nil)
	req4.Header.Set("X-User-ID", "2")
	w4 := httptest.NewRecorder()
	router.ServeHTTP(w4, req4)

	assert.Equal(suite.T(), http.StatusNotFound, w4.Code)

	// ユーザー1のメモが存在することを確認
	req5 := httptest.NewRequest("GET", fmt.Sprintf("/api/memos/%d", memoID), nil)
	req5.Header.Set("X-User-ID", "1")
	w5 := httptest.NewRecorder()
	router.ServeHTTP(w5, req5)

	assert.Equal(suite.T(), http.StatusOK, w5.Code)
}

// モック認証ミドルウェア - ヘッダーからユーザーIDを取得
func (suite *MemoIntegrationTestSuite) mockAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDHeader := c.GetHeader("X-User-ID")
		if userIDHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// ユーザーIDをコンテキストに設定
		userID, err := strconv.Atoi(userIDHeader)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}

// 基本テスト用のシンプルな認証ミドルウェア（固定ユーザーID=1）
func (suite *MemoIntegrationTestSuite) simpleAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// テスト用の固定ユーザーID
		c.Set("user_id", 1)
		c.Next()
	}
}

func TestMemoIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(MemoIntegrationTestSuite))
}
