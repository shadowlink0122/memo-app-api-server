package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"memo-app/src/config"
	"memo-app/src/database"
	"memo-app/src/domain"
	"memo-app/src/infrastructure/repository"
	"memo-app/src/interface/handler"
	"memo-app/src/logger"
	"memo-app/src/middleware"
	srcRepository "memo-app/src/repository"
	"memo-app/src/service"
	"memo-app/src/usecase"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
)

type MemoIntegrationTestSuite struct {
	suite.Suite
	router       *gin.Engine
	db           *database.DB
	handler      *handler.MemoHandler
	usecase      usecase.MemoUsecase
	repo         domain.MemoRepository
	jwtService   service.JWTService
	userRepo     srcRepository.UserRepository
	testUserID   int
	testJWTToken string
}

func (suite *MemoIntegrationTestSuite) SetupSuite() {
	// テスト用環境変数の設定（Docker Composeと一致させる）
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_USER", "memo_user")
	os.Setenv("DB_PASSWORD", "memo_password")
	os.Setenv("DB_NAME", "memo_db")
	os.Setenv("DB_SSL_MODE", "disable")

	// テスト用設定の読み込み
	cfg := config.LoadConfig()

	// ロガーの初期化
	err := logger.InitLogger()
	suite.Require().NoError(err)

	// テスト用データベースの設定
	dbConfig := &database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName, // 環境変数から取得
		SSLMode:  cfg.Database.SSLMode,
	}

	// Docker Composeのデータベースに接続を試行
	suite.db, err = database.NewDB(dbConfig, logger.Log)
	if err != nil {
		suite.T().Skipf("データベース接続に失敗しました。Docker Composeでデータベースを起動してください: %v", err)
		return
	}

	// テーブルの作成（もし存在しない場合）
	suite.createTablesIfNotExists()

	// クリーンアーキテクチャの依存関係注入
	suite.repo = repository.NewMemoRepository(suite.db, logger.Log)
	suite.usecase = usecase.NewMemoUsecase(suite.repo)
	suite.handler = handler.NewMemoHandler(suite.usecase, logger.Log)

	// 認証用のサービスとリポジトリ
	suite.userRepo = srcRepository.NewUserRepository(suite.db.DB)
	suite.jwtService = service.NewJWTService(cfg)

	// テストユーザーの作成とJWTトークンの生成
	suite.createTestUser()

	// Ginルーターの設定
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()

	// ミドルウェアの設定
	suite.router.Use(middleware.LoggerMiddleware())
	suite.router.Use(middleware.CORSMiddleware())

	// ルートの設定
	api := suite.router.Group("/api/memos")
	api.Use(middleware.AuthMiddleware(suite.jwtService, suite.userRepo)) // 認証ミドルウェア
	{
		api.POST("", suite.handler.CreateMemo)
		api.GET("", suite.handler.ListMemos)
		api.GET("/:id", suite.handler.GetMemo)
		api.PUT("/:id", suite.handler.UpdateMemo)
		api.DELETE("/:id", suite.handler.DeleteMemo)
		api.PATCH("/:id/archive", suite.handler.ArchiveMemo)
		api.PATCH("/:id/restore", suite.handler.RestoreMemo)
		api.GET("/search", suite.handler.SearchMemos)
	}
}

func (suite *MemoIntegrationTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
	logger.CloseLogger()
}

func (suite *MemoIntegrationTestSuite) SetupTest() {
	// データベースが利用可能でない場合はスキップ
	if suite.db == nil {
		suite.T().Skip("データベースが利用可能でないため、テストをスキップします")
		return
	}

	// 各テスト前にmemosテーブルをクリーンアップ
	ctx := context.Background()
	_, err := suite.db.ExecContext(ctx, "DELETE FROM memos")
	if err != nil {
		// memosテーブルが存在しない場合は作成
		suite.createTablesIfNotExists()
		_, err = suite.db.ExecContext(ctx, "DELETE FROM memos")
		suite.Require().NoError(err)
	}
}

func (suite *MemoIntegrationTestSuite) TestFullMemoLifecycle() {
	// データベースが利用可能でない場合はスキップ
	if suite.db == nil {
		suite.T().Skip("データベースが利用可能でないため、テストをスキップします")
		return
	}

	// 1. メモ作成
	createReq := usecase.CreateMemoRequest{
		Title:    "Integration Test Memo",
		Content:  "This is an integration test memo",
		Category: "Test",
		Tags:     []string{"integration", "test"},
		Priority: "high",
	}

	createBody, err := json.Marshal(createReq)
	suite.Require().NoError(err)

	req := httptest.NewRequest("POST", "/api/memos", bytes.NewBuffer(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.testJWTToken) // 有効なトークンを使用

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusCreated, w.Code)

	// デバッグ: レスポンスボディを確認
	suite.T().Logf("Response body: %s", w.Body.String())

	var createdMemo handler.MemoResponseDTO
	err = json.Unmarshal(w.Body.Bytes(), &createdMemo)
	suite.Require().NoError(err)
	suite.Equal(createReq.Title, createdMemo.Title)
	suite.Equal(createReq.Content, createdMemo.Content)
	memoID := createdMemo.ID

	// デバッグ: memoIDの値を確認
	suite.T().Logf("Created memo ID: %d (0x%x)", memoID, memoID)

	// memoIDが有効であることを確認
	suite.Require().True(memoID > 0, "Invalid memo ID: %d", memoID)

	// 2. メモ取得
	memoIDStr := fmt.Sprintf("%d", memoID)
	getURL := "/api/memos/" + memoIDStr
	req = httptest.NewRequest("GET", getURL, nil)
	req.Header.Set("Authorization", "Bearer "+suite.testJWTToken)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var retrievedMemo handler.MemoResponseDTO
	err = json.Unmarshal(w.Body.Bytes(), &retrievedMemo)
	suite.Require().NoError(err)
	suite.Equal(createdMemo.ID, retrievedMemo.ID)
	suite.Equal(createdMemo.Title, retrievedMemo.Title)

	// 3. メモ一覧取得
	req = httptest.NewRequest("GET", "/api/memos", nil)
	req.Header.Set("Authorization", "Bearer "+suite.testJWTToken)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	// 4. メモ更新
	updateReq := usecase.UpdateMemoRequest{
		Title:   stringPtr("Updated Integration Test Memo"),
		Content: stringPtr("This memo has been updated"),
	}

	updateBody, err := json.Marshal(updateReq)
	suite.Require().NoError(err)

	updateURL := "/api/memos/" + fmt.Sprintf("%d", memoID)
	req = httptest.NewRequest("PUT", updateURL, bytes.NewBuffer(updateBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.testJWTToken)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var updatedMemo domain.Memo
	err = json.Unmarshal(w.Body.Bytes(), &updatedMemo)
	suite.Require().NoError(err)
	suite.Equal(*updateReq.Title, updatedMemo.Title)
	suite.Equal(*updateReq.Content, updatedMemo.Content)

	// 5. メモアーカイブ
	archiveURL := "/api/memos/" + fmt.Sprintf("%d", memoID) + "/archive"
	req = httptest.NewRequest("PATCH", archiveURL, nil)
	req.Header.Set("Authorization", "Bearer "+suite.testJWTToken)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusNoContent, w.Code)

	// 6. メモリストア
	restoreURL := "/api/memos/" + fmt.Sprintf("%d", memoID) + "/restore"
	req = httptest.NewRequest("PATCH", restoreURL, nil)
	req.Header.Set("Authorization", "Bearer "+suite.testJWTToken)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusNoContent, w.Code)

	// 7. メモ削除
	deleteURL := "/api/memos/" + fmt.Sprintf("%d", memoID)
	req = httptest.NewRequest("DELETE", deleteURL, nil)
	req.Header.Set("Authorization", "Bearer "+suite.testJWTToken)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusNoContent, w.Code)

	// 8. 削除後の取得確認（404エラーになるはず）
	getDeletedURL := "/api/memos/" + fmt.Sprintf("%d", memoID)
	req = httptest.NewRequest("GET", getDeletedURL, nil)
	req.Header.Set("Authorization", "Bearer "+suite.testJWTToken)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusNotFound, w.Code)
}

func (suite *MemoIntegrationTestSuite) TestSearchMemos() {
	// テストデータの作成
	testMemos := []usecase.CreateMemoRequest{
		{
			Title:    "First Test Memo",
			Content:  "Content about golang programming",
			Category: "Programming",
			Tags:     []string{"golang", "backend"},
			Priority: "high",
		},
		{
			Title:    "Second Test Memo",
			Content:  "Content about frontend development",
			Category: "Programming",
			Tags:     []string{"javascript", "frontend"},
			Priority: "medium",
		},
		{
			Title:    "Third Test Memo",
			Content:  "Content about project management",
			Category: "Management",
			Tags:     []string{"project", "planning"},
			Priority: "low",
		},
	}

	// メモを作成
	for _, memoReq := range testMemos {
		createBody, err := json.Marshal(memoReq)
		suite.Require().NoError(err)

		req := httptest.NewRequest("POST", "/api/memos", bytes.NewBuffer(createBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+suite.testJWTToken)

		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)

		suite.Equal(http.StatusCreated, w.Code)
	}

	// 検索テスト
	req := httptest.NewRequest("GET", "/api/memos/search?q=golang", nil)
	req.Header.Set("Authorization", "Bearer "+suite.testJWTToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	// レスポンスの確認
	var searchResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &searchResponse)
	suite.Require().NoError(err)
}

func TestMemoIntegrationTestSuite(t *testing.T) {
	// 統合テストはデータベース接続が必要なため、実際の環境でのみ実行
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	suite.Run(t, new(MemoIntegrationTestSuite))
}

// ヘルパー関数
func stringPtr(s string) *string {
	return &s
}

// createTablesIfNotExists テスト用のテーブルを作成
func (suite *MemoIntegrationTestSuite) createTablesIfNotExists() {
	// users テーブルの作成
	usersSQL := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		username VARCHAR(255) UNIQUE NOT NULL,
		email VARCHAR(255) UNIQUE NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		created_ip VARCHAR(45) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	// memos テーブルの作成（テスト用にuser_idにデフォルト値を設定）
	memosSQL := `
	CREATE TABLE IF NOT EXISTS memos (
		id SERIAL PRIMARY KEY,
		title VARCHAR(200) NOT NULL,
		content TEXT NOT NULL,
		category VARCHAR(50),
		tags JSONB DEFAULT '[]'::jsonb,
		priority VARCHAR(10) NOT NULL DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high')),
		status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived')),
		user_id INTEGER DEFAULT 1,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		completed_at TIMESTAMP WITH TIME ZONE
	);`

	// インデックスの作成
	indexSQL := `
	CREATE INDEX IF NOT EXISTS idx_memos_status ON memos(status);
	CREATE INDEX IF NOT EXISTS idx_memos_category ON memos(category);
	CREATE INDEX IF NOT EXISTS idx_memos_priority ON memos(priority);
	CREATE INDEX IF NOT EXISTS idx_memos_created_at ON memos(created_at);
	CREATE INDEX IF NOT EXISTS idx_memos_updated_at ON memos(updated_at);
	CREATE INDEX IF NOT EXISTS idx_memos_tags ON memos USING GIN (tags);`

	// テーブル作成を実行
	ctx := context.Background()
	_, err := suite.db.ExecContext(ctx, usersSQL)
	suite.Require().NoError(err, "Failed to create users table")

	_, err = suite.db.ExecContext(ctx, memosSQL)
	suite.Require().NoError(err, "Failed to create memos table")

	_, err = suite.db.ExecContext(ctx, indexSQL)
	suite.Require().NoError(err, "Failed to create indexes")
}

// createTestUser は テストユーザーを作成し、JWTトークンを生成します
func (suite *MemoIntegrationTestSuite) createTestUser() {
	ctx := context.Background()

	// テスト用ユーザーの挿入（存在しない場合のみ）- created_ipを含む
	insertUserSQL := `
	INSERT INTO users (username, email, password_hash, created_ip) 
	VALUES ('testuser', 'test@example.com', 'hashed_password', '127.0.0.1') 
	ON CONFLICT (username) DO NOTHING;`

	_, err := suite.db.ExecContext(ctx, insertUserSQL)
	suite.Require().NoError(err, "Failed to insert test user")

	// テスト用ユーザーIDを取得
	getUserSQL := `SELECT id FROM users WHERE username = 'testuser' LIMIT 1;`
	err = suite.db.QueryRowContext(ctx, getUserSQL).Scan(&suite.testUserID)
	suite.Require().NoError(err, "Failed to get test user ID")

	// JWTトークンを生成
	suite.testJWTToken, err = suite.jwtService.GenerateAccessToken(suite.testUserID)
	suite.Require().NoError(err, "Failed to generate JWT token")

	suite.T().Logf("Test user ID: %d", suite.testUserID)
	suite.T().Logf("Test JWT token: %s", suite.testJWTToken[:20]+"...") // 最初の20文字のみログ出力
}
