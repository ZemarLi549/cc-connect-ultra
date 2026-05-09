import { api } from './client';

export interface EnterpriseOverview {
  tenants_count: number;
  users_count: number;
  spaces_count: number;
  skills_count: number;
  bots_count: number;
  roles_count: number;
  projects_count: number;
  providers_count: number;
  imports_count: number;
  usage_count: number;
  total_tokens: number;
  total_cost_micros: number;
}

export interface EnterpriseTenant {
  id: string;
  name: string;
  slug: string;
  description?: string;
  status?: string;
  metadata?: Record<string, string>;
  created_at?: string;
  updated_at?: string;
}

export interface EnterpriseUser {
  id: string;
  tenant_id: string;
  email?: string;
  display_name: string;
  role?: string;
  status?: string;
  token_budget?: number;
  token_used?: number;
  metadata?: Record<string, string>;
  created_at?: string;
  updated_at?: string;
  last_active_at?: string;
}

export interface EnterpriseSpace {
  id: string;
  tenant_id: string;
  owner_user_id: string;
  name: string;
  slug: string;
  workspace_dir?: string;
  project_name?: string;
  visibility?: string;
  status?: string;
  current_provider?: string;
  current_model?: string;
  shared_skill_ids?: string[];
  metadata?: Record<string, string>;
  created_at?: string;
  updated_at?: string;
  last_interaction_at?: string;
}

export interface EnterpriseSkill {
  id: string;
  tenant_id?: string;
  owner_user_id?: string;
  name: string;
  display_name?: string;
  description?: string;
  scope?: string;
  status?: string;
  version?: string;
  prompt?: string;
  tags?: string[];
  source_path?: string;
  metadata?: Record<string, string>;
  created_at?: string;
  updated_at?: string;
}

export interface EnterpriseBot {
  id: string;
  tenant_id?: string;
  space_id?: string;
  owner_user_id?: string;
  name: string;
  slug: string;
  description?: string;
  scope?: string;
  provider_name?: string;
  model_name?: string;
  skill_ids?: string[];
  status?: string;
  metadata?: Record<string, string>;
  created_at?: string;
  updated_at?: string;
}

export interface EnterpriseProvider {
  id: string;
  name: string;
  display_name?: string;
  provider_type?: string;
  base_url?: string;
  default_model?: string;
  models?: string[];
  status?: string;
  tags?: string[];
  metadata?: Record<string, string>;
  created_at?: string;
  updated_at?: string;
}

export interface EnterprisePermission {
  id: string;
  resource: string;
  action: string;
  group?: string;
  description?: string;
  built_in: boolean;
}

export interface EnterpriseRole {
  id: string;
  tenant_id?: string;
  name: string;
  slug: string;
  description?: string;
  scope?: string;
  status?: string;
  permission_ids?: string[];
  metadata?: Record<string, string>;
  created_at?: string;
  updated_at?: string;
}

export interface EnterpriseRoleBinding {
  id: string;
  tenant_id?: string;
  role_id: string;
  user_id: string;
  space_id?: string;
  project_id?: string;
  scope?: string;
  status?: string;
  metadata?: Record<string, string>;
  created_at?: string;
  updated_at?: string;
}

export interface EnterpriseAccessProfile {
  tenant_id?: string;
  user_id?: string;
  space_id?: string;
  project_id?: string;
  role_ids?: string[];
  permission_ids?: string[];
  roles?: EnterpriseRole[];
  permissions?: EnterprisePermission[];
  resolved_at?: string;
}

export interface EnterpriseProjectPlatformConfig {
  type: string;
  options?: Record<string, any>;
}

export interface EnterpriseProjectProviderConfig {
  name: string;
  base_url?: string;
  model?: string;
  thinking?: string;
  agent_types?: string[];
  endpoints?: Record<string, string>;
  agent_models?: Record<string, string>;
  metadata?: Record<string, string>;
}

export interface EnterpriseProjectProfile {
  id: string;
  tenant_id?: string;
  space_id?: string;
  owner_user_id?: string;
  name: string;
  slug: string;
  source?: string;
  workspace_dir?: string;
  base_dir?: string;
  mode?: string;
  agent_type?: string;
  agent_options?: Record<string, any>;
  provider_refs?: string[];
  providers?: EnterpriseProjectProviderConfig[];
  platforms?: EnterpriseProjectPlatformConfig[];
  status?: string;
  metadata?: Record<string, string>;
  created_at?: string;
  updated_at?: string;
}

export interface EnterpriseTask {
  id: string;
  tenant_id?: string;
  space_id?: string;
  owner_user_id?: string;
  assignee_user_id?: string;
  parent_task_id?: string;
  title: string;
  description?: string;
  task_type?: string;
  priority?: string;
  status?: string;
  tags?: string[];
  due_at?: string;
  reminder_at?: string;
  metadata?: Record<string, string>;
  created_at?: string;
  updated_at?: string;
  completed_at?: string;
}

export interface EnterpriseUsageRecord {
  id: string;
  tenant_id?: string;
  user_id?: string;
  space_id?: string;
  project_name?: string;
  provider_name?: string;
  model_name?: string;
  request_kind?: string;
  prompt_tokens?: number;
  completion_tokens?: number;
  total_tokens?: number;
  cost_micros?: number;
  latency_ms?: number;
  occurred_at: string;
}

export interface EnterpriseLeaderboardEntry {
  subject_type: string;
  subject_id: string;
  subject_name: string;
  requests: number;
  tokens: number;
  cost_micros: number;
}

export interface EnterpriseAIOpsSettings {
  organization_name?: string;
  default_project_name?: string;
  default_space_base_dir?: string;
  postgres?: {
    driver?: string;
    dsn?: string;
    max_open_conns?: number;
    max_idle_conns?: number;
    conn_max_lifetime_secs?: number;
  };
  redis?: {
    addr?: string;
    password?: string;
    db?: number;
    key_prefix?: string;
  };
  cocoloop?: {
    enabled?: boolean;
    base_url?: string;
    api_key?: string;
    workspace?: string;
    last_import_at?: string;
  };
  updated_at?: string;
}

export interface EnterpriseImportJob {
  id: string;
  tenant_id?: string;
  owner_user_id?: string;
  source_type?: string;
  source_name?: string;
  source_ref?: string;
  status?: string;
  imported_skills?: number;
  created_at: string;
  updated_at: string;
  completed_at?: string;
}

export async function getEnterpriseOverview() {
  return api.get<EnterpriseOverview>('/enterprise/overview');
}

export async function listEnterpriseTenants() {
  return api.get<{ tenants: EnterpriseTenant[] }>('/enterprise/tenants');
}

export async function createEnterpriseTenant(payload: Partial<EnterpriseTenant>) {
  return api.post<EnterpriseTenant>('/enterprise/tenants', payload);
}

export async function listEnterpriseUsers(params?: { tenant_id?: string }) {
  return api.get<{ users: EnterpriseUser[] }>('/enterprise/users', params as Record<string, string>);
}

export async function createEnterpriseUser(payload: Partial<EnterpriseUser>) {
  return api.post<EnterpriseUser>('/enterprise/users', payload);
}

export async function listEnterpriseSpaces(params?: { tenant_id?: string; owner_user_id?: string }) {
  return api.get<{ spaces: EnterpriseSpace[] }>('/enterprise/spaces', params as Record<string, string>);
}

export async function createEnterpriseSpace(payload: Partial<EnterpriseSpace>) {
  return api.post<EnterpriseSpace>('/enterprise/spaces', payload);
}

export async function listEnterpriseSkills(params?: { scope?: string; tenant_id?: string; owner_user_id?: string }) {
  return api.get<{ skills: EnterpriseSkill[] }>('/enterprise/skills', params as Record<string, string>);
}

export async function createEnterpriseSkill(payload: Partial<EnterpriseSkill>) {
  return api.post<EnterpriseSkill>('/enterprise/skills', payload);
}

export async function listEnterpriseBots(params?: { tenant_id?: string; owner_user_id?: string }) {
  return api.get<{ bots: EnterpriseBot[] }>('/enterprise/bots', params as Record<string, string>);
}

export async function createEnterpriseBot(payload: Partial<EnterpriseBot>) {
  return api.post<EnterpriseBot>('/enterprise/bots', payload);
}

export async function listEnterpriseProviders() {
  return api.get<{ providers: EnterpriseProvider[] }>('/enterprise/providers');
}

export async function createEnterpriseProvider(payload: Partial<EnterpriseProvider>) {
  return api.post<EnterpriseProvider>('/enterprise/providers', payload);
}

export async function listEnterprisePermissions() {
  return api.get<{ permissions: EnterprisePermission[] }>('/enterprise/permissions');
}

export async function listEnterpriseRoles(params?: { tenant_id?: string; scope?: string }) {
  return api.get<{ roles: EnterpriseRole[] }>('/enterprise/roles', params as Record<string, string>);
}

export async function createEnterpriseRole(payload: Partial<EnterpriseRole>) {
  return api.post<EnterpriseRole>('/enterprise/roles', payload);
}

export async function listEnterpriseRoleBindings(params?: {
  tenant_id?: string;
  user_id?: string;
  space_id?: string;
  project_id?: string;
  scope?: string;
}) {
  return api.get<{ bindings: EnterpriseRoleBinding[] }>('/enterprise/role-bindings', params as Record<string, string>);
}

export async function createEnterpriseRoleBinding(payload: Partial<EnterpriseRoleBinding>) {
  return api.post<EnterpriseRoleBinding>('/enterprise/role-bindings', payload);
}

export async function getEnterpriseEffectiveAccess(params: {
  user_id: string;
  tenant_id?: string;
  space_id?: string;
  project_id?: string;
}) {
  return api.get<EnterpriseAccessProfile>('/enterprise/effective-access', params as Record<string, string>);
}

export async function listEnterpriseProjects(params?: { tenant_id?: string; space_id?: string }) {
  return api.get<{ projects: EnterpriseProjectProfile[] }>('/enterprise/projects', params as Record<string, string>);
}

export async function createEnterpriseProject(payload: Partial<EnterpriseProjectProfile>) {
  return api.post<EnterpriseProjectProfile>('/enterprise/projects', payload);
}

export async function listEnterpriseTasks(params?: {
  tenant_id?: string;
  space_id?: string;
  owner_user_id?: string;
  assignee_user_id?: string;
  task_type?: string;
  status?: string;
  priority?: string;
  tag?: string;
  q?: string;
}) {
  return api.get<{ tasks: EnterpriseTask[] }>('/enterprise/tasks', params as Record<string, string>);
}

export async function createEnterpriseTask(payload: Partial<EnterpriseTask>) {
  return api.post<EnterpriseTask>('/enterprise/tasks', payload);
}

export async function getEnterpriseSettings() {
  return api.get<EnterpriseAIOpsSettings>('/enterprise/settings');
}

export async function saveEnterpriseSettings(payload: EnterpriseAIOpsSettings) {
  return api.post<EnterpriseAIOpsSettings>('/enterprise/settings', payload);
}

export async function listEnterpriseImports(params?: { tenant_id?: string }) {
  return api.get<{ imports: EnterpriseImportJob[] }>('/enterprise/imports', params as Record<string, string>);
}

export async function createEnterpriseImport(payload: Partial<EnterpriseImportJob>) {
  return api.post<EnterpriseImportJob>('/enterprise/imports', payload);
}

export async function listEnterpriseUsage(params?: {
  tenant_id?: string;
  user_id?: string;
  space_id?: string;
  provider?: string;
  limit?: string;
}) {
  return api.get<{ usage: EnterpriseUsageRecord[] }>('/enterprise/usage', params as Record<string, string>);
}

export async function getEnterpriseLeaderboard(params?: { group_by?: string; limit?: string }) {
  return api.get<{ group_by: string; leaderboard: EnterpriseLeaderboardEntry[] }>('/enterprise/leaderboard', params as Record<string, string>);
}
