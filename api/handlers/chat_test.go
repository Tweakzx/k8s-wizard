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

type mockAgent struct {
	processCalled bool
	lastMsg       string
	lastFormData  map[string]interface{}
	lastConfirm   *bool
	processErr    error
}

func (m *mockAgent) ProcessCommandWithClarification(ctx context.Context, userMsg string, formData map[string]interface{}, confirm *bool) (string, *models.ClarificationRequest, *models.ActionPreview, error) {
	m.processCalled = true
	m.lastMsg = userMsg
	m.lastFormData = formData
	m.lastConfirm = confirm
	if m.processErr != nil {
		return "", nil, nil, m.processErr
	}
	return "ok", nil, nil, nil
}

func (m *mockAgent) ProcessCommand(ctx context.Context, userMsg string) (string, error) {
	return "ok", nil
}

func (m *mockAgent) GetModelName() string {
	return "mock-model"
}

func (m *mockAgent) SetModel(modelString string) error {
	return nil
}

type mockThreadAwareAgent struct {
	mockAgent
	threadCalled bool
	lastThreadID string
}

func (m *mockThreadAwareAgent) ProcessCommandWithClarificationAndThread(ctx context.Context, userMsg string, formData map[string]interface{}, confirm *bool, threadID string) (string, *models.ClarificationRequest, *models.ActionPreview, error) {
	m.threadCalled = true
	m.lastMsg = userMsg
	m.lastFormData = formData
	m.lastConfirm = confirm
	m.lastThreadID = threadID
	return "ok", nil, nil, nil
}

func TestChatHandler_UsesSessionIDWhenThreadAwareAgent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	agent := &mockThreadAwareAgent{}
	handler := NewChatHandler(agent)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"content":"deploy nginx","sessionId":"session-123","formData":{"name":"nginx"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	handler.Handle(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !agent.threadCalled {
		t.Fatal("expected ProcessCommandWithClarificationAndThread to be called")
	}
	if agent.lastThreadID != "session-123" {
		t.Fatalf("threadID = %q, want %q", agent.lastThreadID, "session-123")
	}
	if agent.processCalled {
		t.Fatal("expected ProcessCommandWithClarification not to be called when thread-aware method exists")
	}
}

func TestChatHandler_FallsBackWhenAgentNotThreadAware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	agent := &mockAgent{}
	handler := NewChatHandler(agent)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	confirm := true
	payload := map[string]interface{}{
		"content":   "deploy nginx",
		"sessionId": "session-456",
		"confirm":   confirm,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(string(raw)))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	handler.Handle(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !agent.processCalled {
		t.Fatal("expected fallback ProcessCommandWithClarification to be called")
	}
	if agent.lastConfirm == nil || !*agent.lastConfirm {
		t.Fatal("expected confirm to be forwarded in fallback path")
	}
}

func TestChatHandler_InvalidJSONReturnsBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewChatHandler(&mockAgent{})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader("{invalid json"))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	handler.Handle(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "无效的请求格式") {
		t.Fatalf("unexpected response body: %s", w.Body.String())
	}
}

func TestChatHandler_SanitizesProviderErrorInResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewChatHandler(&mockAgent{
		processErr: errors.New("provider error: status 429: rate limit"),
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(`{"content":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	handler.Handle(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	var resp models.ChatResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.Error != "上游服务请求过于频繁，请稍后重试" {
		t.Fatalf("error = %q, want %q", resp.Error, "上游服务请求过于频繁，请稍后重试")
	}
	if resp.Model != "mock-model" {
		t.Fatalf("model = %q, want %q", resp.Model, "mock-model")
	}
}
