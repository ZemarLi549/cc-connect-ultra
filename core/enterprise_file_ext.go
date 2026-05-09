package core

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

func (s *EnterpriseStore) ListPermissions() []EnterprisePermission {
	return BuiltinEnterprisePermissions()
}

func (s *EnterpriseStore) ListRoles(tenantID, scope string) []EnterpriseRole {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []EnterpriseRole
	for _, item := range s.snapshot.Roles {
		if tenantID != "" && item.TenantID != tenantID {
			continue
		}
		if scope != "" && item.Scope != scope {
			continue
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *EnterpriseStore) UpsertRole(item EnterpriseRole) (EnterpriseRole, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	item.ID = ensureStableEnterpriseID("role", item.ID, item.Name)
	item.Name = strings.TrimSpace(item.Name)
	if item.Name == "" {
		return EnterpriseRole{}, fmt.Errorf("role name is required")
	}
	if item.Slug == "" {
		item.Slug = enterpriseSlug(item.Name, item.ID)
	}
	if item.Scope == "" {
		item.Scope = "tenant"
	}
	if item.Status == "" {
		item.Status = "active"
	}
	item.PermissionIDs = normalizePermissionIDs(item.PermissionIDs)
	item.UpdatedAt = now
	found := false
	for i := range s.snapshot.Roles {
		if s.snapshot.Roles[i].ID != item.ID {
			continue
		}
		found = true
		item.CreatedAt = s.snapshot.Roles[i].CreatedAt
		if item.CreatedAt.IsZero() {
			item.CreatedAt = now
		}
		s.snapshot.Roles[i] = item
		break
	}
	if !found {
		item.CreatedAt = now
		s.snapshot.Roles = append(s.snapshot.Roles, item)
	}
	return item, s.saveLocked()
}

func (s *EnterpriseStore) ListRoleBindings(filter EnterpriseRoleBindingFilter) []EnterpriseRoleBinding {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []EnterpriseRoleBinding
	for _, item := range s.snapshot.RoleBindings {
		if filter.TenantID != "" && item.TenantID != filter.TenantID {
			continue
		}
		if filter.UserID != "" && item.UserID != filter.UserID {
			continue
		}
		if filter.SpaceID != "" && item.SpaceID != filter.SpaceID {
			continue
		}
		if filter.ProjectID != "" && item.ProjectID != filter.ProjectID {
			continue
		}
		if filter.Scope != "" && item.Scope != filter.Scope {
			continue
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

func (s *EnterpriseStore) UpsertRoleBinding(item EnterpriseRoleBinding) (EnterpriseRoleBinding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	item.ID = ensureStableEnterpriseID("binding", item.ID, item.RoleID+"-"+item.UserID+"-"+item.SpaceID+"-"+item.ProjectID)
	if strings.TrimSpace(item.RoleID) == "" {
		return EnterpriseRoleBinding{}, fmt.Errorf("role_id is required")
	}
	if strings.TrimSpace(item.UserID) == "" {
		return EnterpriseRoleBinding{}, fmt.Errorf("user_id is required")
	}
	if item.Scope == "" {
		item.Scope = "tenant"
	}
	if item.Status == "" {
		item.Status = "active"
	}
	item.UpdatedAt = now
	found := false
	for i := range s.snapshot.RoleBindings {
		if s.snapshot.RoleBindings[i].ID != item.ID {
			continue
		}
		found = true
		item.CreatedAt = s.snapshot.RoleBindings[i].CreatedAt
		if item.CreatedAt.IsZero() {
			item.CreatedAt = now
		}
		s.snapshot.RoleBindings[i] = item
		break
	}
	if !found {
		item.CreatedAt = now
		s.snapshot.RoleBindings = append(s.snapshot.RoleBindings, item)
	}
	return item, s.saveLocked()
}

func (s *EnterpriseStore) ResolveAccess(req EnterpriseAccessRequest) (EnterpriseAccessProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var roles []EnterpriseRole
	for _, role := range s.snapshot.Roles {
		if req.TenantID != "" && role.TenantID != "" && role.TenantID != req.TenantID {
			continue
		}
		roles = append(roles, role)
	}
	var bindings []EnterpriseRoleBinding
	for _, binding := range s.snapshot.RoleBindings {
		if binding.UserID != req.UserID {
			continue
		}
		if req.TenantID != "" && binding.TenantID != "" && binding.TenantID != req.TenantID {
			continue
		}
		bindings = append(bindings, binding)
	}
	return resolveEnterpriseAccessFromData(req, roles, bindings), nil
}

func (s *EnterpriseStore) ListProjects(tenantID, spaceID string) []EnterpriseProjectProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []EnterpriseProjectProfile
	for _, item := range s.snapshot.Projects {
		if tenantID != "" && item.TenantID != tenantID {
			continue
		}
		if spaceID != "" && item.SpaceID != spaceID {
			continue
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *EnterpriseStore) UpsertProject(item EnterpriseProjectProfile) (EnterpriseProjectProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	item.ID = ensureStableEnterpriseID("project", item.ID, item.Name)
	item.Name = strings.TrimSpace(item.Name)
	if item.Name == "" {
		return EnterpriseProjectProfile{}, fmt.Errorf("project name is required")
	}
	if item.Slug == "" {
		item.Slug = enterpriseSlug(item.Name, item.ID)
	}
	if item.Status == "" {
		item.Status = "active"
	}
	if item.Source == "" {
		item.Source = "ui"
	}
	item.UpdatedAt = now
	found := false
	for i := range s.snapshot.Projects {
		if s.snapshot.Projects[i].ID != item.ID {
			continue
		}
		found = true
		item.CreatedAt = s.snapshot.Projects[i].CreatedAt
		if item.CreatedAt.IsZero() {
			item.CreatedAt = now
		}
		s.snapshot.Projects[i] = item
		break
	}
	if !found {
		item.CreatedAt = now
		s.snapshot.Projects = append(s.snapshot.Projects, item)
	}
	return item, s.saveLocked()
}

func (s *EnterpriseStore) SyncProjects(items []EnterpriseProjectProfile) error {
	for _, item := range items {
		item.Source = "config"
		if _, err := s.UpsertProject(item); err != nil {
			return err
		}
	}
	return nil
}

func (s *EnterpriseStore) Close() error {
	return nil
}

func (s *EnterpriseStore) ListTasks(filter EnterpriseTaskFilter) []EnterpriseTask {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []EnterpriseTask
	for _, item := range s.snapshot.Tasks {
		if filter.TenantID != "" && item.TenantID != filter.TenantID {
			continue
		}
		if filter.SpaceID != "" && item.SpaceID != filter.SpaceID {
			continue
		}
		if filter.OwnerUserID != "" && item.OwnerUserID != filter.OwnerUserID {
			continue
		}
		if filter.AssigneeUserID != "" && item.AssigneeUserID != filter.AssigneeUserID {
			continue
		}
		if filter.TaskType != "" && item.TaskType != filter.TaskType {
			continue
		}
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		if filter.Priority != "" && item.Priority != filter.Priority {
			continue
		}
		if filter.Tag != "" {
			found := false
			for _, tag := range item.Tags {
				if strings.EqualFold(strings.TrimSpace(tag), strings.TrimSpace(filter.Tag)) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if q := strings.TrimSpace(strings.ToLower(filter.Query)); q != "" {
			if !strings.Contains(strings.ToLower(item.Title), q) && !strings.Contains(strings.ToLower(item.Description), q) {
				continue
			}
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].DueAt.IsZero() && out[j].DueAt.IsZero() {
			return out[i].UpdatedAt.After(out[j].UpdatedAt)
		}
		if out[i].DueAt.IsZero() {
			return false
		}
		if out[j].DueAt.IsZero() {
			return true
		}
		return out[i].DueAt.Before(out[j].DueAt)
	})
	return out
}

func (s *EnterpriseStore) UpsertTask(item EnterpriseTask) (EnterpriseTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	item.ID = ensureStableEnterpriseID("task", item.ID, item.Title)
	item.Title = strings.TrimSpace(item.Title)
	if item.Title == "" {
		return EnterpriseTask{}, fmt.Errorf("task title is required")
	}
	if item.TaskType == "" {
		item.TaskType = "task"
	}
	if item.Priority == "" {
		item.Priority = "medium"
	}
	if item.Status == "" {
		item.Status = "todo"
	}
	item.Tags = normalizeTagList(item.Tags)
	item.UpdatedAt = now
	if item.Status == "done" && item.CompletedAt.IsZero() {
		item.CompletedAt = now
	}
	found := false
	for i := range s.snapshot.Tasks {
		if s.snapshot.Tasks[i].ID != item.ID {
			continue
		}
		found = true
		item.CreatedAt = s.snapshot.Tasks[i].CreatedAt
		if item.CreatedAt.IsZero() {
			item.CreatedAt = now
		}
		s.snapshot.Tasks[i] = item
		break
	}
	if !found {
		item.CreatedAt = now
		s.snapshot.Tasks = append(s.snapshot.Tasks, item)
	}
	return item, s.saveLocked()
}

func normalizePermissionIDs(in []string) []string {
	seen := make(map[string]bool)
	valid := builtinPermissionMap()
	out := make([]string, 0, len(in))
	for _, item := range in {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		if _, ok := valid[item]; !ok {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func normalizeTagList(in []string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(in))
	for _, item := range in {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		key := strings.ToLower(item)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}
