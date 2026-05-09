import { useCallback, useEffect, useState } from 'react';
import { AlertCircle, FolderKanban, Plus, RefreshCcw, ServerCog, ShieldCheck } from 'lucide-react';
import { Badge, Button, Input } from '@/components/ui';
import {
  createEnterpriseSpace,
  listEnterpriseProviders,
  listEnterpriseSpaces,
  listEnterpriseTenants,
  listEnterpriseUsers,
  type EnterpriseProvider,
  type EnterpriseSpace,
  type EnterpriseTenant,
  type EnterpriseUser,
} from '@/api/enterprise';
import { DataPill, EnterpriseHero, EnterprisePanel, Select, TinyTable } from './shared';
import { formatTime, parseCSV } from './utils';

const defaultForm: Partial<EnterpriseSpace> = {
  tenant_id: '',
  owner_user_id: '',
  name: '',
  workspace_dir: '',
  project_name: 'enterprise-runtime',
  visibility: 'private',
  status: 'active',
  current_provider: '',
  current_model: '',
  shared_skill_ids: [],
};

export default function EnterpriseSpaces() {
  const [tenants, setTenants] = useState<EnterpriseTenant[]>([]);
  const [users, setUsers] = useState<EnterpriseUser[]>([]);
  const [spaces, setSpaces] = useState<EnterpriseSpace[]>([]);
  const [providers, setProviders] = useState<EnterpriseProvider[]>([]);
  const [tenantFilter, setTenantFilter] = useState('');
  const [ownerFilter, setOwnerFilter] = useState('');
  const [form, setForm] = useState(defaultForm);
  const [skillIDs, setSkillIDs] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const fetchData = useCallback(async () => {
    try {
      setLoading(true);
      setError('');
      const [tenantData, userData, spaceData, providerData] = await Promise.all([
        listEnterpriseTenants(),
        listEnterpriseUsers(tenantFilter ? { tenant_id: tenantFilter } : undefined),
        listEnterpriseSpaces({
          tenant_id: tenantFilter || undefined,
          owner_user_id: ownerFilter || undefined,
        }),
        listEnterpriseProviders(),
      ]);
      setTenants(tenantData.tenants || []);
      setUsers(userData.users || []);
      setSpaces(spaceData.spaces || []);
      setProviders(providerData.providers || []);
    } catch (e: any) {
      setError(e.message || '加载用户空间失败');
    } finally {
      setLoading(false);
    }
  }, [ownerFilter, tenantFilter]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  async function handleCreateSpace() {
    if (!form.name || !form.tenant_id || !form.owner_user_id) {
      setError('租户、空间负责人和空间名称不能为空');
      return;
    }
    try {
      setSaving(true);
      setError('');
      await createEnterpriseSpace({
        ...form,
        shared_skill_ids: parseCSV(skillIDs),
      });
      setForm(defaultForm);
      setSkillIDs('');
      await fetchData();
    } catch (e: any) {
      setError(e.message || '创建空间失败');
    } finally {
      setSaving(false);
    }
  }

  const activeSpaces = spaces.filter((item) => (item.status || 'active') === 'active').length;
  const privateSpaces = spaces.filter((item) => (item.visibility || 'private') === 'private').length;
  const providerBoundSpaces = spaces.filter((item) => item.current_provider).length;

  return (
    <div className="space-y-6 animate-fade-in">
      <EnterpriseHero
        eyebrow="用户空间"
        title="为每位员工分配独立空间"
        description="每个空间都可以拥有自己的工作目录、默认项目、模型入口和共享技能挂载，同时仍归属于统一的企业控制面。"
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
        <DataPill label="空间总数" value={spaces.length} />
        <DataPill label="活跃空间" value={activeSpaces} />
        <DataPill label="私有空间" value={privateSpaces} />
        <DataPill label="已绑定模型入口" value={providerBoundSpaces} />
      </div>

      <div className="grid gap-6 xl:grid-cols-[0.92fr_1.08fr]">
        <EnterprisePanel
          title="创建用户空间"
          description="把用户绑定到独立工作目录，并指定默认项目、模型入口和技能集合。"
          action={(
            <div className="inline-flex items-center gap-2 rounded-full border border-emerald-500/15 bg-emerald-500/10 px-3 py-1 text-xs font-medium text-emerald-700 dark:text-emerald-300">
              <ShieldCheck size={13} /> 用户级隔离
            </div>
          )}
        >
          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">所属租户</label>
              <Select value={form.tenant_id || ''} onChange={(e) => setForm((prev) => ({ ...prev, tenant_id: e.target.value }))}>
                <option value="">请选择租户</option>
                {tenants.map((tenant) => (
                  <option key={tenant.id} value={tenant.id}>{tenant.name}</option>
                ))}
              </Select>
            </div>
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">空间负责人</label>
              <Select value={form.owner_user_id || ''} onChange={(e) => setForm((prev) => ({ ...prev, owner_user_id: e.target.value }))}>
                <option value="">请选择负责人</option>
                {users
                  .filter((user) => !form.tenant_id || user.tenant_id === form.tenant_id)
                  .map((user) => (
                    <option key={user.id} value={user.id}>{user.display_name}</option>
                  ))}
              </Select>
            </div>
            <Input
              label="空间名称"
              value={form.name || ''}
              onChange={(e) => setForm((prev) => ({ ...prev, name: e.target.value }))}
              placeholder="财务工作台"
            />
            <Input
              label="工作目录"
              value={form.workspace_dir || ''}
              onChange={(e) => setForm((prev) => ({ ...prev, workspace_dir: e.target.value }))}
              placeholder="D:\\tenant-data\\spaces\\finance-desk"
            />
            <Input
              label="默认项目名"
              value={form.project_name || ''}
              onChange={(e) => setForm((prev) => ({ ...prev, project_name: e.target.value }))}
              placeholder="enterprise-runtime"
            />
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">可见范围</label>
              <Select value={form.visibility || 'private'} onChange={(e) => setForm((prev) => ({ ...prev, visibility: e.target.value }))}>
                <option value="private">private</option>
                <option value="team">team</option>
                <option value="tenant">tenant</option>
              </Select>
            </div>
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">默认模型供应商</label>
              <Select value={form.current_provider || ''} onChange={(e) => setForm((prev) => ({ ...prev, current_provider: e.target.value }))}>
                <option value="">请选择供应商</option>
                {providers.map((provider) => (
                  <option key={provider.id} value={provider.name}>{provider.display_name || provider.name}</option>
                ))}
              </Select>
            </div>
            <Input
              label="默认模型"
              value={form.current_model || ''}
              onChange={(e) => setForm((prev) => ({ ...prev, current_model: e.target.value }))}
              placeholder="claude-sonnet-4"
            />
            <Input
              label="挂载共享技能 ID"
              value={skillIDs}
              onChange={(e) => setSkillIDs(e.target.value)}
              placeholder="skill_finance_table, skill_monthly_report"
            />
          </div>

          <div className="mt-4 flex items-center justify-between gap-3">
            <p className="text-xs text-slate-500 dark:text-slate-400">
              推荐做法是一人一空间，目录、上下文、默认模型都按空间隔离，公共能力通过共享技能和共享机器人提供。
            </p>
            <Button onClick={handleCreateSpace} loading={saving}>
              <Plus size={15} /> 创建空间
            </Button>
          </div>
        </EnterprisePanel>

        <EnterprisePanel
          title="空间清单"
          description="统一查看空间归属、模型入口、技能挂载和最近活跃时间。"
        >
          <div className="mb-4 grid gap-3 md:grid-cols-3">
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
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">负责人筛选</label>
              <Select value={ownerFilter} onChange={(e) => setOwnerFilter(e.target.value)}>
                <option value="">全部用户</option>
                {users.map((user) => (
                  <option key={user.id} value={user.id}>{user.display_name}</option>
                ))}
              </Select>
            </div>
            <div className="rounded-2xl border border-slate-200/80 bg-slate-50/80 px-4 py-3 text-sm text-slate-600 dark:border-white/[0.06] dark:bg-white/[0.03] dark:text-slate-300">
              <div className="flex items-center gap-2 text-xs uppercase tracking-[0.2em] text-slate-400">
                <ServerCog size={13} /> 运行建议
              </div>
              <p className="mt-2 leading-6">控制面可以共用，但工作目录、Provider 和上下文必须按空间进行隔离。</p>
            </div>
          </div>

          <TinyTable>
            <thead className="bg-slate-50/90 text-left text-xs uppercase tracking-[0.18em] text-slate-400 dark:bg-white/[0.03]">
              <tr>
                <th className="px-4 py-3">空间</th>
                <th className="px-4 py-3">负责人</th>
                <th className="px-4 py-3">工作目录</th>
                <th className="px-4 py-3">模型入口</th>
                <th className="px-4 py-3">可见范围</th>
                <th className="px-4 py-3">最近活跃</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-200/80 dark:divide-white/[0.06]">
              {spaces.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-4 py-10 text-center text-sm text-slate-500 dark:text-slate-400">
                    还没有配置空间。
                  </td>
                </tr>
              ) : spaces.map((space) => {
                const owner = users.find((item) => item.id === space.owner_user_id);
                const tenant = tenants.find((item) => item.id === space.tenant_id);
                return (
                  <tr key={space.id}>
                    <td className="px-4 py-3">
                      <div className="flex items-start gap-3">
                        <div className="mt-0.5 rounded-xl bg-emerald-500/10 p-2 text-emerald-600 dark:text-emerald-300">
                          <FolderKanban size={14} />
                        </div>
                        <div>
                          <p className="text-sm font-medium text-slate-900 dark:text-white">{space.name}</p>
                          <p className="text-xs text-slate-400">{tenant?.name || space.tenant_id || '未绑定租户'}</p>
                        </div>
                      </div>
                    </td>
                    <td className="px-4 py-3 text-sm text-slate-600 dark:text-slate-300">{owner?.display_name || space.owner_user_id || '-'}</td>
                    <td className="px-4 py-3 text-sm text-slate-500 dark:text-slate-400">{space.workspace_dir || space.project_name || '-'}</td>
                    <td className="px-4 py-3 text-sm text-slate-500 dark:text-slate-400">
                      {space.current_provider || '-'}
                      {space.current_model ? ` / ${space.current_model}` : ''}
                    </td>
                    <td className="px-4 py-3 text-sm">
                      <Badge variant="outline">{space.visibility || 'private'}</Badge>
                    </td>
                    <td className="px-4 py-3 text-sm text-slate-500 dark:text-slate-400">{formatTime(space.last_interaction_at)}</td>
                  </tr>
                );
              })}
            </tbody>
          </TinyTable>
        </EnterprisePanel>
      </div>
    </div>
  );
}
