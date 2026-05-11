import api from './client';

export interface ProjectSummary {
  name: string;
  agent_type: string;
  platforms: string[];
  sessions_count: number;
  heartbeat_enabled: boolean;
}

export interface PlatformConfigInfo {
  index?: number;
  type: string;
  allow_from?: string;
  options?: Record<string, any>;
}

export interface ProjectDetail {
  name: string;
  agent_type: string;
  work_dir?: string;
  agent_mode?: string;
  show_context_indicator?: boolean;
  reply_footer?: boolean;
  inject_sender?: boolean;
  provider_refs?: string[];
  agent_options?: Record<string, any>;
  platform_configs?: PlatformConfigInfo[];
  platforms: { type: string; connected: boolean }[];
  sessions_count: number;
  active_session_keys: string[];
  heartbeat: {
    enabled: boolean;
    paused: boolean;
    interval_mins: number;
    session_key: string;
  };
  settings: {
    admin_from: string;
    language: string;
    disabled_commands: string[];
  };
}

export interface ProjectSettingsUpdate {
  language?: string;
  admin_from?: string;
  disabled_commands?: string[];
  work_dir?: string;
  mode?: string;
  agent_type?: string;
  show_context_indicator?: boolean;
  reply_footer?: boolean;
  inject_sender?: boolean;
  platform_allow_from?: Record<string, string>;
  agent_options?: Record<string, any>;
  platform_option_updates?: Array<{ index: number; options: Record<string, any> }>;
  remove_platform_indexes?: number[];
}

export const listAgentTypes = () => api.get<{ agents: string[]; platforms: string[] }>('/agents');

export const listProjects = () => api.get<{ projects: ProjectSummary[] }>('/projects');
export const getProject = (name: string) => api.get<ProjectDetail>(`/projects/${name}`);
export const updateProject = (name: string, body: ProjectSettingsUpdate) => api.patch(`/projects/${name}`, body);

export const addPlatformToProject = (projectName: string, body: {
  type: string; options: Record<string, any>; work_dir?: string; agent_type?: string; agent_options?: Record<string, any>;
}) => api.post<{ message: string; restart_required: boolean }>(`/projects/${projectName}/add-platform`, body);

export const deleteProject = (name: string) =>
  api.delete<{ message: string; restart_required: boolean }>(`/projects/${name}`);

export interface FeishuLookupResult {
  project: string;
  query: string;
  users: Array<{ open_id: string; name?: string; en_name?: string; nickname?: string }>;
  chats: Array<{ chat_id: string; name?: string }>;
  warnings?: string[];
}

export const lookupFeishuIds = (projectName: string, query: string, limit = 20) =>
  api.get<FeishuLookupResult>(`/projects/${projectName}/feishu-lookup?q=${encodeURIComponent(query)}&limit=${limit}`);
