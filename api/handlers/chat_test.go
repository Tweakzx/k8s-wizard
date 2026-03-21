package handlers

import (
	"context"
	"encoding/json"
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
}

func (m *mockAgent) ProcessCommandWithClarification(ctx context.Context, userMsg string, formData map[string]interface{}, confirm *bool) (string, *models.ClarificationRequest, *models.ActionPreview, error) {
	m.processCalled = true
	m.lastMsg = userMsg
	m.lastFormData = formData
	m.lastConfirm = confirm
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
