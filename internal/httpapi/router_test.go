package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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

func TestAdminPortal(t *testing.T) {
	r := NewRouter(false)
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.RemoteAddr = "127.0.0.1:1234"

	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}

	if contentType := resp.Header().Get("Content-Type"); contentType == "" {
		t.Fatalf("expected content type header to be set")
	}
}

func TestAdminStatusEndpoint(t *testing.T) {
	r := NewRouter(false)
	req := httptest.NewRequest(http.MethodGet, "/admin/status", nil)
	req.RemoteAddr = "127.0.0.1:1234"

	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}

	var payload adminStatusResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if payload.Service == "" || payload.Status != "ok" {
		t.Fatalf("unexpected payload: %+v", payload)
	}

	if _, err := time.Parse(time.RFC3339, payload.Timestamp); err != nil {
		t.Fatalf("unexpected timestamp: %v", err)
	}
}

func TestAdminTenantsEndpoint(t *testing.T) {
	r := NewRouter(false)
	req := httptest.NewRequest(http.MethodGet, "/admin/api/tenants", nil)
	req.RemoteAddr = "127.0.0.1:1234"

	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}

	var tenants []adminTenant
	if err := json.Unmarshal(resp.Body.Bytes(), &tenants); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(tenants) == 0 {
		t.Fatalf("expected tenants payload")
	}
}

func TestAdminTenantUsersEndpoint(t *testing.T) {
	r := NewRouter(false)
	req := httptest.NewRequest(http.MethodGet, "/admin/api/tenants/default/users", nil)
	req.RemoteAddr = "127.0.0.1:1234"

	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}

	var users []adminUser
	if err := json.Unmarshal(resp.Body.Bytes(), &users); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(users) == 0 {
		t.Fatalf("expected users payload")
	}
}

func TestAdminTenantPage(t *testing.T) {
	r := NewRouter(false)
	req := httptest.NewRequest(http.MethodGet, "/admin/tenants/default", nil)
	req.RemoteAddr = "127.0.0.1:1234"

	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
}

func TestTenantPortal(t *testing.T) {
	r := NewRouter(false)
	req := httptest.NewRequest(http.MethodGet, "/tenant/default", nil)

	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
}

func TestAdminPortalRejectsRemoteAccess(t *testing.T) {
	r := NewRouter(false)
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.RemoteAddr = "203.0.113.10:5555"

	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", resp.Code)
	}
}
