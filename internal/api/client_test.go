package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestApplyAuthHeadersIncludesWorkspace(t *testing.T) {
	client := NewClient("https://api.example.test", "token-123")
	client.WorkspaceID = "ws-abc"

	req := httptest.NewRequest(http.MethodGet, "https://api.example.test/cli/sites", nil)
	client.applyAuthHeaders(req)

	if got := req.Header.Get("Authorization"); got != "Bearer token-123" {
		t.Fatalf("unexpected authorization header: %q", got)
	}

	if got := req.Header.Get(WorkspaceContextHeader); got != "ws-abc" {
		t.Fatalf("unexpected workspace header: %q", got)
	}
}

func TestApplyAuthHeadersOmitsEmptyWorkspace(t *testing.T) {
	client := NewClient("https://api.example.test", "token-123")

	req := httptest.NewRequest(http.MethodGet, "https://api.example.test/cli/sites", nil)
	client.applyAuthHeaders(req)

	if got := req.Header.Get(WorkspaceContextHeader); got != "" {
		t.Fatalf("expected empty workspace header, got %q", got)
	}
}
