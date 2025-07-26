package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"

	"memo-app/src/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStagedDeletion tests the staged deletion feature
func (suite *MemoIntegrationTestSuite) TestStagedDeletion() {
	// テストデータを準備（HTTP経由で作成）
	memoData := map[string]interface{}{
		"title":   "段階削除テストメモ",
		"content": "アクティブなメモ",
	}

	memoJSON, _ := json.Marshal(memoData)
	createReq := httptest.NewRequest("POST", "/api/memos", bytes.NewReader(memoJSON))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	suite.router.ServeHTTP(createW, createReq)

	require.Equal(suite.T(), http.StatusCreated, createW.Code)

	var memo domain.Memo
	err := json.Unmarshal(createW.Body.Bytes(), &memo)
	require.NoError(suite.T(), err)

	// 1. アクティブメモの削除（アーカイブに移動）
	deleteURL := "/api/memos/" + strconv.Itoa(memo.ID)
	req := httptest.NewRequest("DELETE", deleteURL, nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// ステータスコードを確認（204 No Content）
	assert.Equal(suite.T(), http.StatusNoContent, w.Code)

	// メモがアーカイブに移動されていることを確認
	getReq := httptest.NewRequest("GET", deleteURL, nil)
	getW := httptest.NewRecorder()
	suite.router.ServeHTTP(getW, getReq)

	assert.Equal(suite.T(), http.StatusOK, getW.Code)

	var archivedMemo domain.Memo
	err = json.Unmarshal(getW.Body.Bytes(), &archivedMemo)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), domain.StatusArchived, archivedMemo.Status)
	assert.NotNil(suite.T(), archivedMemo.CompletedAt)

	// 2. アーカイブ済みメモの削除（完全削除）
	req2 := httptest.NewRequest("DELETE", deleteURL, nil)
	w2 := httptest.NewRecorder()
	suite.router.ServeHTTP(w2, req2)

	// ステータスコードを確認（204 No Content）
	assert.Equal(suite.T(), http.StatusNoContent, w2.Code)

	// メモが完全に削除されていることを確認
	getReq2 := httptest.NewRequest("GET", deleteURL, nil)
	getW2 := httptest.NewRecorder()
	suite.router.ServeHTTP(getW2, getReq2)

	assert.Equal(suite.T(), http.StatusNotFound, getW2.Code)
}

// TestPermanentDeleteEndpoint tests the permanent delete endpoint
func (suite *MemoIntegrationTestSuite) TestPermanentDeleteEndpoint() {
	// テストデータを準備（HTTP経由で作成）
	memoData := map[string]interface{}{
		"title":   "完全削除テストメモ",
		"content": "直接削除するメモ",
	}

	memoJSON, _ := json.Marshal(memoData)
	createReq := httptest.NewRequest("POST", "/api/memos", bytes.NewReader(memoJSON))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	suite.router.ServeHTTP(createW, createReq)

	require.Equal(suite.T(), http.StatusCreated, createW.Code)

	var memo domain.Memo
	err := json.Unmarshal(createW.Body.Bytes(), &memo)
	require.NoError(suite.T(), err)

	// 完全削除エンドポイントでアクティブメモを削除
	permanentDeleteURL := "/api/memos/" + strconv.Itoa(memo.ID) + "/permanent"
	req := httptest.NewRequest("DELETE", permanentDeleteURL, nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// ステータスコードを確認（204 No Content）
	assert.Equal(suite.T(), http.StatusNoContent, w.Code)

	// メモが完全に削除されていることを確認
	getReq := httptest.NewRequest("GET", "/api/memos/"+strconv.Itoa(memo.ID), nil)
	getW := httptest.NewRecorder()
	suite.router.ServeHTTP(getW, getReq)

	assert.Equal(suite.T(), http.StatusNotFound, getW.Code)
}

// TestStagedDeletionFlow tests the complete staged deletion flow
func (suite *MemoIntegrationTestSuite) TestStagedDeletionFlow() {
	// 1. メモを作成
	createReq := map[string]interface{}{
		"title":   "削除フローテスト",
		"content": "段階的削除のフローをテストします",
	}
	createBody, _ := json.Marshal(createReq)

	req := httptest.NewRequest("POST", "/api/memos", bytes.NewBuffer(createBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	require.Equal(suite.T(), http.StatusCreated, w.Code)

	var createdMemo domain.Memo
	err := json.Unmarshal(w.Body.Bytes(), &createdMemo)
	require.NoError(suite.T(), err)

	memoID := strconv.Itoa(createdMemo.ID)

	// 2. 一回目の削除（アーカイブ）
	deleteReq := httptest.NewRequest("DELETE", "/api/memos/"+memoID, nil)
	deleteW := httptest.NewRecorder()
	suite.router.ServeHTTP(deleteW, deleteReq)

	assert.Equal(suite.T(), http.StatusNoContent, deleteW.Code)

	// アーカイブされていることを確認
	getReq := httptest.NewRequest("GET", "/api/memos/"+memoID, nil)
	getW := httptest.NewRecorder()
	suite.router.ServeHTTP(getW, getReq)

	assert.Equal(suite.T(), http.StatusOK, getW.Code)

	var archivedMemo domain.Memo
	err = json.Unmarshal(getW.Body.Bytes(), &archivedMemo)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), domain.StatusArchived, archivedMemo.Status)

	// 3. 二回目の削除（完全削除）
	deleteReq2 := httptest.NewRequest("DELETE", "/api/memos/"+memoID, nil)
	deleteW2 := httptest.NewRecorder()
	suite.router.ServeHTTP(deleteW2, deleteReq2)

	assert.Equal(suite.T(), http.StatusNoContent, deleteW2.Code)

	// 完全に削除されていることを確認
	getReq2 := httptest.NewRequest("GET", "/api/memos/"+memoID, nil)
	getW2 := httptest.NewRecorder()
	suite.router.ServeHTTP(getW2, getReq2)

	assert.Equal(suite.T(), http.StatusNotFound, getW2.Code)
}
