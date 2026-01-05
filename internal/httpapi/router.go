package httpapi

import (
	"encoding/json"
	"net/http"

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
