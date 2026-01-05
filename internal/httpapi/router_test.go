package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	r := NewRouter(false)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}

	var payload healthResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if payload.Status != "ok" {
		t.Fatalf("unexpected status: %s", payload.Status)
	}
}

func TestWellKnownJMAPEndpoint(t *testing.T) {
	r := NewRouter(false)
	req := httptest.NewRequest(http.MethodGet, "/.well-known/jmap", nil)
	req.Host = "example.com"
	req.Header.Set("X-Forwarded-Proto", "https")

	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}

	var payload wellKnownJMAP
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if payload.APIURL != "https://example.com/jmap" {
		t.Fatalf("unexpected api url: %s", payload.APIURL)
	}

	if payload.Capabilities["urn:ietf:params:jmap:core"] == nil {
		t.Fatalf("missing core capability")
	}
}
