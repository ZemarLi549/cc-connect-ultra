package core

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// EnterpriseTenant models a company/org that owns users, spaces, and shared skills.
type EnterpriseTenant struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Slug        string            `json:"slug"`
	Description string            `json:"description,omitempty"`
	Status      string            `json:"status,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// EnterpriseUser models a human end-user inside a tenant.
type EnterpriseUser struct {
	ID           string            `json:"id"`
	TenantID     string            `json:"tenant_id"`
	Email        string            `json:"email,omitempty"`
	DisplayName  string            `json:"display_name"`
	Role         string            `json:"role,omitempty"`
	Status       string            `json:"status,omitempty"`
	TokenBudget  int64             `json:"token_budget,omitempty"`
	TokenUsed    int64             `json:"token_used,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	LastActiveAt time.Time         `json:"last_active_at,omitempty"`
}

// EnterpriseSpace models an isolated personal/team workspace.
type EnterpriseSpace struct {
	ID                string            `json:"id"`
	TenantID          string            `json:"tenant_id"`
	OwnerUserID       string            `json:"owner_user_id"`
	Name              string            `json:"name"`
	Slug              string            `json:"slug"`
	WorkspaceDir      string            `json:"workspace_dir,omitempty"`
	ProjectName       string            `json:"project_name,omitempty"`
	Visibility        string            `json:"visibility,omitempty"`
	Status            string            `json:"status,omitempty"`
	CurrentProvider   string            `json:"current_provider,omitempty"`
	CurrentModel      string            `json:"current_model,omitempty"`
	SharedSkillIDs    []string          `json:"shared_skill_ids,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
	LastInteractionAt time.Time         `json:"last_interaction_at,omitempty"`
}

// EnterpriseSkill models a private/tenant/public skill asset.
type EnterpriseSkill struct {
	ID          string            `json:"id"`
	TenantID    string            `json:"tenant_id,omitempty"`
	OwnerUserID string            `json:"owner_user_id,omitempty"`
	Name        string            `json:"name"`
	DisplayName string            `json:"display_name,omitempty"`
	Description string            `json:"description,omitempty"`
	Scope       string            `json:"scope,omitempty"`  // private, tenant, public
	Status      string            `json:"status,omitempty"` // draft, published, archived
	Version     string            `json:"version,omitempty"`
	Prompt      string            `json:"prompt,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	SourcePath  string            `json:"source_path,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// EnterpriseProvider models a switchable model/provider entry exposed to users.
type EnterpriseProvider struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	DisplayName  string            `json:"display_name,omitempty"`
	ProviderType string            `json:"provider_type,omitempty"`
	BaseURL      string            `json:"base_url,omitempty"`
	DefaultModel string            `json:"default_model,omitempty"`
	Models       []string          `json:"models,omitempty"`
	Status       string            `json:"status,omitempty"`
	Tags         []string          `json:"tags,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// EnterpriseBot models a shared or user-scoped assistant exposed inside the enterprise.
type EnterpriseBot struct {
	ID           string            `json:"id"`
	TenantID     string            `json:"tenant_id,omitempty"`
	SpaceID      string            `json:"space_id,omitempty"`
	OwnerUserID  string            `json:"owner_user_id,omitempty"`
	Name         string            `json:"name"`
	Slug         string            `json:"slug"`
	Description  string            `json:"description,omitempty"`
	Scope        string            `json:"scope,omitempty"` // personal, team, tenant, public
	ProviderName string            `json:"provider_name,omitempty"`
	ModelName    string            `json:"model_name,omitempty"`
	SkillIDs     []string          `json:"skill_ids,omitempty"`
	Status       string            `json:"status,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// EnterpriseSQLConfig is the future database backend configuration.
type EnterpriseSQLConfig struct {
	Driver              string `json:"driver,omitempty"`
	DSN                 string `json:"dsn,omitempty"`
	MaxOpenConns        int    `json:"max_open_conns,omitempty"`
	MaxIdleConns        int    `json:"max_idle_conns,omitempty"`
	ConnMaxLifetimeSecs int    `json:"conn_max_lifetime_secs,omitempty"`
}

// EnterpriseRedisConfig is the future cache/session backend configuration.
type EnterpriseRedisConfig struct {
	Addr      string `json:"addr,omitempty"`
	Password  string `json:"password,omitempty"`
	DB        int    `json:"db,omitempty"`
	KeyPrefix string `json:"key_prefix,omitempty"`
}

// EnterpriseCocoloopConfig stores import integration details.
type EnterpriseCocoloopConfig struct {
	Enabled      bool      `json:"enabled,omitempty"`
	BaseURL      string    `json:"base_url,omitempty"`
	APIKey       string    `json:"api_key,omitempty"`
	Workspace    string    `json:"workspace,omitempty"`
	LastImportAt time.Time `json:"last_import_at,omitempty"`
}

// EnterpriseAIOpsSettings holds system-wide AIOps configuration for the enterprise console.
type EnterpriseAIOpsSettings struct {
	OrganizationName    string                   `json:"organization_name,omitempty"`
	DefaultProjectName  string                   `json:"default_project_name,omitempty"`
	DefaultSpaceBaseDir string                   `json:"default_space_base_dir,omitempty"`
	Postgres            EnterpriseSQLConfig      `json:"postgres"`
	Redis               EnterpriseRedisConfig    `json:"redis"`
	Cocoloop            EnterpriseCocoloopConfig `json:"cocoloop"`
	UpdatedAt           time.Time                `json:"updated_at,omitempty"`
}

// EnterpriseUsageRecord captures per-request accounting for ranking, quotas, and cost.
type EnterpriseUsageRecord struct {
	ID               string            `json:"id"`
	TenantID         string            `json:"tenant_id,omitempty"`
	UserID           string            `json:"user_id,omitempty"`
	SpaceID          string            `json:"space_id,omitempty"`
	ProjectName      string            `json:"project_name,omitempty"`
	ProviderName     string            `json:"provider_name,omitempty"`
	ModelName        string            `json:"model_name,omitempty"`
	RequestKind      string            `json:"request_kind,omitempty"`
	PromptTokens     int64             `json:"prompt_tokens,omitempty"`
	CompletionTokens int64             `json:"completion_tokens,omitempty"`
	TotalTokens      int64             `json:"total_tokens,omitempty"`
	CostMicros       int64             `json:"cost_micros,omitempty"`
	LatencyMs        int64             `json:"latency_ms,omitempty"`
	OccurredAt       time.Time         `json:"occurred_at"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

// EnterpriseSkillImportRequest records an import job from external skill ecosystems.
type EnterpriseSkillImportRequest struct {
	ID             string            `json:"id"`
	TenantID       string            `json:"tenant_id,omitempty"`
	OwnerUserID    string            `json:"owner_user_id,omitempty"`
	SourceType     string            `json:"source_type,omitempty"` // cocoloop, git, zip, manual
	SourceName     string            `json:"source_name,omitempty"`
	SourceRef      string            `json:"source_ref,omitempty"`
	Status         string            `json:"status,omitempty"`
	ImportedSkills int               `json:"imported_skills,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	CompletedAt    time.Time         `json:"completed_at,omitempty"`
}

// EnterpriseTask models a task, goal, or reminder inside a tenant/workspace.
type EnterpriseTask struct {
	ID              string            `json:"id"`
	TenantID        string            `json:"tenant_id,omitempty"`
	SpaceID         string            `json:"space_id,omitempty"`
	OwnerUserID     string            `json:"owner_user_id,omitempty"`
	AssigneeUserID  string            `json:"assignee_user_id,omitempty"`
	ParentTaskID    string            `json:"parent_task_id,omitempty"`
	Title           string            `json:"title"`
	Description     string            `json:"description,omitempty"`
	TaskType        string            `json:"task_type,omitempty"` // task, goal, reminder
	Priority        string            `json:"priority,omitempty"`  // urgent, high, medium, low, none
	Status          string            `json:"status,omitempty"`    // todo, in_progress, done, cancelled
	Tags            []string          `json:"tags,omitempty"`
	DueAt           time.Time         `json:"due_at,omitempty"`
	ReminderAt      time.Time         `json:"reminder_at,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	CompletedAt     time.Time         `json:"completed_at,omitempty"`
}

// EnterpriseTaskFilter filters task queries in the enterprise control plane.
type EnterpriseTaskFilter struct {
	TenantID       string
	SpaceID        string
	OwnerUserID    string
	AssigneeUserID string
	TaskType       string
	Status         string
	Priority       string
	Tag            string
	Query          string
}

// EnterpriseOverview summarizes the current enterprise state.
type EnterpriseOverview struct {
	TenantsCount    int   `json:"tenants_count"`
	UsersCount      int   `json:"users_count"`
	SpacesCount     int   `json:"spaces_count"`
	SkillsCount     int   `json:"skills_count"`
	BotsCount       int   `json:"bots_count"`
	RolesCount      int   `json:"roles_count"`
	ProjectsCount   int   `json:"projects_count"`
	TasksCount      int   `json:"tasks_count"`
	ProvidersCount  int   `json:"providers_count"`
	ImportsCount    int   `json:"imports_count"`
	UsageCount      int   `json:"usage_count"`
	TotalTokens     int64 `json:"total_tokens"`
	TotalCostMicros int64 `json:"total_cost_micros"`
}

// EnterpriseUsageFilter filters usage list queries.
type EnterpriseUsageFilter struct {
	TenantID string
	UserID   string
	SpaceID  string
	Provider string
	Limit    int
}

// EnterpriseLeaderboardEntry is an aggregate ranking row.
type EnterpriseLeaderboardEntry struct {
	SubjectType string `json:"subject_type"`
	SubjectID   string `json:"subject_id"`
	SubjectName string `json:"subject_name"`
	Requests    int64  `json:"requests"`
	Tokens      int64  `json:"tokens"`
	CostMicros  int64  `json:"cost_micros"`
}

type enterpriseSnapshot struct {
	Version      int                            `json:"version"`
	UpdatedAt    time.Time                      `json:"updated_at"`
	Tenants      []EnterpriseTenant             `json:"tenants"`
	Users        []EnterpriseUser               `json:"users"`
	Spaces       []EnterpriseSpace              `json:"spaces"`
	Skills       []EnterpriseSkill              `json:"skills"`
	Bots         []EnterpriseBot                `json:"bots"`
	Roles        []EnterpriseRole               `json:"roles"`
	RoleBindings []EnterpriseRoleBinding        `json:"role_bindings"`
	Projects     []EnterpriseProjectProfile     `json:"projects"`
	Tasks        []EnterpriseTask               `json:"tasks"`
	Providers    []EnterpriseProvider           `json:"providers"`
	Imports      []EnterpriseSkillImportRequest `json:"imports"`
	Settings     EnterpriseAIOpsSettings        `json:"settings"`
	Usage        []EnterpriseUsageRecord        `json:"usage"`
}

// EnterpriseStore is a lightweight file-backed store for enterprise control-plane data.
type EnterpriseStore struct {
	mu       sync.RWMutex
	path     string
	snapshot enterpriseSnapshot
}

func NewEnterpriseStore(path string) *EnterpriseStore {
	s := &EnterpriseStore{
		path: path,
		snapshot: enterpriseSnapshot{
			Version: 1,
		},
	}
	if path != "" {
		s.loadLocked()
	}
	return s
}

func (s *EnterpriseStore) ListTenants() []EnterpriseTenant {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := append([]EnterpriseTenant(nil), s.snapshot.Tenants...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *EnterpriseStore) UpsertTenant(item EnterpriseTenant) (EnterpriseTenant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	item.ID = ensureEnterpriseID("tenant", item.ID)
	item.Name = strings.TrimSpace(item.Name)
	if item.Name == "" {
		return EnterpriseTenant{}, fmt.Errorf("tenant name is required")
	}
	if item.Slug == "" {
		item.Slug = enterpriseSlug(item.Name, item.ID)
	}
	item.UpdatedAt = now
	found := false
	for i := range s.snapshot.Tenants {
		if s.snapshot.Tenants[i].ID != item.ID {
			continue
		}
		found = true
		item.CreatedAt = s.snapshot.Tenants[i].CreatedAt
		if item.CreatedAt.IsZero() {
			item.CreatedAt = now
		}
		s.snapshot.Tenants[i] = item
		break
	}
	if !found {
		item.CreatedAt = now
		s.snapshot.Tenants = append(s.snapshot.Tenants, item)
	}
	return item, s.saveLocked()
}

func (s *EnterpriseStore) ListUsers(tenantID string) []EnterpriseUser {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []EnterpriseUser
	for _, item := range s.snapshot.Users {
		if tenantID != "" && item.TenantID != tenantID {
			continue
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DisplayName < out[j].DisplayName })
	return out
}

func (s *EnterpriseStore) UpsertUser(item EnterpriseUser) (EnterpriseUser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	item.ID = ensureEnterpriseID("user", item.ID)
	item.DisplayName = strings.TrimSpace(item.DisplayName)
	if item.DisplayName == "" {
		item.DisplayName = strings.TrimSpace(item.Email)
	}
	if item.DisplayName == "" {
		return EnterpriseUser{}, fmt.Errorf("user display_name or email is required")
	}
	item.UpdatedAt = now
	found := false
	for i := range s.snapshot.Users {
		if s.snapshot.Users[i].ID != item.ID {
			continue
		}
		found = true
		item.CreatedAt = s.snapshot.Users[i].CreatedAt
		if item.CreatedAt.IsZero() {
			item.CreatedAt = now
		}
		if item.LastActiveAt.IsZero() {
			item.LastActiveAt = s.snapshot.Users[i].LastActiveAt
		}
		s.snapshot.Users[i] = item
		break
	}
	if !found {
		item.CreatedAt = now
		s.snapshot.Users = append(s.snapshot.Users, item)
	}
	return item, s.saveLocked()
}

func (s *EnterpriseStore) ListSpaces(tenantID, ownerUserID string) []EnterpriseSpace {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []EnterpriseSpace
	for _, item := range s.snapshot.Spaces {
		if tenantID != "" && item.TenantID != tenantID {
			continue
		}
		if ownerUserID != "" && item.OwnerUserID != ownerUserID {
			continue
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *EnterpriseStore) UpsertSpace(item EnterpriseSpace) (EnterpriseSpace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	item.ID = ensureEnterpriseID("space", item.ID)
	item.Name = strings.TrimSpace(item.Name)
	if item.Name == "" {
		return EnterpriseSpace{}, fmt.Errorf("space name is required")
	}
	if item.Slug == "" {
		item.Slug = enterpriseSlug(item.Name, item.ID)
	}
	item.UpdatedAt = now
	found := false
	for i := range s.snapshot.Spaces {
		if s.snapshot.Spaces[i].ID != item.ID {
			continue
		}
		found = true
		item.CreatedAt = s.snapshot.Spaces[i].CreatedAt
		if item.CreatedAt.IsZero() {
			item.CreatedAt = now
		}
		if item.LastInteractionAt.IsZero() {
			item.LastInteractionAt = s.snapshot.Spaces[i].LastInteractionAt
		}
		s.snapshot.Spaces[i] = item
		break
	}
	if !found {
		item.CreatedAt = now
		s.snapshot.Spaces = append(s.snapshot.Spaces, item)
	}
	return item, s.saveLocked()
}

func (s *EnterpriseStore) ListSkills(scope, tenantID, ownerUserID string) []EnterpriseSkill {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []EnterpriseSkill
	for _, item := range s.snapshot.Skills {
		if scope != "" && item.Scope != scope {
			continue
		}
		if tenantID != "" && item.TenantID != tenantID {
			continue
		}
		if ownerUserID != "" && item.OwnerUserID != ownerUserID {
			continue
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *EnterpriseStore) UpsertSkill(item EnterpriseSkill) (EnterpriseSkill, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	item.ID = ensureEnterpriseID("skill", item.ID)
	item.Name = strings.TrimSpace(item.Name)
	if item.Name == "" {
		return EnterpriseSkill{}, fmt.Errorf("skill name is required")
	}
	if item.Scope == "" {
		item.Scope = "private"
	}
	if item.Status == "" {
		item.Status = "draft"
	}
	item.UpdatedAt = now
	found := false
	for i := range s.snapshot.Skills {
		if s.snapshot.Skills[i].ID != item.ID {
			continue
		}
		found = true
		item.CreatedAt = s.snapshot.Skills[i].CreatedAt
		if item.CreatedAt.IsZero() {
			item.CreatedAt = now
		}
		s.snapshot.Skills[i] = item
		break
	}
	if !found {
		item.CreatedAt = now
		s.snapshot.Skills = append(s.snapshot.Skills, item)
	}
	return item, s.saveLocked()
}

func (s *EnterpriseStore) ListProviders() []EnterpriseProvider {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := append([]EnterpriseProvider(nil), s.snapshot.Providers...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *EnterpriseStore) ListBots(tenantID, ownerUserID string) []EnterpriseBot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []EnterpriseBot
	for _, item := range s.snapshot.Bots {
		if tenantID != "" && item.TenantID != tenantID {
			continue
		}
		if ownerUserID != "" && item.OwnerUserID != ownerUserID {
			continue
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *EnterpriseStore) UpsertBot(item EnterpriseBot) (EnterpriseBot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	item.ID = ensureEnterpriseID("bot", item.ID)
	item.Name = strings.TrimSpace(item.Name)
	if item.Name == "" {
		return EnterpriseBot{}, fmt.Errorf("bot name is required")
	}
	if item.Slug == "" {
		item.Slug = enterpriseSlug(item.Name, item.ID)
	}
	if item.Status == "" {
		item.Status = "active"
	}
	if item.Scope == "" {
		item.Scope = "tenant"
	}
	item.UpdatedAt = now
	found := false
	for i := range s.snapshot.Bots {
		if s.snapshot.Bots[i].ID != item.ID {
			continue
		}
		found = true
		item.CreatedAt = s.snapshot.Bots[i].CreatedAt
		if item.CreatedAt.IsZero() {
			item.CreatedAt = now
		}
		s.snapshot.Bots[i] = item
		break
	}
	if !found {
		item.CreatedAt = now
		s.snapshot.Bots = append(s.snapshot.Bots, item)
	}
	return item, s.saveLocked()
}

func (s *EnterpriseStore) UpsertProvider(item EnterpriseProvider) (EnterpriseProvider, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	item.ID = ensureEnterpriseID("provider", item.ID)
	item.Name = strings.TrimSpace(item.Name)
	if item.Name == "" {
		return EnterpriseProvider{}, fmt.Errorf("provider name is required")
	}
	if item.Status == "" {
		item.Status = "enabled"
	}
	item.UpdatedAt = now
	found := false
	for i := range s.snapshot.Providers {
		if s.snapshot.Providers[i].ID != item.ID {
			continue
		}
		found = true
		item.CreatedAt = s.snapshot.Providers[i].CreatedAt
		if item.CreatedAt.IsZero() {
			item.CreatedAt = now
		}
		s.snapshot.Providers[i] = item
		break
	}
	if !found {
		item.CreatedAt = now
		s.snapshot.Providers = append(s.snapshot.Providers, item)
	}
	return item, s.saveLocked()
}

func (s *EnterpriseStore) GetSettings() EnterpriseAIOpsSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snapshot.Settings
}

func (s *EnterpriseStore) SaveSettings(item EnterpriseAIOpsSettings) (EnterpriseAIOpsSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item.UpdatedAt = time.Now().UTC()
	if item.Postgres.Driver == "" {
		item.Postgres.Driver = "postgres"
	}
	s.snapshot.Settings = item
	return item, s.saveLocked()
}

func (s *EnterpriseStore) ListImports(tenantID string) []EnterpriseSkillImportRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []EnterpriseSkillImportRequest
	for _, item := range s.snapshot.Imports {
		if tenantID != "" && item.TenantID != tenantID {
			continue
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func (s *EnterpriseStore) AddImport(item EnterpriseSkillImportRequest) (EnterpriseSkillImportRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	item.ID = ensureEnterpriseID("import", item.ID)
	if item.SourceType == "" {
		item.SourceType = "manual"
	}
	if item.Status == "" {
		item.Status = "queued"
	}
	item.CreatedAt = now
	item.UpdatedAt = now
	s.snapshot.Imports = append(s.snapshot.Imports, item)
	if strings.EqualFold(item.SourceType, "cocoloop") {
		s.snapshot.Settings.Cocoloop.LastImportAt = now
	}
	return item, s.saveLocked()
}

func (s *EnterpriseStore) ListUsage(filter EnterpriseUsageFilter) []EnterpriseUsageRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]EnterpriseUsageRecord, 0, len(s.snapshot.Usage))
	for _, item := range s.snapshot.Usage {
		if filter.TenantID != "" && item.TenantID != filter.TenantID {
			continue
		}
		if filter.UserID != "" && item.UserID != filter.UserID {
			continue
		}
		if filter.SpaceID != "" && item.SpaceID != filter.SpaceID {
			continue
		}
		if filter.Provider != "" && item.ProviderName != filter.Provider {
			continue
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].OccurredAt.After(out[j].OccurredAt) })
	if filter.Limit > 0 && len(out) > filter.Limit {
		out = out[:filter.Limit]
	}
	return out
}

func (s *EnterpriseStore) AddUsage(item EnterpriseUsageRecord) (EnterpriseUsageRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item.ID = ensureEnterpriseID("usage", item.ID)
	if item.OccurredAt.IsZero() {
		item.OccurredAt = time.Now().UTC()
	}
	if item.TotalTokens == 0 {
		item.TotalTokens = item.PromptTokens + item.CompletionTokens
	}
	s.snapshot.Usage = append(s.snapshot.Usage, item)
	s.bumpUsageRollupsLocked(item)
	return item, s.saveLocked()
}

func (s *EnterpriseStore) Overview() EnterpriseOverview {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var totalTokens int64
	var totalCostMicros int64
	for _, item := range s.snapshot.Usage {
		totalTokens += item.TotalTokens
		totalCostMicros += item.CostMicros
	}
	return EnterpriseOverview{
		TenantsCount:    len(s.snapshot.Tenants),
		UsersCount:      len(s.snapshot.Users),
		SpacesCount:     len(s.snapshot.Spaces),
		SkillsCount:     len(s.snapshot.Skills),
		BotsCount:       len(s.snapshot.Bots),
		RolesCount:      len(s.snapshot.Roles),
		ProjectsCount:   len(s.snapshot.Projects),
		ProvidersCount:  len(s.snapshot.Providers),
		ImportsCount:    len(s.snapshot.Imports),
		UsageCount:      len(s.snapshot.Usage),
		TotalTokens:     totalTokens,
		TotalCostMicros: totalCostMicros,
	}
}

func (s *EnterpriseStore) Leaderboard(groupBy string, limit int) []EnterpriseLeaderboardEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if groupBy == "" {
		groupBy = "user"
	}

	type aggregate struct {
		EnterpriseLeaderboardEntry
	}
	acc := make(map[string]*aggregate)
	for _, item := range s.snapshot.Usage {
		subjectID := ""
		subjectName := ""
		switch groupBy {
		case "tenant":
			subjectID = item.TenantID
			subjectName = s.tenantNameLocked(item.TenantID)
		case "space":
			subjectID = item.SpaceID
			subjectName = s.spaceNameLocked(item.SpaceID)
		default:
			subjectID = item.UserID
			subjectName = s.userNameLocked(item.UserID)
			groupBy = "user"
		}
		if strings.TrimSpace(subjectID) == "" {
			continue
		}
		a := acc[subjectID]
		if a == nil {
			a = &aggregate{EnterpriseLeaderboardEntry: EnterpriseLeaderboardEntry{
				SubjectType: groupBy,
				SubjectID:   subjectID,
				SubjectName: subjectName,
			}}
			acc[subjectID] = a
		}
		a.Requests++
		a.Tokens += item.TotalTokens
		a.CostMicros += item.CostMicros
	}

	out := make([]EnterpriseLeaderboardEntry, 0, len(acc))
	for _, item := range acc {
		out = append(out, item.EnterpriseLeaderboardEntry)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Tokens == out[j].Tokens {
			return out[i].SubjectName < out[j].SubjectName
		}
		return out[i].Tokens > out[j].Tokens
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func (s *EnterpriseStore) bumpUsageRollupsLocked(item EnterpriseUsageRecord) {
	if item.UserID != "" {
		for i := range s.snapshot.Users {
			if s.snapshot.Users[i].ID == item.UserID {
				s.snapshot.Users[i].TokenUsed += item.TotalTokens
				s.snapshot.Users[i].LastActiveAt = item.OccurredAt
				s.snapshot.Users[i].UpdatedAt = time.Now().UTC()
				break
			}
		}
	}
	if item.SpaceID != "" {
		for i := range s.snapshot.Spaces {
			if s.snapshot.Spaces[i].ID == item.SpaceID {
				s.snapshot.Spaces[i].LastInteractionAt = item.OccurredAt
				s.snapshot.Spaces[i].UpdatedAt = time.Now().UTC()
				if item.ProviderName != "" {
					s.snapshot.Spaces[i].CurrentProvider = item.ProviderName
				}
				if item.ModelName != "" {
					s.snapshot.Spaces[i].CurrentModel = item.ModelName
				}
				break
			}
		}
	}
}

func (s *EnterpriseStore) tenantNameLocked(id string) string {
	for _, item := range s.snapshot.Tenants {
		if item.ID == id {
			return item.Name
		}
	}
	return id
}

func (s *EnterpriseStore) userNameLocked(id string) string {
	for _, item := range s.snapshot.Users {
		if item.ID == id {
			return item.DisplayName
		}
	}
	return id
}

func (s *EnterpriseStore) spaceNameLocked(id string) string {
	for _, item := range s.snapshot.Spaces {
		if item.ID == id {
			return item.Name
		}
	}
	return id
}

func (s *EnterpriseStore) saveLocked() error {
	s.snapshot.Version = 1
	s.snapshot.UpdatedAt = time.Now().UTC()
	if s.path == "" {
		return nil
	}
	data, err := json.MarshalIndent(s.snapshot, "", "  ")
	if err != nil {
		return err
	}
	return AtomicWriteFile(s.path, data, 0o644)
}

func (s *EnterpriseStore) loadLocked() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if !os.IsNotExist(err) {
			s.snapshot = enterpriseSnapshot{Version: 1}
		}
		return
	}
	var snap enterpriseSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		s.snapshot = enterpriseSnapshot{Version: 1}
		return
	}
	if snap.Version == 0 {
		snap.Version = 1
	}
	s.snapshot = snap
}

func ensureEnterpriseID(prefix, current string) string {
	current = strings.TrimSpace(current)
	if current != "" {
		return current
	}
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

func enterpriseSlug(input, fallback string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	var b strings.Builder
	lastHyphen := false
	for _, r := range input {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastHyphen = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastHyphen = false
		default:
			if !lastHyphen && b.Len() > 0 {
				b.WriteByte('-')
				lastHyphen = true
			}
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug != "" {
		return slug
	}
	return strings.ReplaceAll(fallback, "_", "-")
}
