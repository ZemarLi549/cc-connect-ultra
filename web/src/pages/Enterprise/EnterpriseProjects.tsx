import { useCallback, useEffect, useMemo, useState } from 'react';
import { AlertCircle, Database, FolderGit2, Plus, RefreshCcw } from 'lucide-react';
import { Badge, Button, Input, Textarea } from '@/components/ui';
import {
  createEnterpriseProject,
  listEnterpriseProjects,
  listEnterpriseSpaces,
  listEnterpriseTenants,
  listEnterpriseUsers,
  type EnterpriseProjectProfile,
  type EnterpriseSpace,
  type EnterpriseTenant,
  type EnterpriseUser,
} from '@/api/enterprise';
import { DataPill, EnterpriseHero, EnterprisePanel, Select, TinyTable } from './shared';
import { formatTime, joinCSV, parseCSV, parseJSONArray, parseJSONObject, safeJSONStringify } from './utils';

const defaultProjectForm: Partial<EnterpriseProjectProfile> = {
  tenant_id: '',
  space_id: '',
  owner_user_id: '',
  name: '',
  source: 'ui',
  workspace_dir: '',
  base_dir: '',
  mode: 'default',
  agent_type: 'claudecode',
  provider_refs: [],
  status: 'active',
};

export default function EnterpriseProjects() {
  const [tenants, setTenants] = useState<EnterpriseTenant[]>([]);
  const [users, setUsers] = useState<EnterpriseUser[]>([]);
  const [spaces, setSpaces] = useState<EnterpriseSpace[]>([]);
  const [projects, setProjects] = useState<EnterpriseProjectProfile[]>([]);
  const [tenantFilter, setTenantFilter] = useState('');
  const [spaceFilter, setSpaceFilter] = useState('');
  const [form, setForm] = useState(defaultProjectForm);
  const [providerRefsInput, setProviderRefsInput] = useState('');
  const [agentOptionsInput, setAgentOptionsInput] = useState('{\n  "work_dir": "",\n  "mode": "default"\n}');
  const [providersInput, setProvidersInput] = useState('[]');
  const [platformsInput, setPlatformsInput] = useState('[]');
  const [metadataInput, setMetadataInput] = useState('{}');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const fetchData = useCallback(async () => {
    try {
      setLoading(true);
      setError('');
      const [tenantData, userData, spaceData, projectData] = await Promise.all([
        listEnterpriseTenants(),
        listEnterpriseUsers(tenantFilter ? { tenant_id: tenantFilter } : undefined),
        listEnterpriseSpaces(tenantFilter ? { tenant_id: tenantFilter } : undefined),
        listEnterpriseProjects({
          tenant_id: tenantFilter || undefined,
          space_id: spaceFilter || undefined,
        }),
      ]);
      setTenants(tenantData.tenants || []);
      setUsers(userData.users || []);
      setSpaces(spaceData.spaces || []);
      setProjects(projectData.projects || []);
    } catch (e: any) {
      setError(e.message || '加载项目档案失败');
    } finally {
      setLoading(false);
    }
  }, [spaceFilter, tenantFilter]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const sourceStats = useMemo(() => {
    return {
      config: projects.filter((item) => item.source === 'config').length,
      ui: projects.filter((item) => item.source === 'ui').length,
      active: projects.filter((item) => (item.status || 'active') === 'active').length,
    };
  }, [projects]);

  async function handleCreateProject() {
    if (!form.name) {
      setError('项目名称不能为空');
      return;
    }
    try {
      const payload: Partial<EnterpriseProjectProfile> = {
        ...form,
        provider_refs: parseCSV(providerRefsInput),
        agent_options: parseJSONObject(agentOptionsInput, 'Agent 选项'),
        providers: parseJSONArray(providersInput, '项目 Providers'),
        platforms: parseJSONArray(platformsInput, '项目 Platforms'),
        metadata: parseJSONObject(metadataInput, '项目元数据'),
      };
      setSaving(true);
      setError('');
      await createEnterpriseProject(payload);
      setForm(defaultProjectForm);
      setProviderRefsInput('');
      setAgentOptionsInput('{\n  "work_dir": "",\n  "mode": "default"\n}');
      setProvidersInput('[]');
      setPlatformsInput('[]');
      setMetadataInput('{}');
      await fetchData();
    } catch (e: any) {
      setError(e.message || '创建项目档案失败');
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="space-y-6 animate-fade-in">
      <EnterpriseHero
        eyebrow="项目档案"
        title="统一存储 Space、Agent 与 Projects 配置"
        description="项目档案用于保存工作目录、Agent 选项、Provider 引用、平台接入和来源信息，适合作为企业统一控制面的运行档案层。"
        actions={(
          <Button variant="secondary" onClick={fetchData} loading={loading}>
            <RefreshCcw size={15} /> 刷新
          </Button>
        )}
      />

      {error && (
        <div className="flex items-center gap-2 rounded-2xl border border-red-500/20 bg-red-500/10 px-4 py-3 text-sm text-red-600 dark:text-red-300">
          <AlertCircle size={16} /> {error}
        </div>
      )}

      <div className="grid gap-3 md:grid-cols-4">
        <DataPill label="项目档案" value={projects.length} />
        <DataPill label="活跃项目" value={sourceStats.active} />
        <DataPill label="配置同步来源" value={sourceStats.config} />
        <DataPill label="控制面创建" value={sourceStats.ui} />
      </div>

      <div className="grid gap-6 xl:grid-cols-[1fr_1fr]">
        <EnterprisePanel
          title="创建项目档案"
          description="这里可以把工作目录、Agent 参数、Provider 配置和平台清单统一存入控制面。"
          action={(
            <div className="inline-flex items-center gap-2 rounded-full border border-emerald-500/15 bg-emerald-500/10 px-3 py-1 text-xs font-medium text-emerald-700 dark:text-emerald-300">
              <Database size={13} /> PG 控制面存储
            </div>
          )}
        >
          <div className="grid gap-4 md:grid-cols-2">
            <Input
              label="项目名称"
              value={form.name || ''}
              onChange={(e) => setForm((prev) => ({ ...prev, name: e.target.value }))}
              placeholder="ops-runtime"
            />
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">来源</label>
              <Select value={form.source || 'ui'} onChange={(e) => setForm((prev) => ({ ...prev, source: e.target.value }))}>
                <option value="ui">ui</option>
                <option value="config">config</option>
                <option value="sync">sync</option>
                <option value="import">import</option>
              </Select>
            </div>
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">所属租户</label>
              <Select value={form.tenant_id || ''} onChange={(e) => setForm((prev) => ({ ...prev, tenant_id: e.target.value }))}>
                <option value="">不绑定租户</option>
                {tenants.map((tenant) => (
                  <option key={tenant.id} value={tenant.id}>{tenant.name}</option>
                ))}
              </Select>
            </div>
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">所属空间</label>
              <Select value={form.space_id || ''} onChange={(e) => setForm((prev) => ({ ...prev, space_id: e.target.value }))}>
                <option value="">不绑定空间</option>
                {spaces
                  .filter((space) => !form.tenant_id || space.tenant_id === form.tenant_id)
                  .map((space) => (
                    <option key={space.id} value={space.id}>{space.name}</option>
                  ))}
              </Select>
            </div>
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">归属用户</label>
              <Select value={form.owner_user_id || ''} onChange={(e) => setForm((prev) => ({ ...prev, owner_user_id: e.target.value }))}>
                <option value="">不指定用户</option>
                {users
                  .filter((user) => !form.tenant_id || user.tenant_id === form.tenant_id)
                  .map((user) => (
                    <option key={user.id} value={user.id}>{user.display_name}</option>
                  ))}
              </Select>
            </div>
            <Input
              label="工作目录"
              value={form.workspace_dir || ''}
              onChange={(e) => setForm((prev) => ({ ...prev, workspace_dir: e.target.value }))}
              placeholder="D:\\ops\\runtime"
            />
            <Input
              label="基础目录"
              value={form.base_dir || ''}
              onChange={(e) => setForm((prev) => ({ ...prev, base_dir: e.target.value }))}
              placeholder="D:\\lzx_projects\\ai-analyze"
            />
            <Input
              label="Agent 类型"
              value={form.agent_type || ''}
              onChange={(e) => setForm((prev) => ({ ...prev, agent_type: e.target.value }))}
              placeholder="claudecode"
            />
            <Input
              label="运行模式"
              value={form.mode || ''}
              onChange={(e) => setForm((prev) => ({ ...prev, mode: e.target.value }))}
              placeholder="default"
            />
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">状态</label>
              <Select value={form.status || 'active'} onChange={(e) => setForm((prev) => ({ ...prev, status: e.target.value }))}>
                <option value="active">active</option>
                <option value="draft">draft</option>
                <option value="disabled">disabled</option>
              </Select>
            </div>
          </div>

          <div className="mt-4">
            <Input
              label="Provider 引用列表"
              value={providerRefsInput}
              onChange={(e) => setProviderRefsInput(e.target.value)}
              placeholder="deepseek, glm, openai-enterprise"
            />
          </div>

          <div className="mt-4 grid gap-4">
            <Textarea label="Agent 选项 JSON" value={agentOptionsInput} onChange={(e) => setAgentOptionsInput(e.target.value)} rows={6} />
            <Textarea label="项目 Providers JSON 数组" value={providersInput} onChange={(e) => setProvidersInput(e.target.value)} rows={6} />
            <Textarea label="项目 Platforms JSON 数组" value={platformsInput} onChange={(e) => setPlatformsInput(e.target.value)} rows={6} />
            <Textarea label="项目元数据 JSON" value={metadataInput} onChange={(e) => setMetadataInput(e.target.value)} rows={4} />
          </div>

          <div className="mt-4 flex items-center justify-between gap-3">
            <p className="text-xs text-slate-500 dark:text-slate-400">
              如果你要把 Space 目录、Agent 配置、Projects 全部统一进 PG，这个档案层就是最合适的承载位置。
            </p>
            <Button onClick={handleCreateProject} loading={saving}>
              <Plus size={15} /> 保存项目档案
            </Button>
          </div>
        </EnterprisePanel>

        <EnterprisePanel title="项目档案清单" description="查看当前所有项目档案，以及它们来自配置同步还是控制面创建。">
          <div className="mb-4 grid gap-3 md:grid-cols-2">
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">租户筛选</label>
              <Select value={tenantFilter} onChange={(e) => setTenantFilter(e.target.value)}>
                <option value="">全部租户</option>
                {tenants.map((tenant) => (
                  <option key={tenant.id} value={tenant.id}>{tenant.name}</option>
                ))}
              </Select>
            </div>
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">空间筛选</label>
              <Select value={spaceFilter} onChange={(e) => setSpaceFilter(e.target.value)}>
                <option value="">全部空间</option>
                {spaces
                  .filter((space) => !tenantFilter || space.tenant_id === tenantFilter)
                  .map((space) => (
                    <option key={space.id} value={space.id}>{space.name}</option>
                  ))}
              </Select>
            </div>
          </div>

          <TinyTable>
            <thead className="bg-slate-50/90 text-left text-xs uppercase tracking-[0.18em] text-slate-400 dark:bg-white/[0.03]">
              <tr>
                <th className="px-4 py-3">项目</th>
                <th className="px-4 py-3">空间 / 租户</th>
                <th className="px-4 py-3">Agent</th>
                <th className="px-4 py-3">Provider 引用</th>
                <th className="px-4 py-3">来源 / 状态</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-200/80 dark:divide-white/[0.06]">
              {projects.length === 0 ? (
                <tr>
                  <td colSpan={5} className="px-4 py-10 text-center text-sm text-slate-500 dark:text-slate-400">
                    还没有项目档案。
                  </td>
                </tr>
              ) : projects.map((project) => {
                const tenant = tenants.find((item) => item.id === project.tenant_id);
                const space = spaces.find((item) => item.id === project.space_id);
                return (
                  <tr key={project.id}>
                    <td className="px-4 py-3">
                      <div className="flex items-start gap-3">
                        <div className="mt-0.5 rounded-xl bg-emerald-500/10 p-2 text-emerald-600 dark:text-emerald-300">
                          <FolderGit2 size={14} />
                        </div>
                        <div>
                          <p className="text-sm font-medium text-slate-900 dark:text-white">{project.name}</p>
                          <p className="text-xs text-slate-400">{project.workspace_dir || project.base_dir || '-'}</p>
                        </div>
                      </div>
                    </td>
                    <td className="px-4 py-3 text-sm text-slate-500 dark:text-slate-400">
                      {space?.name || project.space_id || '-'}
                      <div className="text-xs text-slate-400">{tenant?.name || project.tenant_id || '全局'}</div>
                    </td>
                    <td className="px-4 py-3 text-sm text-slate-500 dark:text-slate-400">
                      {project.agent_type || '-'}
                      <div className="text-xs text-slate-400">{project.mode || 'default'}</div>
                    </td>
                    <td className="px-4 py-3 text-sm text-slate-500 dark:text-slate-400">{joinCSV(project.provider_refs) || '-'}</td>
                    <td className="px-4 py-3 text-sm">
                      <div className="flex flex-wrap gap-2">
                        <Badge variant="outline">{project.source || 'ui'}</Badge>
                        <Badge variant={(project.status || 'active') === 'active' ? 'success' : 'outline'}>{project.status || 'active'}</Badge>
                      </div>
                      <div className="mt-1 text-xs text-slate-400">{formatTime(project.updated_at)}</div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </TinyTable>

          <div className="mt-4 grid gap-3">
            {projects.slice(0, 2).map((project) => (
              <div key={`${project.id}-json`} className="rounded-2xl border border-slate-200/80 bg-slate-50/80 p-4 dark:border-white/[0.06] dark:bg-white/[0.03]">
                <p className="text-xs uppercase tracking-[0.18em] text-slate-400">{project.name} 配置快照</p>
                <pre className="mt-3 overflow-x-auto text-xs leading-6 text-slate-600 dark:text-slate-300">
{safeJSONStringify({
  agent_options: project.agent_options,
  providers: project.providers,
  platforms: project.platforms,
  metadata: project.metadata,
}, '{}')}
                </pre>
              </div>
            ))}
          </div>
        </EnterprisePanel>
      </div>
    </div>
  );
}
