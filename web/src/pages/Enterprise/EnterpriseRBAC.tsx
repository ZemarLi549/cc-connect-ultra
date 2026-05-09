import { useCallback, useEffect, useMemo, useState } from 'react';
import { AlertCircle, Plus, RefreshCcw, Shield, ShieldCheck, UserRoundCheck } from 'lucide-react';
import { Badge, Button, Input, Textarea } from '@/components/ui';
import {
  createEnterpriseRole,
  createEnterpriseRoleBinding,
  getEnterpriseEffectiveAccess,
  listEnterprisePermissions,
  listEnterpriseProjects,
  listEnterpriseRoleBindings,
  listEnterpriseRoles,
  listEnterpriseSpaces,
  listEnterpriseTenants,
  listEnterpriseUsers,
  type EnterpriseAccessProfile,
  type EnterprisePermission,
  type EnterpriseProjectProfile,
  type EnterpriseRole,
  type EnterpriseRoleBinding,
  type EnterpriseSpace,
  type EnterpriseTenant,
  type EnterpriseUser,
} from '@/api/enterprise';
import { DataPill, EnterpriseHero, EnterprisePanel, Select, TinyTable } from './shared';
import { formatTime } from './utils';

const defaultRoleForm: Partial<EnterpriseRole> = {
  tenant_id: '',
  name: '',
  description: '',
  scope: 'tenant',
  status: 'active',
  permission_ids: [],
};

const defaultBindingForm: Partial<EnterpriseRoleBinding> = {
  tenant_id: '',
  role_id: '',
  user_id: '',
  space_id: '',
  project_id: '',
  scope: 'tenant',
  status: 'active',
};

export default function EnterpriseRBAC() {
  const [permissions, setPermissions] = useState<EnterprisePermission[]>([]);
  const [tenants, setTenants] = useState<EnterpriseTenant[]>([]);
  const [users, setUsers] = useState<EnterpriseUser[]>([]);
  const [spaces, setSpaces] = useState<EnterpriseSpace[]>([]);
  const [projects, setProjects] = useState<EnterpriseProjectProfile[]>([]);
  const [roles, setRoles] = useState<EnterpriseRole[]>([]);
  const [bindings, setBindings] = useState<EnterpriseRoleBinding[]>([]);
  const [tenantFilter, setTenantFilter] = useState('');
  const [scopeFilter, setScopeFilter] = useState('');
  const [roleForm, setRoleForm] = useState(defaultRoleForm);
  const [bindingForm, setBindingForm] = useState(defaultBindingForm);
  const [selectedPermissionIDs, setSelectedPermissionIDs] = useState<string[]>([]);
  const [bindingMetadataInput, setBindingMetadataInput] = useState('{}');
  const [roleMetadataInput, setRoleMetadataInput] = useState('{}');
  const [accessQuery, setAccessQuery] = useState({
    tenant_id: '',
    user_id: '',
    space_id: '',
    project_id: '',
  });
  const [accessProfile, setAccessProfile] = useState<EnterpriseAccessProfile | null>(null);
  const [loading, setLoading] = useState(true);
  const [savingRole, setSavingRole] = useState(false);
  const [savingBinding, setSavingBinding] = useState(false);
  const [resolving, setResolving] = useState(false);
  const [error, setError] = useState('');

  const fetchData = useCallback(async () => {
    try {
      setLoading(true);
      setError('');
      const [permissionData, tenantData, userData, spaceData, projectData, roleData, bindingData] = await Promise.all([
        listEnterprisePermissions(),
        listEnterpriseTenants(),
        listEnterpriseUsers(tenantFilter ? { tenant_id: tenantFilter } : undefined),
        listEnterpriseSpaces(tenantFilter ? { tenant_id: tenantFilter } : undefined),
        listEnterpriseProjects(tenantFilter ? { tenant_id: tenantFilter } : undefined),
        listEnterpriseRoles({
          tenant_id: tenantFilter || undefined,
          scope: scopeFilter || undefined,
        }),
        listEnterpriseRoleBindings({
          tenant_id: tenantFilter || undefined,
          scope: scopeFilter || undefined,
        }),
      ]);
      setPermissions(permissionData.permissions || []);
      setTenants(tenantData.tenants || []);
      setUsers(userData.users || []);
      setSpaces(spaceData.spaces || []);
      setProjects(projectData.projects || []);
      setRoles(roleData.roles || []);
      setBindings(bindingData.bindings || []);
    } catch (e: any) {
      setError(e.message || '加载 RBAC 数据失败');
    } finally {
      setLoading(false);
    }
  }, [scopeFilter, tenantFilter]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const permissionsByGroup = useMemo(() => {
    return permissions.reduce<Record<string, EnterprisePermission[]>>((acc, item) => {
      const key = item.group || 'other';
      if (!acc[key]) acc[key] = [];
      acc[key].push(item);
      return acc;
    }, {});
  }, [permissions]);

  function togglePermission(permissionID: string) {
    setSelectedPermissionIDs((current) => {
      if (current.includes(permissionID)) {
        return current.filter((item) => item !== permissionID);
      }
      return [...current, permissionID];
    });
  }

  async function handleCreateRole() {
    if (!roleForm.name) {
      setError('角色名称不能为空');
      return;
    }
    if (selectedPermissionIDs.length === 0) {
      setError('至少要选择一个权限');
      return;
    }
    let metadata: Record<string, string> | undefined;
    try {
      metadata = roleMetadataInput.trim() ? JSON.parse(roleMetadataInput) : undefined;
    } catch {
      setError('角色元数据必须是合法 JSON');
      return;
    }
    try {
      setSavingRole(true);
      setError('');
      await createEnterpriseRole({
        ...roleForm,
        permission_ids: selectedPermissionIDs,
        metadata,
      });
      setRoleForm(defaultRoleForm);
      setSelectedPermissionIDs([]);
      setRoleMetadataInput('{}');
      await fetchData();
    } catch (e: any) {
      setError(e.message || '创建角色失败');
    } finally {
      setSavingRole(false);
    }
  }

  async function handleCreateBinding() {
    if (!bindingForm.role_id || !bindingForm.user_id) {
      setError('角色和用户不能为空');
      return;
    }
    let metadata: Record<string, string> | undefined;
    try {
      metadata = bindingMetadataInput.trim() ? JSON.parse(bindingMetadataInput) : undefined;
    } catch {
      setError('绑定元数据必须是合法 JSON');
      return;
    }
    try {
      setSavingBinding(true);
      setError('');
      await createEnterpriseRoleBinding({
        ...bindingForm,
        metadata,
      });
      setBindingForm(defaultBindingForm);
      setBindingMetadataInput('{}');
      await fetchData();
    } catch (e: any) {
      setError(e.message || '创建角色绑定失败');
    } finally {
      setSavingBinding(false);
    }
  }

  async function handleResolveAccess() {
    if (!accessQuery.user_id) {
      setError('请选择要解析的用户');
      return;
    }
    try {
      setResolving(true);
      setError('');
      const profile = await getEnterpriseEffectiveAccess({
        user_id: accessQuery.user_id,
        tenant_id: accessQuery.tenant_id || undefined,
        space_id: accessQuery.space_id || undefined,
        project_id: accessQuery.project_id || undefined,
      });
      setAccessProfile(profile);
    } catch (e: any) {
      setError(e.message || '解析有效权限失败');
    } finally {
      setResolving(false);
    }
  }

  return (
    <div className="space-y-6 animate-fade-in">
      <EnterpriseHero
        eyebrow="角色权限"
        title="企业 RBAC 与权限分配"
        description="支持全局、租户、空间、项目四级作用域的角色与绑定管理，并可实时解析用户在某个上下文下的有效权限。"
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
        <DataPill label="权限项" value={permissions.length} />
        <DataPill label="角色数" value={roles.length} />
        <DataPill label="绑定数" value={bindings.length} />
        <DataPill label="项目作用域" value={bindings.filter((item) => item.scope === 'project').length} />
      </div>

      <div className="grid gap-6 xl:grid-cols-[1fr_1fr]">
        <EnterprisePanel
          title="创建角色"
          description="把多个能力点组合成一个可复用角色，用于后续按租户、空间、项目批量授权。"
          action={(
            <div className="inline-flex items-center gap-2 rounded-full border border-emerald-500/15 bg-emerald-500/10 px-3 py-1 text-xs font-medium text-emerald-700 dark:text-emerald-300">
              <Shield size={13} /> 权限模板
            </div>
          )}
        >
          <div className="grid gap-4 md:grid-cols-2">
            <Input
              label="角色名称"
              value={roleForm.name || ''}
              onChange={(e) => setRoleForm((prev) => ({ ...prev, name: e.target.value }))}
              placeholder="Space Admin"
            />
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">所属租户</label>
              <Select value={roleForm.tenant_id || ''} onChange={(e) => setRoleForm((prev) => ({ ...prev, tenant_id: e.target.value }))}>
                <option value="">全局角色</option>
                {tenants.map((tenant) => (
                  <option key={tenant.id} value={tenant.id}>{tenant.name}</option>
                ))}
              </Select>
            </div>
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">作用域</label>
              <Select value={roleForm.scope || 'tenant'} onChange={(e) => setRoleForm((prev) => ({ ...prev, scope: e.target.value }))}>
                <option value="global">global</option>
                <option value="tenant">tenant</option>
                <option value="space">space</option>
                <option value="project">project</option>
              </Select>
            </div>
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">状态</label>
              <Select value={roleForm.status || 'active'} onChange={(e) => setRoleForm((prev) => ({ ...prev, status: e.target.value }))}>
                <option value="active">active</option>
                <option value="draft">draft</option>
                <option value="disabled">disabled</option>
              </Select>
            </div>
          </div>

          <div className="mt-4">
            <Textarea
              label="角色说明"
              value={roleForm.description || ''}
              onChange={(e) => setRoleForm((prev) => ({ ...prev, description: e.target.value }))}
              placeholder="拥有空间查看、空间管理、项目配置和角色查看能力。"
              rows={3}
            />
          </div>

          <div className="mt-4 space-y-4">
            {Object.entries(permissionsByGroup).map(([group, items]) => (
              <div key={group} className="rounded-2xl border border-slate-200/80 bg-slate-50/80 p-4 dark:border-white/[0.06] dark:bg-white/[0.03]">
                <p className="text-xs uppercase tracking-[0.18em] text-slate-400">{group}</p>
                <div className="mt-3 flex flex-wrap gap-2">
                  {items.map((item) => {
                    const active = selectedPermissionIDs.includes(item.id);
                    return (
                      <button
                        key={item.id}
                        type="button"
                        onClick={() => togglePermission(item.id)}
                        className={`rounded-full border px-3 py-1 text-xs transition ${
                          active
                            ? 'border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300'
                            : 'border-slate-200/80 text-slate-600 hover:border-emerald-500/30 hover:text-emerald-700 dark:border-white/[0.08] dark:text-slate-300'
                        }`}
                        title={item.description}
                      >
                        {item.id}
                      </button>
                    );
                  })}
                </div>
              </div>
            ))}
          </div>

          <div className="mt-4">
            <Textarea
              label="角色元数据 JSON"
              value={roleMetadataInput}
              onChange={(e) => setRoleMetadataInput(e.target.value)}
              rows={4}
            />
          </div>

          <div className="mt-4 flex items-center justify-between gap-3">
            <p className="text-xs text-slate-500 dark:text-slate-400">
              角色建议按职责组合，不要把所有权限堆进一个管理员角色，后续绑定会更清晰。
            </p>
            <Button onClick={handleCreateRole} loading={savingRole}>
              <Plus size={15} /> 创建角色
            </Button>
          </div>
        </EnterprisePanel>

        <EnterprisePanel
          title="创建角色绑定"
          description="把角色分配给具体用户，并限定作用到租户、空间或项目。"
          action={(
            <div className="inline-flex items-center gap-2 rounded-full border border-emerald-500/15 bg-emerald-500/10 px-3 py-1 text-xs font-medium text-emerald-700 dark:text-emerald-300">
              <UserRoundCheck size={13} /> 用户授权
            </div>
          )}
        >
          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">所属租户</label>
              <Select value={bindingForm.tenant_id || ''} onChange={(e) => setBindingForm((prev) => ({ ...prev, tenant_id: e.target.value }))}>
                <option value="">全局绑定</option>
                {tenants.map((tenant) => (
                  <option key={tenant.id} value={tenant.id}>{tenant.name}</option>
                ))}
              </Select>
            </div>
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">角色</label>
              <Select value={bindingForm.role_id || ''} onChange={(e) => setBindingForm((prev) => ({ ...prev, role_id: e.target.value }))}>
                <option value="">请选择角色</option>
                {roles
                  .filter((role) => !bindingForm.tenant_id || !role.tenant_id || role.tenant_id === bindingForm.tenant_id)
                  .map((role) => (
                    <option key={role.id} value={role.id}>{role.name}</option>
                  ))}
              </Select>
            </div>
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">用户</label>
              <Select value={bindingForm.user_id || ''} onChange={(e) => setBindingForm((prev) => ({ ...prev, user_id: e.target.value }))}>
                <option value="">请选择用户</option>
                {users
                  .filter((user) => !bindingForm.tenant_id || user.tenant_id === bindingForm.tenant_id)
                  .map((user) => (
                    <option key={user.id} value={user.id}>{user.display_name}</option>
                  ))}
              </Select>
            </div>
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">作用域</label>
              <Select value={bindingForm.scope || 'tenant'} onChange={(e) => setBindingForm((prev) => ({ ...prev, scope: e.target.value }))}>
                <option value="global">global</option>
                <option value="tenant">tenant</option>
                <option value="space">space</option>
                <option value="project">project</option>
              </Select>
            </div>
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">空间</label>
              <Select value={bindingForm.space_id || ''} onChange={(e) => setBindingForm((prev) => ({ ...prev, space_id: e.target.value }))}>
                <option value="">不指定空间</option>
                {spaces
                  .filter((space) => !bindingForm.tenant_id || space.tenant_id === bindingForm.tenant_id)
                  .map((space) => (
                    <option key={space.id} value={space.id}>{space.name}</option>
                  ))}
              </Select>
            </div>
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">项目</label>
              <Select value={bindingForm.project_id || ''} onChange={(e) => setBindingForm((prev) => ({ ...prev, project_id: e.target.value }))}>
                <option value="">不指定项目</option>
                {projects
                  .filter((project) => !bindingForm.tenant_id || project.tenant_id === bindingForm.tenant_id)
                  .map((project) => (
                    <option key={project.id} value={project.id}>{project.name}</option>
                  ))}
              </Select>
            </div>
          </div>

          <div className="mt-4">
            <Textarea
              label="绑定元数据 JSON"
              value={bindingMetadataInput}
              onChange={(e) => setBindingMetadataInput(e.target.value)}
              rows={4}
            />
          </div>

          <div className="mt-4 flex items-center justify-between gap-3">
            <p className="text-xs text-slate-500 dark:text-slate-400">
              如果绑定范围是 `space` 或 `project`，建议显式选中对应空间或项目，避免权限含义模糊。
            </p>
            <Button onClick={handleCreateBinding} loading={savingBinding}>
              <Plus size={15} /> 创建绑定
            </Button>
          </div>
        </EnterprisePanel>
      </div>

      <div className="grid gap-6 xl:grid-cols-[0.96fr_1.04fr]">
        <EnterprisePanel title="角色与绑定清单" description="按租户和作用域查看当前 RBAC 配置。">
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
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">作用域筛选</label>
              <Select value={scopeFilter} onChange={(e) => setScopeFilter(e.target.value)}>
                <option value="">全部作用域</option>
                <option value="global">global</option>
                <option value="tenant">tenant</option>
                <option value="space">space</option>
                <option value="project">project</option>
              </Select>
            </div>
          </div>

          <div className="space-y-6">
            <TinyTable>
              <thead className="bg-slate-50/90 text-left text-xs uppercase tracking-[0.18em] text-slate-400 dark:bg-white/[0.03]">
                <tr>
                  <th className="px-4 py-3">角色</th>
                  <th className="px-4 py-3">作用域</th>
                  <th className="px-4 py-3">权限数</th>
                  <th className="px-4 py-3">更新时间</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-200/80 dark:divide-white/[0.06]">
                {roles.length === 0 ? (
                  <tr>
                    <td colSpan={4} className="px-4 py-8 text-center text-sm text-slate-500 dark:text-slate-400">暂无角色。</td>
                  </tr>
                ) : roles.map((role) => (
                  <tr key={role.id}>
                    <td className="px-4 py-3">
                      <div>
                        <p className="text-sm font-medium text-slate-900 dark:text-white">{role.name}</p>
                        <p className="text-xs text-slate-400">{role.description || role.tenant_id || '全局角色'}</p>
                      </div>
                    </td>
                    <td className="px-4 py-3 text-sm"><Badge variant="outline">{role.scope || 'tenant'}</Badge></td>
                    <td className="px-4 py-3 text-sm text-slate-500 dark:text-slate-400">{role.permission_ids?.length || 0}</td>
                    <td className="px-4 py-3 text-sm text-slate-500 dark:text-slate-400">{formatTime(role.updated_at)}</td>
                  </tr>
                ))}
              </tbody>
            </TinyTable>

            <TinyTable>
              <thead className="bg-slate-50/90 text-left text-xs uppercase tracking-[0.18em] text-slate-400 dark:bg-white/[0.03]">
                <tr>
                  <th className="px-4 py-3">绑定</th>
                  <th className="px-4 py-3">范围</th>
                  <th className="px-4 py-3">目标</th>
                  <th className="px-4 py-3">更新时间</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-200/80 dark:divide-white/[0.06]">
                {bindings.length === 0 ? (
                  <tr>
                    <td colSpan={4} className="px-4 py-8 text-center text-sm text-slate-500 dark:text-slate-400">暂无绑定。</td>
                  </tr>
                ) : bindings.map((binding) => {
                  const role = roles.find((item) => item.id === binding.role_id);
                  const user = users.find((item) => item.id === binding.user_id);
                  return (
                    <tr key={binding.id}>
                      <td className="px-4 py-3">
                        <div>
                          <p className="text-sm font-medium text-slate-900 dark:text-white">{role?.name || binding.role_id}</p>
                          <p className="text-xs text-slate-400">{user?.display_name || binding.user_id}</p>
                        </div>
                      </td>
                      <td className="px-4 py-3 text-sm"><Badge variant="outline">{binding.scope || 'tenant'}</Badge></td>
                      <td className="px-4 py-3 text-sm text-slate-500 dark:text-slate-400">{binding.project_id || binding.space_id || binding.tenant_id || 'global'}</td>
                      <td className="px-4 py-3 text-sm text-slate-500 dark:text-slate-400">{formatTime(binding.updated_at)}</td>
                    </tr>
                  );
                })}
              </tbody>
            </TinyTable>
          </div>
        </EnterprisePanel>

        <EnterprisePanel
          title="有效权限解析"
          description="选中用户和上下文，实时查看系统最终解析出的角色和权限。"
          action={(
            <div className="inline-flex items-center gap-2 rounded-full border border-emerald-500/15 bg-emerald-500/10 px-3 py-1 text-xs font-medium text-emerald-700 dark:text-emerald-300">
              <ShieldCheck size={13} /> 权限求值
            </div>
          )}
        >
          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">租户</label>
              <Select value={accessQuery.tenant_id} onChange={(e) => setAccessQuery((prev) => ({ ...prev, tenant_id: e.target.value }))}>
                <option value="">全局</option>
                {tenants.map((tenant) => (
                  <option key={tenant.id} value={tenant.id}>{tenant.name}</option>
                ))}
              </Select>
            </div>
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">用户</label>
              <Select value={accessQuery.user_id} onChange={(e) => setAccessQuery((prev) => ({ ...prev, user_id: e.target.value }))}>
                <option value="">请选择用户</option>
                {users
                  .filter((user) => !accessQuery.tenant_id || user.tenant_id === accessQuery.tenant_id)
                  .map((user) => (
                    <option key={user.id} value={user.id}>{user.display_name}</option>
                  ))}
              </Select>
            </div>
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">空间</label>
              <Select value={accessQuery.space_id} onChange={(e) => setAccessQuery((prev) => ({ ...prev, space_id: e.target.value }))}>
                <option value="">不指定空间</option>
                {spaces
                  .filter((space) => !accessQuery.tenant_id || space.tenant_id === accessQuery.tenant_id)
                  .map((space) => (
                    <option key={space.id} value={space.id}>{space.name}</option>
                  ))}
              </Select>
            </div>
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">项目</label>
              <Select value={accessQuery.project_id} onChange={(e) => setAccessQuery((prev) => ({ ...prev, project_id: e.target.value }))}>
                <option value="">不指定项目</option>
                {projects
                  .filter((project) => !accessQuery.tenant_id || project.tenant_id === accessQuery.tenant_id)
                  .map((project) => (
                    <option key={project.id} value={project.id}>{project.name}</option>
                  ))}
              </Select>
            </div>
          </div>

          <div className="mt-4 flex justify-end">
            <Button onClick={handleResolveAccess} loading={resolving}>解析权限</Button>
          </div>

          <div className="mt-4 grid gap-4 md:grid-cols-2">
            <div className="rounded-2xl border border-slate-200/80 bg-slate-50/80 p-4 dark:border-white/[0.06] dark:bg-white/[0.03]">
              <p className="text-xs uppercase tracking-[0.18em] text-slate-400">命中角色</p>
              <div className="mt-3 flex flex-wrap gap-2">
                {(accessProfile?.roles || []).length === 0 ? (
                  <p className="text-sm text-slate-500 dark:text-slate-400">还没有解析结果。</p>
                ) : accessProfile?.roles?.map((role) => (
                  <Badge key={role.id} variant="info">{role.name}</Badge>
                ))}
              </div>
            </div>
            <div className="rounded-2xl border border-slate-200/80 bg-slate-50/80 p-4 dark:border-white/[0.06] dark:bg-white/[0.03]">
              <p className="text-xs uppercase tracking-[0.18em] text-slate-400">有效权限</p>
              <div className="mt-3 flex flex-wrap gap-2">
                {(accessProfile?.permission_ids || []).length === 0 ? (
                  <p className="text-sm text-slate-500 dark:text-slate-400">还没有解析结果。</p>
                ) : accessProfile?.permission_ids?.map((permissionID) => (
                  <Badge key={permissionID} variant="outline">{permissionID}</Badge>
                ))}
              </div>
            </div>
          </div>
        </EnterprisePanel>
      </div>
    </div>
  );
}
