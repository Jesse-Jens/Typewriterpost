package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type healthResponse struct {
	Status string `json:"status"`
}

type wellKnownJMAP struct {
	PrimaryAccounts map[string]string `json:"primaryAccounts"`
	Capabilities    map[string]any    `json:"capabilities"`
	APIURL          string            `json:"apiUrl"`
	DownloadURL     string            `json:"downloadUrl"`
	UploadURL       string            `json:"uploadUrl"`
	EventSourceURL  string            `json:"eventSourceUrl"`
}

type adminStatusResponse struct {
	Service   string `json:"service"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

// NewRouter configures the HTTP routes for the JMAP API service.
func NewRouter(enableRequestLogging bool) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	if enableRequestLogging {
		r.Use(middleware.Logger)
	}

	r.Get("/healthz", handleHealth)
	r.Get("/.well-known/jmap", handleWellKnown)
	r.Get("/admin", handleAdmin)
	r.Get("/admin/status", handleAdminStatus)

	return r
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}

func handleWellKnown(w http.ResponseWriter, r *http.Request) {
	base := schemeFromRequest(r) + "://" + r.Host

	resp := wellKnownJMAP{
		PrimaryAccounts: map[string]string{},
		Capabilities: map[string]any{
			"urn:ietf:params:jmap:core": map[string]any{
				"maxSizeUpload":         10 * 1024 * 1024,
				"maxConcurrentRequests": 8,
				"maxCallsInRequest":     32,
				"maxObjectsInGet":       1024,
				"maxObjectsInSet":       1024,
			},
		},
		APIURL:         base + "/jmap",
		DownloadURL:    base + "/download/{accountId}/{blobId}",
		UploadURL:      base + "/upload/{accountId}/",
		EventSourceURL: base + "/event-source",
	}

	writeJSON(w, http.StatusOK, resp)
}

func handleAdmin(w http.ResponseWriter, r *http.Request) {
	base := schemeFromRequest(r) + "://" + r.Host
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8"/>
    <title>Typewriterpost Admin</title>
    <style>
      body { font-family: "Georgia", "Times New Roman", serif; background: #f8f5ef; color: #0d0d0d; margin: 2rem; }
      .card { background: #fff; padding: 1.5rem; border-radius: 16px; box-shadow: 0 10px 24px rgba(0,0,0,0.08); max-width: 720px; }
      h1 { margin-top: 0; }
      a { color: #b87333; text-decoration: none; }
      a:hover { text-decoration: underline; }
      code { background: #f1ede5; padding: 0.1rem 0.3rem; border-radius: 4px; }
    </style>
  </head>
  <body>
    <div class="card">
      <h1>Typewriterpost Admin</h1>
      <p>This is a minimal admin portal for validating the JMAP API scaffold.</p>
      <ul>
        <li><a href="` + base + `/healthz">Health check</a></li>
        <li><a href="` + base + `/.well-known/jmap">JMAP discovery</a></li>
        <li><a href="` + base + `/admin/status">Admin status (JSON)</a></li>
      </ul>
      <p>Next steps: add authentication, tenant management, and mailbox provisioning workflows.</p>
    </div>
  </body>
</html>`))
}

func handleAdminStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, adminStatusResponse{
		Service:   "typewriterpost-jmap-api",
		Status:    "ok",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func schemeFromRequest(r *http.Request) string {
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		return proto
	}
	if r.TLS != nil {
		return "https"
	}
	if r.URL != nil && r.URL.Scheme != "" {
		return r.URL.Scheme
	}
	return "http"
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
