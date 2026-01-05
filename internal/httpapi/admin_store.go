package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

type adminStore struct {
	mu       sync.Mutex
	admin    adminCredentials
	tenants  map[string]*tenantRecord
	sessions map[string]sessionRecord
}

type adminCredentials struct {
	Username string
	Password string
	Updated  time.Time
}

type tenantRecord struct {
	ID       string
	Name     string
	Plan     string
	Status   string
	Domain   string
	UserCap  int
	UserUsed int
	Users    []adminUser
	Domains  []string
	Contact  string
	Admin    adminCredentials
}

type sessionRecord struct {
	Username  string
	CreatedAt time.Time
}

func newAdminStore() *adminStore {
	store := &adminStore{
		admin: adminCredentials{
			Username: "global-admin",
			Password: "change-me-now",
			Updated:  time.Now().UTC(),
		},
		tenants:  map[string]*tenantRecord{},
		sessions: map[string]sessionRecord{},
	}
	store.tenants["default"] = &tenantRecord{
		ID:       "default",
		Name:     "Default Tenant",
		Plan:     "enterprise",
		Status:   "active",
		Domain:   "example.local",
		UserCap:  -1,
		UserUsed: 1,
		Users: []adminUser{
			{
				ID:     "user-1",
				Email:  "admin@example.local",
				Role:   "owner",
				Status: "active",
			},
		},
		Domains: []string{"example.local"},
		Contact: "admin@example.local",
		Admin: adminCredentials{
			Username: "tenant-admin",
			Password: "change-me-now",
			Updated:  time.Now().UTC(),
		},
	}
	return store
}

func (s *adminStore) authenticate(username, password string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.admin.Username == username && s.admin.Password == password
}

func (s *adminStore) updateAdmin(username, password string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.admin = adminCredentials{
		Username: username,
		Password: password,
		Updated:  time.Now().UTC(),
	}
}

func (s *adminStore) currentAdmin() adminCredentials {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.admin
}

func (s *adminStore) newSession(username string) (string, error) {
	token, err := randomToken()
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[token] = sessionRecord{Username: username, CreatedAt: time.Now().UTC()}
	return token, nil
}

func (s *adminStore) sessionValid(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.sessions[token]
	return ok
}

func (s *adminStore) deleteSession(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}

func (s *adminStore) listTenants() []*tenantRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	tenants := make([]*tenantRecord, 0, len(s.tenants))
	for _, tenant := range s.tenants {
		tenants = append(tenants, tenant)
	}
	return tenants
}

func (s *adminStore) getTenant(id string) (*tenantRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tenant, ok := s.tenants[id]
	return tenant, ok
}

func (s *adminStore) addTenant(id, name, domain, plan string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.tenants[id]; exists {
		return errors.New("tenant already exists")
	}
	userCap := 10
	if plan == "enterprise" {
		userCap = -1
	}
	s.tenants[id] = &tenantRecord{
		ID:       id,
		Name:     name,
		Plan:     plan,
		Status:   "active",
		Domain:   domain,
		UserCap:  userCap,
		UserUsed: 0,
		Users:    []adminUser{},
		Domains:  []string{domain},
		Contact:  "admin@" + domain,
		Admin: adminCredentials{
			Username: "tenant-admin",
			Password: "change-me-now",
			Updated:  time.Now().UTC(),
		},
	}
	return nil
}

func (s *adminStore) removeTenant(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tenants, id)
}

func (s *adminStore) addTenantUser(id, email, role string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tenant, ok := s.tenants[id]
	if !ok {
		return errors.New("tenant not found")
	}
	userID := "user-" + randomID()
	tenant.Users = append(tenant.Users, adminUser{
		ID:     userID,
		Email:  email,
		Role:   role,
		Status: "active",
	})
	tenant.UserUsed = len(tenant.Users)
	return nil
}

func (s *adminStore) addTenantDomain(id, domain string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tenant, ok := s.tenants[id]
	if !ok {
		return errors.New("tenant not found")
	}
	tenant.Domains = append(tenant.Domains, domain)
	tenant.Domain = tenant.Domains[0]
	return nil
}

func (s *adminStore) removeTenantUser(id, email string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tenant, ok := s.tenants[id]
	if !ok {
		return
	}
	filtered := tenant.Users[:0]
	for _, user := range tenant.Users {
		if user.Email != email {
			filtered = append(filtered, user)
		}
	}
	tenant.Users = filtered
	tenant.UserUsed = len(tenant.Users)
}

func (s *adminStore) removeTenantDomain(id, domain string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tenant, ok := s.tenants[id]
	if !ok {
		return
	}
	filtered := tenant.Domains[:0]
	for _, existing := range tenant.Domains {
		if existing != domain {
			filtered = append(filtered, existing)
		}
	}
	if len(filtered) == 0 {
		filtered = append(filtered, tenant.Domain)
	}
	tenant.Domains = filtered
	tenant.Domain = tenant.Domains[0]
}

func (s *adminStore) updateTenantAdmin(id, username, password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tenant, ok := s.tenants[id]
	if !ok {
		return errors.New("tenant not found")
	}
	tenant.Admin = adminCredentials{
		Username: username,
		Password: password,
		Updated:  time.Now().UTC(),
	}
	return nil
}

func (s *adminStore) updateTenantDetails(id, name, contact, plan string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tenant, ok := s.tenants[id]
	if !ok {
		return errors.New("tenant not found")
	}
	if name != "" {
		tenant.Name = name
	}
	if contact != "" {
		tenant.Contact = contact
	}
	if plan != "" {
		tenant.Plan = plan
		if plan == "enterprise" {
			tenant.UserCap = -1
		} else if tenant.UserCap < 0 {
			tenant.UserCap = 10
		}
	}
	return nil
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func randomID() string {
	buf := make([]byte, 4)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}
