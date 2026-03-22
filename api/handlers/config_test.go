package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"k8s-wizard/api/models"

	"github.com/gin-gonic/gin"
)

type mockConfigAgent struct {
	setModelErr error
}

func (m *mockConfigAgent) ProcessCommandWithClarification(ctx context.Context, userMsg string, formData map[string]interface{}, confirm *bool) (string, *models.ClarificationRequest, *models.ActionPreview, error) {
	return "", nil, nil, nil
}

func (m *mockConfigAgent) ProcessCommand(ctx context.Context, userMsg string) (string, error) {
	return "", nil
}

func (m *mockConfigAgent) GetModelName() string {
	return "mock-model"
}

func (m *mockConfigAgent) SetModel(modelString string) error {
	return m.setModelErr
}

func TestConfigHandler_SetModel_InvalidModelReturnsBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewConfigHandler(&mockConfigAgent{
		setModelErr: errors.New("failed to parse model: invalid model format: invalid-model"),
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPut, "/api/config/model", strings.NewReader(`{"model":"invalid-model","persist":false}`))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	handler.SetModel(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	var resp models.SetModelResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Success {
		t.Fatal("expected success=false")
	}
	if !strings.Contains(resp.Error, "模型切换失败") {
		t.Fatalf("unexpected error response: %s", resp.Error)
	}
}

func TestConfigHandler_SetModel_InternalErrorReturnsInternalServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewConfigHandler(&mockConfigAgent{
		setModelErr: errors.New("failed to create LLM client: API key not configured"),
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPut, "/api/config/model", strings.NewReader(`{"model":"glm/glm-4-flash","persist":false}`))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	handler.SetModel(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	var resp models.SetModelResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Success {
		t.Fatal("expected success=false")
	}
}
