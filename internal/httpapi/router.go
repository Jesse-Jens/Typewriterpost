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

type adminTenant struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Plan     string `json:"plan"`
	Status   string `json:"status"`
	Domain   string `json:"domain"`
	UserCap  int    `json:"userCap"`
	UserUsed int    `json:"userUsed"`
}

type adminUser struct {
	ID     string `json:"id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	Status string `json:"status"`
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
	r.Get("/admin/api/tenants", handleAdminTenants)
	r.Get("/admin/api/tenants/{tenantId}/users", handleAdminTenantUsers)

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
    <title>Typewriterpost Global Admin Center</title>
    <style>
      :root {
        color-scheme: light;
      }
      body {
        font-family: "Georgia", "Times New Roman", serif;
        background: #f8f5ef;
        color: #0d0d0d;
        margin: 0;
      }
      header {
        background: #0f2a1f;
        color: #f8f5ef;
        padding: 1.5rem 2rem;
        border-bottom: 4px solid #b87333;
      }
      header h1 { margin: 0; font-weight: 600; }
      header p { margin: 0.4rem 0 0; opacity: 0.8; }
      main {
        display: grid;
        grid-template-columns: 280px 1fr;
        gap: 1.5rem;
        padding: 2rem;
      }
      nav {
        background: #fff;
        border-radius: 16px;
        padding: 1.5rem;
        box-shadow: 0 10px 24px rgba(0,0,0,0.08);
        height: fit-content;
      }
      nav h2 {
        margin-top: 0;
        font-size: 1.1rem;
        letter-spacing: 0.03em;
        text-transform: uppercase;
      }
      nav ul { list-style: none; padding: 0; margin: 0; }
      nav li { margin: 0.6rem 0; }
      nav a {
        color: #b87333;
        text-decoration: none;
        font-weight: 600;
      }
      nav a:hover { text-decoration: underline; }
      section {
        background: #fff;
        border-radius: 18px;
        padding: 1.5rem 2rem;
        box-shadow: 0 10px 24px rgba(0,0,0,0.08);
        margin-bottom: 1.5rem;
      }
      h2 { margin-top: 0; }
      .grid {
        display: grid;
        gap: 1rem;
        grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
      }
      .metric {
        border: 1px solid #efe6da;
        border-radius: 12px;
        padding: 1rem;
        background: #fdfbf8;
      }
      .pill {
        display: inline-block;
        padding: 0.15rem 0.6rem;
        border-radius: 999px;
        background: #b87333;
        color: #fff;
        font-size: 0.85rem;
      }
      table {
        width: 100%;
        border-collapse: collapse;
      }
      th, td {
        text-align: left;
        padding: 0.6rem;
        border-bottom: 1px solid #efe6da;
      }
      th { font-weight: 600; }
      code { background: #f1ede5; padding: 0.1rem 0.3rem; border-radius: 4px; }
      footer {
        padding: 0 2rem 2rem;
        color: #4f4a44;
      }
    </style>
  </head>
  <body>
    <header>
      <h1>Typewriterpost Global Admin Center</h1>
      <p>Central command for tenants, domains, users, and service health.</p>
    </header>
    <main>
      <nav>
        <h2>Navigation</h2>
        <ul>
          <li><a href="` + base + `/admin">Overview</a></li>
          <li><a href="` + base + `/admin/api/tenants">Tenants API</a></li>
          <li><a href="` + base + `/admin/api/tenants/default/users">Tenant users API</a></li>
          <li><a href="` + base + `/admin/status">Admin status</a></li>
          <li><a href="` + base + `/healthz">Health check</a></li>
          <li><a href="` + base + `/.well-known/jmap">JMAP discovery</a></li>
        </ul>
      </nav>
      <div>
        <section>
          <h2>Platform Snapshot</h2>
          <div class="grid">
            <div class="metric">
              <div class="pill">Status</div>
              <h3>Healthy</h3>
              <p>All core services responding.</p>
            </div>
            <div class="metric">
              <div class="pill">Tenants</div>
              <h3>1 Active</h3>
              <p>Global admin can create and manage tenants.</p>
            </div>
            <div class="metric">
              <div class="pill">Mail</div>
              <h3>Postfix + Dovecot</h3>
              <p>SMTP/IMAP running with TLS enabled.</p>
            </div>
          </div>
        </section>
        <section>
          <h2>Tenants</h2>
          <p>Seeded data for now. Use the API to integrate with provisioning workflows.</p>
          <table>
            <thead>
              <tr>
                <th>Tenant</th>
                <th>Domain</th>
                <th>Plan</th>
                <th>Users</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td>Default Tenant</td>
                <td>example.local</td>
                <td>Enterprise</td>
                <td>1 / 25</td>
                <td><span class="pill">Active</span></td>
              </tr>
            </tbody>
          </table>
          <p>API: <code>` + base + `/admin/api/tenants</code></p>
        </section>
        <section>
          <h2>Tenant Users</h2>
          <p>Manage tenant users per company. API stub: <code>` + base + `/admin/api/tenants/{tenantId}/users</code></p>
          <table>
            <thead>
              <tr>
                <th>User</th>
                <th>Role</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td>admin@example.local</td>
                <td>Owner</td>
                <td><span class="pill">Active</span></td>
              </tr>
            </tbody>
          </table>
        </section>
      </div>
    </main>
    <footer>
      <p>Next steps: wire authentication, tenant provisioning, and mailbox lifecycle automation.</p>
    </footer>
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

func handleAdminTenants(w http.ResponseWriter, r *http.Request) {
	tenants := []adminTenant{
		{
			ID:       "default",
			Name:     "Default Tenant",
			Plan:     "enterprise",
			Status:   "active",
			Domain:   "example.local",
			UserCap:  25,
			UserUsed: 1,
		},
	}
	writeJSON(w, http.StatusOK, tenants)
}

func handleAdminTenantUsers(w http.ResponseWriter, r *http.Request) {
	_ = chi.URLParam(r, "tenantId")
	users := []adminUser{
		{
			ID:     "user-1",
			Email:  "admin@example.local",
			Role:   "owner",
			Status: "active",
		},
	}
	writeJSON(w, http.StatusOK, users)
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
