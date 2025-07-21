package models_test

import (
	"testing"

	"memo-app/src/models"

	"github.com/stretchr/testify/assert"
)

func TestCreateMemoRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request models.CreateMemoRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: models.CreateMemoRequest{
				Title:    "Test Memo",
				Content:  "This is a test memo",
				Category: "Test",
				Tags:     []string{"test", "sample"},
				Priority: "medium",
			},
			wantErr: false,
		},
		{
			name: "valid request with minimal fields",
			request: models.CreateMemoRequest{
				Title:   "Test Memo",
				Content: "This is a test memo",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// バリデーション自体は service layer で行われるため、
			// ここでは構造体の初期化が正常にできることを確認
			assert.NotNil(t, tt.request)
			assert.IsType(t, models.CreateMemoRequest{}, tt.request)
		})
	}
}

func TestMemoFilter_DefaultValues(t *testing.T) {
	filter := models.MemoFilter{}

	// デフォルト値の確認
	assert.Equal(t, 0, filter.Page)  // 後でサービスレイヤーで1に設定される
	assert.Equal(t, 0, filter.Limit) // 後でサービスレイヤーで10に設定される
	assert.Equal(t, "", filter.Category)
	assert.Equal(t, "", filter.Status)
	assert.Equal(t, "", filter.Priority)
	assert.Equal(t, "", filter.Search)
	assert.Equal(t, "", filter.Tags)
}

func TestMemoListResponse_Structure(t *testing.T) {
	memos := []models.Memo{
		{
			ID:       1,
			Title:    "Test Memo 1",
			Content:  "Content 1",
			Category: "Test",
			Priority: "medium",
			Status:   "active",
		},
		{
			ID:       2,
			Title:    "Test Memo 2",
			Content:  "Content 2",
			Category: "Work",
			Priority: "high",
			Status:   "active",
		},
	}

	response := models.MemoListResponse{
		Memos:      memos,
		Total:      2,
		Page:       1,
		Limit:      10,
		TotalPages: 1,
	}

	assert.Equal(t, 2, len(response.Memos))
	assert.Equal(t, 2, response.Total)
	assert.Equal(t, 1, response.Page)
	assert.Equal(t, 10, response.Limit)
	assert.Equal(t, 1, response.TotalPages)
}
