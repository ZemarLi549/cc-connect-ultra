import { useCallback, useEffect, useState } from 'react';
import { AlertCircle, Download, Plus, Puzzle, RefreshCcw, Workflow } from 'lucide-react';
import { Badge, Button, Input, Textarea } from '@/components/ui';
import {
  createEnterpriseImport,
  createEnterpriseSkill,
  listEnterpriseImports,
  listEnterpriseSkills,
  listEnterpriseTenants,
  listEnterpriseUsers,
  type EnterpriseImportJob,
  type EnterpriseSkill,
  type EnterpriseTenant,
  type EnterpriseUser,
} from '@/api/enterprise';
import { DataPill, EnterpriseHero, EnterprisePanel, Select, TinyTable } from './shared';
import { formatTime, parseCSV } from './utils';

const defaultSkillForm: Partial<EnterpriseSkill> = {
  tenant_id: '',
  owner_user_id: '',
  name: '',
  display_name: '',
  description: '',
  scope: 'private',
  status: 'draft',
  version: 'v1',
  prompt: '',
  source_path: '',
  tags: [],
};

const defaultImportForm = {
  tenant_id: '',
  owner_user_id: '',
  source_type: 'cocoloop',
  source_name: '',
  source_ref: '',
  status: 'queued',
};

export default function EnterpriseSkills() {
  const [tenants, setTenants] = useState<EnterpriseTenant[]>([]);
  const [users, setUsers] = useState<EnterpriseUser[]>([]);
  const [skills, setSkills] = useState<EnterpriseSkill[]>([]);
  const [imports, setImports] = useState<EnterpriseImportJob[]>([]);
  const [scopeFilter, setScopeFilter] = useState('');
  const [tenantFilter, setTenantFilter] = useState('');
  const [skillForm, setSkillForm] = useState(defaultSkillForm);
  const [importForm, setImportForm] = useState(defaultImportForm);
  const [tagsInput, setTagsInput] = useState('');
  const [loading, setLoading] = useState(true);
  const [savingSkill, setSavingSkill] = useState(false);
  const [savingImport, setSavingImport] = useState(false);
  const [error, setError] = useState('');

  const fetchData = useCallback(async () => {
    try {
      setLoading(true);
      setError('');
      const [tenantData, userData, skillData, importData] = await Promise.all([
        listEnterpriseTenants(),
        listEnterpriseUsers(tenantFilter ? { tenant_id: tenantFilter } : undefined),
        listEnterpriseSkills({
          scope: scopeFilter || undefined,
          tenant_id: tenantFilter || undefined,
        }),
        listEnterpriseImports(tenantFilter ? { tenant_id: tenantFilter } : undefined),
      ]);
      setTenants(tenantData.tenants || []);
      setUsers(userData.users || []);
      setSkills(skillData.skills || []);
      setImports(importData.imports || []);
    } catch (e: any) {
      setError(e.message || '加载技能工坊失败');
    } finally {
      setLoading(false);
    }
  }, [scopeFilter, tenantFilter]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  async function handleCreateSkill() {
    if (!skillForm.name) {
      setError('技能名称不能为空');
      return;
    }
    try {
      setSavingSkill(true);
      setError('');
      await createEnterpriseSkill({
        ...skillForm,
        tags: parseCSV(tagsInput),
      });
      setSkillForm(defaultSkillForm);
      setTagsInput('');
      await fetchData();
    } catch (e: any) {
      setError(e.message || '创建技能失败');
    } finally {
      setSavingSkill(false);
    }
  }

  async function handleCreateImport() {
    if (!importForm.source_name && !importForm.source_ref) {
      setError('导入来源名称或来源地址不能为空');
      return;
    }
    try {
      setSavingImport(true);
      setError('');
      await createEnterpriseImport(importForm);
      setImportForm(defaultImportForm);
      await fetchData();
    } catch (e: any) {
      setError(e.message || '创建导入任务失败');
    } finally {
      setSavingImport(false);
    }
  }

  const publishedCount = skills.filter((item) => (item.status || 'draft') === 'published').length;
  const publicCount = skills.filter((item) => (item.scope || 'private') === 'public').length;
  const tenantCount = skills.filter((item) => (item.scope || 'private') === 'tenant').length;

  return (
    <div className="space-y-6 animate-fade-in">
      <EnterpriseHero
        eyebrow="技能工坊"
        title="让技能成为企业内部可复用资产"
        description="支持每个人创建私有技能，也支持租户共享、公共开放，以及 Cocoloop、Git 等外部来源导入。"
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
        <DataPill label="技能总数" value={skills.length} />
        <DataPill label="已发布" value={publishedCount} />
        <DataPill label="租户共享" value={tenantCount} />
        <DataPill label="公共技能" value={publicCount} />
      </div>

      <div className="grid gap-6 xl:grid-cols-[0.92fr_1.08fr]">
        <div className="space-y-6">
          <EnterprisePanel
            title="创建技能"
            description="把提示词、约束和输出规范沉淀为技能，并决定它是个人私有、租户共享还是公共能力。"
          >
            <div className="grid gap-4 md:grid-cols-2">
              <Input
                label="技能名称"
                value={skillForm.name || ''}
                onChange={(e) => setSkillForm((prev) => ({ ...prev, name: e.target.value }))}
                placeholder="table-operator"
              />
              <Input
                label="展示名称"
                value={skillForm.display_name || ''}
                onChange={(e) => setSkillForm((prev) => ({ ...prev, display_name: e.target.value }))}
                placeholder="表格处理助手"
              />
              <div className="space-y-1.5">
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">作用范围</label>
                <Select value={skillForm.scope || 'private'} onChange={(e) => setSkillForm((prev) => ({ ...prev, scope: e.target.value }))}>
                  <option value="private">private</option>
                  <option value="tenant">tenant</option>
                  <option value="public">public</option>
                </Select>
              </div>
              <div className="space-y-1.5">
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">状态</label>
                <Select value={skillForm.status || 'draft'} onChange={(e) => setSkillForm((prev) => ({ ...prev, status: e.target.value }))}>
                  <option value="draft">draft</option>
                  <option value="published">published</option>
                  <option value="archived">archived</option>
                </Select>
              </div>
              <div className="space-y-1.5">
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">所属租户</label>
                <Select value={skillForm.tenant_id || ''} onChange={(e) => setSkillForm((prev) => ({ ...prev, tenant_id: e.target.value }))}>
                  <option value="">不绑定租户</option>
                  {tenants.map((tenant) => (
                    <option key={tenant.id} value={tenant.id}>{tenant.name}</option>
                  ))}
                </Select>
              </div>
              <div className="space-y-1.5">
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">创建人</label>
                <Select value={skillForm.owner_user_id || ''} onChange={(e) => setSkillForm((prev) => ({ ...prev, owner_user_id: e.target.value }))}>
                  <option value="">不指定创建人</option>
                  {users
                    .filter((user) => !skillForm.tenant_id || user.tenant_id === skillForm.tenant_id)
                    .map((user) => (
                      <option key={user.id} value={user.id}>{user.display_name}</option>
                    ))}
                </Select>
              </div>
              <Input
                label="版本号"
                value={skillForm.version || ''}
                onChange={(e) => setSkillForm((prev) => ({ ...prev, version: e.target.value }))}
                placeholder="v1"
              />
              <Input
                label="标签"
                value={tagsInput}
                onChange={(e) => setTagsInput(e.target.value)}
                placeholder="表格, 报表, 办公"
              />
              <Input
                label="源文件路径"
                value={skillForm.source_path || ''}
                onChange={(e) => setSkillForm((prev) => ({ ...prev, source_path: e.target.value }))}
                placeholder="skills/table-operator/SKILL.md"
              />
            </div>

            <div className="mt-4">
              <Textarea
                label="技能说明"
                value={skillForm.description || ''}
                onChange={(e) => setSkillForm((prev) => ({ ...prev, description: e.target.value }))}
                placeholder="处理表格、整理字段、生成总结，输出适合办公场景的结构化结果。"
                rows={3}
              />
            </div>
            <div className="mt-4">
              <Textarea
                label="Prompt / 技能内容"
                value={skillForm.prompt || ''}
                onChange={(e) => setSkillForm((prev) => ({ ...prev, prompt: e.target.value }))}
                placeholder="当用户请求修改表格时，先确认目标列，保留原值，并输出清晰的变更说明。"
                rows={8}
              />
            </div>

            <div className="mt-4 flex items-center justify-between gap-3">
              <p className="text-xs text-slate-500 dark:text-slate-400">
                建议先从私有技能试运行，稳定后再升级为租户级或公共技能。
              </p>
              <Button onClick={handleCreateSkill} loading={savingSkill}>
                <Plus size={15} /> 创建技能
              </Button>
            </div>
          </EnterprisePanel>

          <EnterprisePanel
            title="导入技能"
            description="将 Cocoloop、Git 或其他来源的技能任务统一纳入企业技能目录。"
            action={(
              <div className="inline-flex items-center gap-2 rounded-full border border-emerald-500/15 bg-emerald-500/10 px-3 py-1 text-xs font-medium text-emerald-700 dark:text-emerald-300">
                <Download size={13} /> 外部导入
              </div>
            )}
          >
            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-1.5">
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">来源类型</label>
                <Select value={importForm.source_type} onChange={(e) => setImportForm((prev) => ({ ...prev, source_type: e.target.value }))}>
                  <option value="cocoloop">cocoloop</option>
                  <option value="git">git</option>
                  <option value="zip">zip</option>
                  <option value="manual">manual</option>
                </Select>
              </div>
              <Input
                label="来源名称"
                value={importForm.source_name}
                onChange={(e) => setImportForm((prev) => ({ ...prev, source_name: e.target.value }))}
                placeholder="office-skill-bundle"
              />
              <Input
                label="来源地址"
                value={importForm.source_ref}
                onChange={(e) => setImportForm((prev) => ({ ...prev, source_ref: e.target.value }))}
                placeholder="https://cocoloop.example.com/workspaces/office-skills"
              />
              <div className="space-y-1.5">
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">所属租户</label>
                <Select value={importForm.tenant_id} onChange={(e) => setImportForm((prev) => ({ ...prev, tenant_id: e.target.value }))}>
                  <option value="">全局导入</option>
                  {tenants.map((tenant) => (
                    <option key={tenant.id} value={tenant.id}>{tenant.name}</option>
                  ))}
                </Select>
              </div>
            </div>

            <div className="mt-4 flex items-center justify-between gap-3">
              <p className="text-xs text-slate-500 dark:text-slate-400">
                导入任务当前会先写入企业控制面记录，后续可继续接入 Worker 做自动拉取与落库。
              </p>
              <Button onClick={handleCreateImport} loading={savingImport}>
                <Workflow size={15} /> 创建导入任务
              </Button>
            </div>
          </EnterprisePanel>
        </div>

        <EnterprisePanel
          title="技能目录与导入历史"
          description="查看已经沉淀的技能资产，以及所有外部导入任务。"
        >
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
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">范围筛选</label>
              <Select value={scopeFilter} onChange={(e) => setScopeFilter(e.target.value)}>
                <option value="">全部范围</option>
                <option value="private">private</option>
                <option value="tenant">tenant</option>
                <option value="public">public</option>
              </Select>
            </div>
          </div>

          <div className="space-y-6">
            <TinyTable>
              <thead className="bg-slate-50/90 text-left text-xs uppercase tracking-[0.18em] text-slate-400 dark:bg-white/[0.03]">
                <tr>
                  <th className="px-4 py-3">技能</th>
                  <th className="px-4 py-3">范围</th>
                  <th className="px-4 py-3">状态</th>
                  <th className="px-4 py-3">标签</th>
                  <th className="px-4 py-3">更新时间</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-200/80 dark:divide-white/[0.06]">
                {skills.length === 0 ? (
                  <tr>
                    <td colSpan={5} className="px-4 py-10 text-center text-sm text-slate-500 dark:text-slate-400">
                      还没有创建技能。
                    </td>
                  </tr>
                ) : skills.map((skill) => (
                  <tr key={skill.id}>
                    <td className="px-4 py-3">
                      <div className="flex items-start gap-3">
                        <div className="mt-0.5 rounded-xl bg-emerald-500/10 p-2 text-emerald-600 dark:text-emerald-300">
                          <Puzzle size={14} />
                        </div>
                        <div>
                          <p className="text-sm font-medium text-slate-900 dark:text-white">{skill.display_name || skill.name}</p>
                          <p className="text-xs text-slate-400">{skill.description || skill.source_path || '-'}</p>
                        </div>
                      </div>
                    </td>
                    <td className="px-4 py-3 text-sm"><Badge variant="outline">{skill.scope || 'private'}</Badge></td>
                    <td className="px-4 py-3 text-sm"><Badge variant={(skill.status || 'draft') === 'published' ? 'success' : 'outline'}>{skill.status || 'draft'}</Badge></td>
                    <td className="px-4 py-3 text-sm text-slate-500 dark:text-slate-400">{(skill.tags || []).join(', ') || '-'}</td>
                    <td className="px-4 py-3 text-sm text-slate-500 dark:text-slate-400">{formatTime(skill.updated_at)}</td>
                  </tr>
                ))}
              </tbody>
            </TinyTable>

            <TinyTable>
              <thead className="bg-slate-50/90 text-left text-xs uppercase tracking-[0.18em] text-slate-400 dark:bg-white/[0.03]">
                <tr>
                  <th className="px-4 py-3">导入任务</th>
                  <th className="px-4 py-3">来源</th>
                  <th className="px-4 py-3">状态</th>
                  <th className="px-4 py-3">导入数量</th>
                  <th className="px-4 py-3">更新时间</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-200/80 dark:divide-white/[0.06]">
                {imports.length === 0 ? (
                  <tr>
                    <td colSpan={5} className="px-4 py-10 text-center text-sm text-slate-500 dark:text-slate-400">
                      还没有导入任务。
                    </td>
                  </tr>
                ) : imports.map((item) => (
                  <tr key={item.id}>
                    <td className="px-4 py-3 text-sm font-medium text-slate-900 dark:text-white">{item.source_name || item.id}</td>
                    <td className="px-4 py-3 text-sm text-slate-500 dark:text-slate-400">{item.source_type || 'manual'} / {item.source_ref || '-'}</td>
                    <td className="px-4 py-3 text-sm"><Badge variant="outline">{item.status || 'queued'}</Badge></td>
                    <td className="px-4 py-3 text-sm text-slate-500 dark:text-slate-400">{item.imported_skills ?? 0}</td>
                    <td className="px-4 py-3 text-sm text-slate-500 dark:text-slate-400">{formatTime(item.updated_at)}</td>
                  </tr>
                ))}
              </tbody>
            </TinyTable>
          </div>
        </EnterprisePanel>
      </div>
    </div>
  );
}
