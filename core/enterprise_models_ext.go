package core

import (
	"sort"
	"strings"
	"time"
)

// EnterprisePermission is a built-in RBAC permission exposed by the control plane.
type EnterprisePermission struct {
	ID          string `json:"id"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
	Group       string `json:"group,omitempty"`
	Description string `json:"description,omitempty"`
	BuiltIn     bool   `json:"built_in"`
}

// EnterpriseRole is a reusable RBAC role that bundles permissions.
type EnterpriseRole struct {
	ID            string            `json:"id"`
	TenantID      string            `json:"tenant_id,omitempty"`
	Name          string            `json:"name"`
	Slug          string            `json:"slug"`
	Description   string            `json:"description,omitempty"`
	Scope         string            `json:"scope,omitempty"`  // global, tenant, space, project
	Status        string            `json:"status,omitempty"` // active, draft, disabled
	PermissionIDs []string          `json:"permission_ids,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

// EnterpriseRoleBinding assigns a role to a user in a specific scope.
type EnterpriseRoleBinding struct {
	ID        string            `json:"id"`
	TenantID  string            `json:"tenant_id,omitempty"`
	RoleID    string            `json:"role_id"`
	UserID    string            `json:"user_id"`
	SpaceID   string            `json:"space_id,omitempty"`
	ProjectID string            `json:"project_id,omitempty"`
	Scope     string            `json:"scope,omitempty"`  // global, tenant, space, project
	Status    string            `json:"status,omitempty"` // active, disabled
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// EnterpriseRoleBindingFilter filters role binding queries.
type EnterpriseRoleBindingFilter struct {
	TenantID  string
	UserID    string
	SpaceID   string
	ProjectID string
	Scope     string
}

// EnterpriseProjectPlatformConfig stores runtime platform wiring for a project profile.
type EnterpriseProjectPlatformConfig struct {
	Type    string         `json:"type"`
	Options map[string]any `json:"options,omitempty"`
}

// EnterpriseProjectProviderConfig stores project-level provider wiring.
type EnterpriseProjectProviderConfig struct {
	Name        string            `json:"name"`
	BaseURL     string            `json:"base_url,omitempty"`
	Model       string            `json:"model,omitempty"`
	Thinking    string            `json:"thinking,omitempty"`
	AgentTypes  []string          `json:"agent_types,omitempty"`
	Endpoints   map[string]string `json:"endpoints,omitempty"`
	AgentModels map[string]string `json:"agent_models,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// EnterpriseProjectProfile stores a unified project/agent runtime profile in the control plane.
type EnterpriseProjectProfile struct {
	ID           string                            `json:"id"`
	TenantID     string                            `json:"tenant_id,omitempty"`
	SpaceID      string                            `json:"space_id,omitempty"`
	OwnerUserID  string                            `json:"owner_user_id,omitempty"`
	Name         string                            `json:"name"`
	Slug         string                            `json:"slug"`
	Source       string                            `json:"source,omitempty"` // config, ui, sync
	WorkspaceDir string                            `json:"workspace_dir,omitempty"`
	BaseDir      string                            `json:"base_dir,omitempty"`
	Mode         string                            `json:"mode,omitempty"`
	AgentType    string                            `json:"agent_type,omitempty"`
	AgentOptions map[string]any                    `json:"agent_options,omitempty"`
	ProviderRefs []string                          `json:"provider_refs,omitempty"`
	Providers    []EnterpriseProjectProviderConfig `json:"providers,omitempty"`
	Platforms    []EnterpriseProjectPlatformConfig `json:"platforms,omitempty"`
	Status       string                            `json:"status,omitempty"`
	Metadata     map[string]string                 `json:"metadata,omitempty"`
	CreatedAt    time.Time                         `json:"created_at"`
	UpdatedAt    time.Time                         `json:"updated_at"`
}

// EnterpriseAccessRequest describes the scope where effective access should be resolved.
type EnterpriseAccessRequest struct {
	TenantID  string `json:"tenant_id,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	SpaceID   string `json:"space_id,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
}

// EnterpriseAccessProfile is the resolved RBAC view for a user.
type EnterpriseAccessProfile struct {
	TenantID      string                 `json:"tenant_id,omitempty"`
	UserID        string                 `json:"user_id,omitempty"`
	SpaceID       string                 `json:"space_id,omitempty"`
	ProjectID     string                 `json:"project_id,omitempty"`
	RoleIDs       []string               `json:"role_ids,omitempty"`
	PermissionIDs []string               `json:"permission_ids,omitempty"`
	Roles         []EnterpriseRole       `json:"roles,omitempty"`
	Permissions   []EnterprisePermission `json:"permissions,omitempty"`
	ResolvedAt    time.Time              `json:"resolved_at"`
}

// EnterpriseStoreOptions drives enterprise control-plane storage backend selection.
type EnterpriseStoreOptions struct {
	FilePath     string
	Postgres     EnterpriseSQLConfig
	Redis        EnterpriseRedisConfig
	SeedSettings EnterpriseAIOpsSettings
}

// EnterpriseDataStore is the common backend abstraction for enterprise control-plane data.
type EnterpriseDataStore interface {
	ListTenants() []EnterpriseTenant
	UpsertTenant(EnterpriseTenant) (EnterpriseTenant, error)
	ListUsers(tenantID string) []EnterpriseUser
	UpsertUser(EnterpriseUser) (EnterpriseUser, error)
	ListSpaces(tenantID, ownerUserID string) []EnterpriseSpace
	UpsertSpace(EnterpriseSpace) (EnterpriseSpace, error)
	ListSkills(scope, tenantID, ownerUserID string) []EnterpriseSkill
	UpsertSkill(EnterpriseSkill) (EnterpriseSkill, error)
	ListBots(tenantID, ownerUserID string) []EnterpriseBot
	UpsertBot(EnterpriseBot) (EnterpriseBot, error)
	ListProviders() []EnterpriseProvider
	UpsertProvider(EnterpriseProvider) (EnterpriseProvider, error)
	GetSettings() EnterpriseAIOpsSettings
	SaveSettings(EnterpriseAIOpsSettings) (EnterpriseAIOpsSettings, error)
	ListImports(tenantID string) []EnterpriseSkillImportRequest
	AddImport(EnterpriseSkillImportRequest) (EnterpriseSkillImportRequest, error)
	ListUsage(EnterpriseUsageFilter) []EnterpriseUsageRecord
	AddUsage(EnterpriseUsageRecord) (EnterpriseUsageRecord, error)
	Overview() EnterpriseOverview
	Leaderboard(groupBy string, limit int) []EnterpriseLeaderboardEntry
	ListPermissions() []EnterprisePermission
	ListRoles(tenantID, scope string) []EnterpriseRole
	UpsertRole(EnterpriseRole) (EnterpriseRole, error)
	ListRoleBindings(EnterpriseRoleBindingFilter) []EnterpriseRoleBinding
	UpsertRoleBinding(EnterpriseRoleBinding) (EnterpriseRoleBinding, error)
	ResolveAccess(EnterpriseAccessRequest) (EnterpriseAccessProfile, error)
	ListProjects(tenantID, spaceID string) []EnterpriseProjectProfile
	UpsertProject(EnterpriseProjectProfile) (EnterpriseProjectProfile, error)
	SyncProjects([]EnterpriseProjectProfile) error
	ListTasks(EnterpriseTaskFilter) []EnterpriseTask
	UpsertTask(EnterpriseTask) (EnterpriseTask, error)
	Close() error
}

// BuiltinEnterprisePermissions returns the default permission catalog for enterprise RBAC.
func BuiltinEnterprisePermissions() []EnterprisePermission {
	permissions := []EnterprisePermission{
		{ID: "tenant:view", Resource: "tenant", Action: "view", Group: "tenant", Description: "View tenant information", BuiltIn: true},
		{ID: "tenant:manage", Resource: "tenant", Action: "manage", Group: "tenant", Description: "Manage tenants", BuiltIn: true},
		{ID: "user:view", Resource: "user", Action: "view", Group: "user", Description: "View users", BuiltIn: true},
		{ID: "user:manage", Resource: "user", Action: "manage", Group: "user", Description: "Manage users and budgets", BuiltIn: true},
		{ID: "space:view", Resource: "space", Action: "view", Group: "workspace", Description: "View user spaces", BuiltIn: true},
		{ID: "space:manage", Resource: "space", Action: "manage", Group: "workspace", Description: "Create or update spaces", BuiltIn: true},
		{ID: "space:execute", Resource: "space", Action: "execute", Group: "workspace", Description: "Execute workloads in a space", BuiltIn: true},
		{ID: "project:view", Resource: "project", Action: "view", Group: "runtime", Description: "View runtime project profiles", BuiltIn: true},
		{ID: "project:manage", Resource: "project", Action: "manage", Group: "runtime", Description: "Manage runtime project profiles", BuiltIn: true},
		{ID: "project:agent_config", Resource: "project", Action: "agent_config", Group: "runtime", Description: "Change project agent configuration", BuiltIn: true},
		{ID: "skill:view", Resource: "skill", Action: "view", Group: "skill", Description: "View skills", BuiltIn: true},
		{ID: "skill:manage", Resource: "skill", Action: "manage", Group: "skill", Description: "Create or update skills", BuiltIn: true},
		{ID: "skill:publish", Resource: "skill", Action: "publish", Group: "skill", Description: "Publish tenant/public skills", BuiltIn: true},
		{ID: "skill:import", Resource: "skill", Action: "import", Group: "skill", Description: "Import Cocoloop or external skills", BuiltIn: true},
		{ID: "bot:view", Resource: "bot", Action: "view", Group: "bot", Description: "View enterprise bots", BuiltIn: true},
		{ID: "bot:manage", Resource: "bot", Action: "manage", Group: "bot", Description: "Create or update enterprise bots", BuiltIn: true},
		{ID: "provider:view", Resource: "provider", Action: "view", Group: "provider", Description: "View model providers", BuiltIn: true},
		{ID: "provider:manage", Resource: "provider", Action: "manage", Group: "provider", Description: "Manage model providers", BuiltIn: true},
		{ID: "analytics:view", Resource: "analytics", Action: "view", Group: "analytics", Description: "View analytics and rankings", BuiltIn: true},
		{ID: "usage:view", Resource: "usage", Action: "view", Group: "analytics", Description: "View detailed usage records", BuiltIn: true},
		{ID: "task:view", Resource: "task", Action: "view", Group: "task", Description: "View enterprise tasks", BuiltIn: true},
		{ID: "task:manage", Resource: "task", Action: "manage", Group: "task", Description: "Create or update enterprise tasks", BuiltIn: true},
		{ID: "role:view", Resource: "role", Action: "view", Group: "rbac", Description: "View roles and bindings", BuiltIn: true},
		{ID: "role:manage", Resource: "role", Action: "manage", Group: "rbac", Description: "Manage roles and role bindings", BuiltIn: true},
		{ID: "settings:view", Resource: "settings", Action: "view", Group: "settings", Description: "View enterprise settings", BuiltIn: true},
		{ID: "settings:manage", Resource: "settings", Action: "manage", Group: "settings", Description: "Manage enterprise settings", BuiltIn: true},
	}
	sort.Slice(permissions, func(i, j int) bool { return permissions[i].ID < permissions[j].ID })
	return permissions
}

func builtinPermissionMap() map[string]EnterprisePermission {
	out := make(map[string]EnterprisePermission)
	for _, item := range BuiltinEnterprisePermissions() {
		out[item.ID] = item
	}
	return out
}

func ensureStableEnterpriseID(prefix, current, seed string) string {
	current = strings.TrimSpace(current)
	if current != "" {
		return current
	}
	slug := enterpriseSlug(seed, "")
	if slug != "" {
		return prefix + "_" + slug
	}
	return ensureEnterpriseID(prefix, current)
}

func scopeMatches(bindingScope, tenantID, spaceID, projectID string, binding EnterpriseRoleBinding) bool {
	switch bindingScope {
	case "global":
		return true
	case "space":
		return binding.SpaceID != "" && binding.SpaceID == spaceID
	case "project":
		return binding.ProjectID != "" && binding.ProjectID == projectID
	default:
		if binding.TenantID == "" {
			return true
		}
		return binding.TenantID == tenantID
	}
}

func resolveEnterpriseAccessFromData(
	req EnterpriseAccessRequest,
	roles []EnterpriseRole,
	bindings []EnterpriseRoleBinding,
) EnterpriseAccessProfile {
	roleByID := make(map[string]EnterpriseRole, len(roles))
	for _, role := range roles {
		roleByID[role.ID] = role
	}
	permissionCatalog := builtinPermissionMap()
	roleSeen := make(map[string]bool)
	permSeen := make(map[string]bool)
	var resolvedRoles []EnterpriseRole
	var permissionIDs []string

	for _, binding := range bindings {
		if binding.UserID != req.UserID {
			continue
		}
		if status := strings.TrimSpace(binding.Status); status != "" && status != "active" {
			continue
		}
		scope := strings.TrimSpace(binding.Scope)
		if scope == "" {
			scope = "tenant"
		}
		if !scopeMatches(scope, req.TenantID, req.SpaceID, req.ProjectID, binding) {
			continue
		}
		role, ok := roleByID[binding.RoleID]
		if !ok {
			continue
		}
		if !roleSeen[role.ID] {
			roleSeen[role.ID] = true
			resolvedRoles = append(resolvedRoles, role)
		}
		for _, permID := range role.PermissionIDs {
			if permSeen[permID] {
				continue
			}
			if _, ok := permissionCatalog[permID]; !ok {
				continue
			}
			permSeen[permID] = true
			permissionIDs = append(permissionIDs, permID)
		}
	}

	sort.Slice(resolvedRoles, func(i, j int) bool { return resolvedRoles[i].Name < resolvedRoles[j].Name })
	sort.Strings(permissionIDs)

	permissions := make([]EnterprisePermission, 0, len(permissionIDs))
	for _, permID := range permissionIDs {
		permissions = append(permissions, permissionCatalog[permID])
	}

	roleIDs := make([]string, 0, len(resolvedRoles))
	for _, role := range resolvedRoles {
		roleIDs = append(roleIDs, role.ID)
	}

	return EnterpriseAccessProfile{
		TenantID:      req.TenantID,
		UserID:        req.UserID,
		SpaceID:       req.SpaceID,
		ProjectID:     req.ProjectID,
		RoleIDs:       roleIDs,
		PermissionIDs: permissionIDs,
		Roles:         resolvedRoles,
		Permissions:   permissions,
		ResolvedAt:    time.Now().UTC(),
	}
}
