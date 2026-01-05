package httpapi

import (
	"encoding/json"
	"net"
	"net/http"
	"net/netip"
	"strconv"
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

type adminPageData struct {
	AdminUsername string
	Tenants       []*tenantRecord
	Message       string
}

type tenantPageData struct {
	Tenant  *tenantRecord
	Message string
}

// NewRouter configures the HTTP routes for the JMAP API service.
func NewRouter(enableRequestLogging bool) http.Handler {
	r := chi.NewRouter()
	store := newAdminStore()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	if enableRequestLogging {
		r.Use(middleware.Logger)
	}

	r.Get("/healthz", handleHealth)
	r.Get("/.well-known/jmap", handleWellKnown)
	r.Route("/admin", func(r chi.Router) {
		r.Use(requireLocalhost)
		r.Get("/login", handleAdminLogin(store))
		r.Post("/login", handleAdminLogin(store))
		r.With(requireAdminSession(store)).Get("/", handleAdmin(store))
		r.With(requireAdminSession(store)).Get("/status", handleAdminStatus)
		r.With(requireAdminSession(store)).Post("/logout", handleAdminLogout(store))
		r.With(requireAdminSession(store)).Post("/settings", handleAdminSettings(store))
		r.With(requireAdminSession(store)).Get("/api/tenants", handleAdminTenants(store))
		r.With(requireAdminSession(store)).Get("/api/tenants/{tenantId}/users", handleAdminTenantUsers(store))
		r.With(requireAdminSession(store)).Get("/tenants/{tenantId}", handleAdminTenant(store))
		r.With(requireAdminSession(store)).Post("/tenants", handleAdminTenantCreate(store))
		r.With(requireAdminSession(store)).Post("/tenants/{tenantId}/remove", handleAdminTenantRemove(store))
		r.With(requireAdminSession(store)).Post("/tenants/{tenantId}/users", handleAdminTenantAddUser(store))
		r.With(requireAdminSession(store)).Post("/tenants/{tenantId}/users/remove", handleAdminTenantRemoveUser(store))
		r.With(requireAdminSession(store)).Post("/tenants/{tenantId}/domains", handleAdminTenantAddDomain(store))
		r.With(requireAdminSession(store)).Post("/tenants/{tenantId}/domains/remove", handleAdminTenantRemoveDomain(store))
	})
	r.Get("/tenant/{tenantId}", handleTenantPortal(store))
	r.Get("/tenant/{tenantId}/login", handleTenantLogin)

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

func handleAdmin(store *adminStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		base := schemeFromRequest(r) + "://" + r.Host
		data := adminPageData{
			AdminUsername: store.currentAdmin().Username,
			Tenants:       store.listTenants(),
			Message:       r.URL.Query().Get("message"),
		}
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
      h3 { margin-top: 0.4rem; }
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
      .actions a {
        display: inline-block;
        margin-right: 0.6rem;
        padding: 0.35rem 0.7rem;
        border-radius: 999px;
        background: #0f2a1f;
        color: #f8f5ef;
        font-size: 0.85rem;
        text-decoration: none;
      }
      .actions a.secondary {
        background: #b87333;
      }
      code { background: #f1ede5; padding: 0.1rem 0.3rem; border-radius: 4px; }
      .message {
        background: #f1ede5;
        padding: 0.6rem 0.8rem;
        border-radius: 12px;
        margin-bottom: 1rem;
      }
      form { margin-top: 0.8rem; }
      label { display: block; margin-bottom: 0.2rem; }
      input, select {
        padding: 0.4rem 0.6rem;
        border-radius: 8px;
        border: 1px solid #e2d7c7;
        width: 100%;
      }
      .form-row {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
        gap: 0.8rem;
      }
      .btn {
        margin-top: 0.8rem;
        padding: 0.45rem 0.9rem;
        border-radius: 999px;
        border: none;
        background: #0f2a1f;
        color: #f8f5ef;
        font-weight: 600;
        cursor: pointer;
      }
      .btn.secondary { background: #b87333; }
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
        </ul>
        <h2>System</h2>
        <ul>
          <li><a href="` + base + `/admin/status">Admin status</a></li>
          <li><a href="` + base + `/healthz">Health check</a></li>
          <li><a href="` + base + `/.well-known/jmap">JMAP discovery</a></li>
        </ul>
      </nav>
      <div>
        <section>
          <h2>Platform Snapshot</h2>
          ` + renderMessage(data.Message) + `
          <div class="grid">
            <div class="metric">
              <div class="pill">Status</div>
              <h3>Healthy</h3>
              <p>All core services responding.</p>
            </div>
            <div class="metric">
              <div class="pill">Tenants</div>
              <h3>` + formatCount(len(data.Tenants)) + ` Active</h3>
              <p>Global admin can create and manage tenants.</p>
            </div>
            <div class="metric">
              <div class="pill">Mail</div>
              <h3>Postfix</h3>
              <p>SMTP transport with TLS enabled (JMAP-native mail storage planned).</p>
            </div>
          </div>
          <form method="post" action="` + base + `/admin/logout">
            <button class="btn secondary" type="submit">Sign out</button>
          </form>
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
                <th>Actions</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              ` + renderTenantRows(base, data.Tenants) + `
            </tbody>
          </table>
          <form method="post" action="` + base + `/admin/tenants">
            <div class="form-row">
              <div>
                <label for="tenant-id">Tenant ID</label>
                <input id="tenant-id" name="tenant_id" placeholder="acme" required />
              </div>
              <div>
                <label for="tenant-name">Tenant name</label>
                <input id="tenant-name" name="tenant_name" placeholder="Acme Corp" required />
              </div>
              <div>
                <label for="tenant-domain">Primary domain</label>
                <input id="tenant-domain" name="tenant_domain" placeholder="acme.test" required />
              </div>
              <div>
                <label for="tenant-plan">Plan</label>
                <select id="tenant-plan" name="tenant_plan">
                  <option value="starter">Starter</option>
                  <option value="growth">Growth</option>
                  <option value="enterprise" selected>Enterprise</option>
                </select>
              </div>
            </div>
            <button class="btn" type="submit">Add tenant</button>
          </form>
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
                <th>Tenant</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td>admin@example.local</td>
                <td>Owner</td>
                <td>Default Tenant</td>
                <td><span class="pill">Active</span></td>
              </tr>
            </tbody>
          </table>
          <div class="actions">
            <a href="` + base + `/admin/tenants/default">Manage tenant</a>
            <a class="secondary" href="` + base + `/tenant/default">Take over</a>
          </div>
        </section>
        <section>
          <h2>Global admin credentials</h2>
          <p>Update the global admin credentials for this service.</p>
          <form method="post" action="` + base + `/admin/settings">
            <div class="form-row">
              <div>
                <label for="admin-username">Username</label>
                <input id="admin-username" name="admin_username" value="` + data.AdminUsername + `" required />
              </div>
              <div>
                <label for="admin-password">Password</label>
                <input id="admin-password" name="admin_password" type="password" placeholder="new password" required />
              </div>
            </div>
            <button class="btn secondary" type="submit">Update credentials</button>
          </form>
          <p class="note">Default login: <code>` + data.AdminUsername + ` / change-me-now</code></p>
        </section>
      </div>
    </main>
    <footer>
      <p>Next steps: wire authentication, tenant provisioning, mailbox lifecycle automation, and JMAP-native mail storage.</p>
    </footer>
  </body>
</html>`))
	}
}

func handleAdminLogin(store *adminStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if err := r.ParseForm(); err == nil {
				username := r.FormValue("username")
				password := r.FormValue("password")
				if store.authenticate(username, password) {
					token, err := store.newSession(username)
					if err == nil {
						http.SetCookie(w, &http.Cookie{
							Name:     "admin_session",
							Value:    token,
							Path:     "/",
							HttpOnly: true,
							SameSite: http.SameSiteStrictMode,
						})
						http.Redirect(w, r, "/admin?message=Signed+in", http.StatusSeeOther)
						return
					}
				}
			}
			http.Redirect(w, r, "/admin/login?error=invalid", http.StatusSeeOther)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		errorMessage := ""
		if r.URL.Query().Get("error") != "" {
			errorMessage = `<div class="message">Invalid credentials.</div>`
		}
		_, _ = w.Write([]byte(`<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8"/>
    <title>Global Admin Login</title>
    <style>
      body { font-family: "Georgia", "Times New Roman", serif; background: #f8f5ef; color: #0d0d0d; margin: 0; }
      header { background: #0f2a1f; color: #f8f5ef; padding: 1.5rem 2rem; }
      main { padding: 2rem; }
      .card { background: #fff; padding: 1.5rem; border-radius: 16px; box-shadow: 0 10px 24px rgba(0,0,0,0.08); max-width: 640px; }
      .field { margin: 0.6rem 0; }
      input { width: 100%; padding: 0.5rem 0.6rem; border-radius: 8px; border: 1px solid #e2d7c7; }
      button { margin-top: 0.6rem; padding: 0.5rem 0.9rem; border-radius: 999px; border: none; background: #b87333; color: #fff; font-weight: 600; }
      .message { background: #f1ede5; padding: 0.6rem 0.8rem; border-radius: 12px; margin-bottom: 1rem; }
    </style>
  </head>
  <body>
    <header>
      <h1>Global Admin Login</h1>
      <p>Sign in to manage tenants and platform settings.</p>
    </header>
    <main>
      <div class="card">
        ` + errorMessage + `
        <form method="post" action="/admin/login">
          <div class="field">
            <label for="username">Username</label>
            <input id="username" name="username" type="text" required />
          </div>
          <div class="field">
            <label for="password">Password</label>
            <input id="password" name="password" type="password" required />
          </div>
          <button type="submit">Sign in</button>
        </form>
      </div>
    </main>
  </body>
</html>`))
	}
}

func handleAdminLogout(store *adminStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie("admin_session"); err == nil {
			store.deleteSession(cookie.Value)
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "admin_session",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})
		http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
	}
}

func handleAdminSettings(store *adminStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err == nil {
			username := r.FormValue("admin_username")
			password := r.FormValue("admin_password")
			if username != "" && password != "" {
				store.updateAdmin(username, password)
				http.Redirect(w, r, "/admin?message=Admin+credentials+updated", http.StatusSeeOther)
				return
			}
		}
		http.Redirect(w, r, "/admin?message=Failed+to+update+credentials", http.StatusSeeOther)
	}
}

func handleAdminStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, adminStatusResponse{
		Service:   "typewriterpost-jmap-api",
		Status:    "ok",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func handleAdminTenants(store *adminStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenants := store.listTenants()
		payload := make([]adminTenant, 0, len(tenants))
		for _, tenant := range tenants {
			payload = append(payload, adminTenant{
				ID:       tenant.ID,
				Name:     tenant.Name,
				Plan:     tenant.Plan,
				Status:   tenant.Status,
				Domain:   tenant.Domain,
				UserCap:  tenant.UserCap,
				UserUsed: tenant.UserUsed,
			})
		}
		writeJSON(w, http.StatusOK, payload)
	}
}

func handleAdminTenantUsers(store *adminStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := chi.URLParam(r, "tenantId")
		tenant, ok := store.getTenant(tenantID)
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, tenant.Users)
	}
}

func handleAdminTenant(store *adminStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := chi.URLParam(r, "tenantId")
		tenant, ok := store.getTenant(tenantID)
		if !ok {
			http.NotFound(w, r)
			return
		}
		data := tenantPageData{
			Tenant:  tenant,
			Message: r.URL.Query().Get("message"),
		}
		base := schemeFromRequest(r) + "://" + r.Host
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8"/>
    <title>Tenant Management</title>
    <style>
      body { font-family: "Georgia", "Times New Roman", serif; background: #f8f5ef; color: #0d0d0d; margin: 0; }
      header { background: #0f2a1f; color: #f8f5ef; padding: 1.5rem 2rem; }
      main { padding: 2rem; }
      .card { background: #fff; padding: 1.5rem; border-radius: 16px; box-shadow: 0 10px 24px rgba(0,0,0,0.08); max-width: 900px; }
      .actions a { display: inline-block; margin-right: 0.6rem; padding: 0.35rem 0.7rem; border-radius: 999px; background: #b87333; color: #fff; text-decoration: none; }
      .actions a.secondary { background: #0f2a1f; }
      .field { margin: 0.6rem 0; }
      input { padding: 0.4rem 0.6rem; border-radius: 8px; border: 1px solid #e2d7c7; }
      form { margin-top: 1rem; }
      .message { background: #f1ede5; padding: 0.6rem 0.8rem; border-radius: 12px; margin-bottom: 1rem; }
      .grid { display: grid; gap: 1rem; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); }
      table { width: 100%; border-collapse: collapse; margin-top: 0.6rem; }
      th, td { text-align: left; padding: 0.5rem; border-bottom: 1px solid #efe6da; }
    </style>
  </head>
  <body>
    <header>
      <h1>Tenant Management</h1>
      <p>Tenant: ` + tenantID + `</p>
    </header>
    <main>
      <div class="card">
        <h2>Administration</h2>
        ` + renderMessage(data.Message) + `
        <p>Manage domain settings, users, quotas, licensing, and policy enforcement for this tenant.</p>
        <div class="actions">
          <a href="` + base + `/tenant/` + tenantID + `">Open tenant portal</a>
          <form method="post" action="` + base + `/admin/tenants/` + tenantID + `/remove" style="display:inline;">
            <button class="btn secondary" type="submit">Remove tenant</button>
          </form>
        </div>
      </div>
      <div class="card" style="margin-top: 1.5rem;">
        <h2>Tenant users</h2>
        <table>
          <thead>
            <tr>
              <th>Email</th>
              <th>Role</th>
              <th>Status</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            ` + renderTenantUsers(data.Tenant) + `
          </tbody>
        </table>
        <form method="post" action="` + base + `/admin/tenants/` + tenantID + `/users">
          <div class="grid">
            <div>
              <label for="user-email">User email</label>
              <input id="user-email" name="user_email" placeholder="user@` + tenantID + `" required />
            </div>
            <div>
              <label for="user-role">Role</label>
              <input id="user-role" name="user_role" placeholder="member" required />
            </div>
          </div>
          <button class="btn" type="submit">Add user</button>
        </form>
      </div>
      <div class="card" style="margin-top: 1.5rem;">
        <h2>Tenant domains</h2>
        <table>
          <thead>
            <tr>
              <th>Domain</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            ` + renderTenantDomains(base, data.Tenant) + `
          </tbody>
        </table>
        <form method="post" action="` + base + `/admin/tenants/` + tenantID + `/domains">
          <div class="field">
            <label for="domain-name">Domain</label>
            <input id="domain-name" name="domain_name" placeholder="example.com" required />
          </div>
          <button class="btn" type="submit">Add domain</button>
        </form>
      </div>
    </main>
  </body>
</html>`))
	}
}

func handleTenantPortal(store *adminStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := chi.URLParam(r, "tenantId")
		tenant, ok := store.getTenant(tenantID)
		if !ok {
			http.NotFound(w, r)
			return
		}
		base := schemeFromRequest(r) + "://" + r.Host
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8"/>
    <title>Tenant Admin Portal</title>
    <style>
      body { font-family: "Georgia", "Times New Roman", serif; background: #f8f5ef; color: #0d0d0d; margin: 0; }
      header { background: #b87333; color: #f8f5ef; padding: 1.5rem 2rem; }
      main { padding: 2rem; }
      .card { background: #fff; padding: 1.5rem; border-radius: 16px; box-shadow: 0 10px 24px rgba(0,0,0,0.08); max-width: 900px; }
      .grid { display: grid; gap: 1rem; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); }
      .metric { border: 1px solid #efe6da; border-radius: 12px; padding: 1rem; background: #fdfbf8; }
      .actions a { display: inline-block; margin-right: 0.6rem; padding: 0.35rem 0.7rem; border-radius: 999px; background: #0f2a1f; color: #fff; text-decoration: none; }
      .field { margin: 0.6rem 0; }
      input { padding: 0.4rem 0.6rem; border-radius: 8px; border: 1px solid #e2d7c7; }
    </style>
  </head>
  <body>
    <header>
      <h1>Tenant Admin Portal</h1>
      <p>Tenant: ` + tenantID + `</p>
    </header>
    <main>
      <div class="card">
        <h2>Tenant Overview</h2>
        <div class="grid">
          <div class="metric">
            <h3>Users</h3>
            <p>` + formatCount(tenant.UserUsed) + ` / ` + formatCount(tenant.UserCap) + ` active</p>
          </div>
          <div class="metric">
            <h3>Domains</h3>
            <p>` + tenant.Domain + `</p>
          </div>
          <div class="metric">
            <h3>Policies</h3>
            <p>Default security baseline</p>
          </div>
        </div>
        <div class="actions">
          <a href="` + base + `/tenant/` + tenantID + `/login">Tenant admin login</a>
        </div>
      </div>
      <div class="card" style="margin-top: 1.5rem;">
        <h2>Self-service mailbox signup</h2>
        <p>This tenant supports end-user self-service mailbox creation for the tenant domain.</p>
        <div class="grid">
          <div class="metric">
            <h3>Primary domain</h3>
            <p>` + tenant.Domain + `</p>
          </div>
          <div class="metric">
            <h3>Signup status</h3>
            <p>Enabled</p>
          </div>
        </div>
        <p>Planned flow: invite users, validate domain ownership, and provision JMAP mailboxes.</p>
      </div>
      <div class="card" style="margin-top: 1.5rem;">
        <h2>Tenant management</h2>
        <div class="grid">
          <div class="metric">
            <h3>Users</h3>
            <p>Create, suspend, and reset passwords for tenant users.</p>
          </div>
          <div class="metric">
            <h3>Domains</h3>
            <p>Assign domains, manage DNS records, and track verification.</p>
          </div>
          <div class="metric">
            <h3>Licensing</h3>
            <p>Manage seat limits, plan tier, and billing tags.</p>
          </div>
        </div>
        <div class="actions">
          <a href="` + base + `/tenant/` + tenantID + `/login">Open tenant admin</a>
        </div>
      </div>
    </main>
  </body>
</html>`))
	}
}

func handleTenantLogin(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8"/>
    <title>Tenant Admin Login</title>
    <style>
      body { font-family: "Georgia", "Times New Roman", serif; background: #f8f5ef; color: #0d0d0d; margin: 0; }
      header { background: #b87333; color: #f8f5ef; padding: 1.5rem 2rem; }
      main { padding: 2rem; }
      .card { background: #fff; padding: 1.5rem; border-radius: 16px; box-shadow: 0 10px 24px rgba(0,0,0,0.08); max-width: 640px; }
      .field { margin: 0.6rem 0; }
      input { width: 100%; padding: 0.5rem 0.6rem; border-radius: 8px; border: 1px solid #e2d7c7; }
      button { margin-top: 0.6rem; padding: 0.5rem 0.9rem; border-radius: 999px; border: none; background: #0f2a1f; color: #fff; font-weight: 600; }
      .note { color: #4f4a44; font-size: 0.9rem; }
    </style>
  </head>
  <body>
    <header>
      <h1>Tenant Admin Login</h1>
      <p>Tenant: ` + tenantID + `</p>
    </header>
    <main>
      <div class="card">
        <h2>Sign in</h2>
        <div class="field">
          <label for="email">Admin email</label>
          <input id="email" type="email" placeholder="admin@` + tenantID + `"/>
        </div>
        <div class="field">
          <label for="password">Password</label>
          <input id="password" type="password" placeholder="••••••••"/>
        </div>
        <button type="button">Sign in</button>
        <p class="note">Use default tenant admin credentials created by the Global Admin Center.</p>
      </div>
    </main>
  </body>
</html>`))
}

func requireLocalhost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			http.Error(w, "unable to determine client address", http.StatusForbidden)
			return
		}
		addr, err := netip.ParseAddr(host)
		if err != nil || !addr.IsLoopback() {
			http.Error(w, "admin portal is available on localhost only", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requireAdminSession(store *adminStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("admin_session")
			if err != nil || !store.sessionValid(cookie.Value) {
				http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func handleAdminTenantCreate(store *adminStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err == nil {
			id := r.FormValue("tenant_id")
			name := r.FormValue("tenant_name")
			domain := r.FormValue("tenant_domain")
			plan := r.FormValue("tenant_plan")
			if id != "" && name != "" && domain != "" {
				if err := store.addTenant(id, name, domain, plan); err == nil {
					http.Redirect(w, r, "/admin?message=Tenant+created", http.StatusSeeOther)
					return
				}
			}
		}
		http.Redirect(w, r, "/admin?message=Failed+to+create+tenant", http.StatusSeeOther)
	}
}

func handleAdminTenantRemove(store *adminStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := chi.URLParam(r, "tenantId")
		store.removeTenant(tenantID)
		http.Redirect(w, r, "/admin?message=Tenant+removed", http.StatusSeeOther)
	}
}

func handleAdminTenantAddUser(store *adminStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := chi.URLParam(r, "tenantId")
		if err := r.ParseForm(); err == nil {
			email := r.FormValue("user_email")
			role := r.FormValue("user_role")
			if email != "" && role != "" {
				if err := store.addTenantUser(tenantID, email, role); err == nil {
					http.Redirect(w, r, "/admin/tenants/"+tenantID+"?message=User+added", http.StatusSeeOther)
					return
				}
			}
		}
		http.Redirect(w, r, "/admin/tenants/"+tenantID+"?message=Failed+to+add+user", http.StatusSeeOther)
	}
}

func handleAdminTenantRemoveUser(store *adminStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := chi.URLParam(r, "tenantId")
		if err := r.ParseForm(); err == nil {
			email := r.FormValue("user_email")
			if email != "" {
				store.removeTenantUser(tenantID, email)
				http.Redirect(w, r, "/admin/tenants/"+tenantID+"?message=User+removed", http.StatusSeeOther)
				return
			}
		}
		http.Redirect(w, r, "/admin/tenants/"+tenantID+"?message=Failed+to+remove+user", http.StatusSeeOther)
	}
}

func handleAdminTenantAddDomain(store *adminStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := chi.URLParam(r, "tenantId")
		if err := r.ParseForm(); err == nil {
			domain := r.FormValue("domain_name")
			if domain != "" {
				if err := store.addTenantDomain(tenantID, domain); err == nil {
					http.Redirect(w, r, "/admin/tenants/"+tenantID+"?message=Domain+added", http.StatusSeeOther)
					return
				}
			}
		}
		http.Redirect(w, r, "/admin/tenants/"+tenantID+"?message=Failed+to+add+domain", http.StatusSeeOther)
	}
}

func handleAdminTenantRemoveDomain(store *adminStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := chi.URLParam(r, "tenantId")
		if err := r.ParseForm(); err == nil {
			domain := r.FormValue("domain_name")
			if domain != "" {
				store.removeTenantDomain(tenantID, domain)
				http.Redirect(w, r, "/admin/tenants/"+tenantID+"?message=Domain+removed", http.StatusSeeOther)
				return
			}
		}
		http.Redirect(w, r, "/admin/tenants/"+tenantID+"?message=Failed+to+remove+domain", http.StatusSeeOther)
	}
}

func renderMessage(message string) string {
	if message == "" {
		return ""
	}
	return `<div class="message">` + message + `</div>`
}

func renderTenantRows(base string, tenants []*tenantRecord) string {
	if len(tenants) == 0 {
		return `<tr><td colspan="6">No tenants found.</td></tr>`
	}
	rows := ""
	for _, tenant := range tenants {
		rows += `<tr>
  <td>` + tenant.Name + `</td>
  <td>` + tenant.Domain + `</td>
  <td>` + tenant.Plan + `</td>
  <td>` + formatCount(tenant.UserUsed) + ` / ` + formatCount(tenant.UserCap) + `</td>
  <td class="actions">
    <a href="` + base + `/admin/tenants/` + tenant.ID + `">Manage</a>
    <a class="secondary" href="` + base + `/tenant/` + tenant.ID + `">Take over</a>
    <form method="post" action="` + base + `/admin/tenants/` + tenant.ID + `/remove" style="display:inline;">
      <button class="btn secondary" type="submit">Remove</button>
    </form>
  </td>
  <td><span class="pill">` + tenant.Status + `</span></td>
</tr>`
	}
	return rows
}

func renderTenantUsers(tenant *tenantRecord) string {
	if tenant == nil || len(tenant.Users) == 0 {
		return `<tr><td colspan="4">No users yet.</td></tr>`
	}
	rows := ""
	for _, user := range tenant.Users {
		rows += `<tr><td>` + user.Email + `</td><td>` + user.Role + `</td><td>` + user.Status + `</td><td>
      <form method="post" action="/admin/tenants/` + tenant.ID + `/users/remove" style="display:inline;">
        <input type="hidden" name="user_email" value="` + user.Email + `" />
        <button class="btn secondary" type="submit">Remove</button>
      </form>
    </td></tr>`
	}
	return rows
}

func renderTenantDomains(base string, tenant *tenantRecord) string {
	if tenant == nil || len(tenant.Domains) == 0 {
		return `<tr><td colspan="2">No domains assigned.</td></tr>`
	}
	rows := ""
	for _, domain := range tenant.Domains {
		rows += `<tr><td>` + domain + `</td><td>
      <form method="post" action="` + base + `/admin/tenants/` + tenant.ID + `/domains/remove" style="display:inline;">
        <input type="hidden" name="domain_name" value="` + domain + `" />
        <button class="btn secondary" type="submit">Remove</button>
      </form>
    </td></tr>`
	}
	return rows
}

func formatCount(value int) string {
	return strconv.Itoa(value)
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
