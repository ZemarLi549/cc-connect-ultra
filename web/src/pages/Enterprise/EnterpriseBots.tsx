import { useCallback, useEffect, useState } from 'react';
import { AlertCircle, Bot, Plus, RefreshCcw, Share2, Sparkles } from 'lucide-react';
import { Badge, Button, Input, Textarea } from '@/components/ui';
import {
  createEnterpriseBot,
  listEnterpriseBots,
  listEnterpriseProviders,
  listEnterpriseSkills,
  listEnterpriseTenants,
  listEnterpriseUsers,
  type EnterpriseBot,
  type EnterpriseProvider,
  type EnterpriseSkill,
  type EnterpriseTenant,
  type EnterpriseUser,
} from '@/api/enterprise';
import { DataPill, EnterpriseHero, EnterprisePanel, Select, TinyTable } from './shared';
import { formatTime, parseCSV } from './utils';

const defaultForm: Partial<EnterpriseBot> = {
  tenant_id: '',
  owner_user_id: '',
  name: '',
  description: '',
  scope: 'tenant',
  provider_name: '',
  model_name: '',
  skill_ids: [],
  status: 'active',
};

export default function EnterpriseBots() {
  const [tenants, setTenants] = useState<EnterpriseTenant[]>([]);
  const [users, setUsers] = useState<EnterpriseUser[]>([]);
  const [skills, setSkills] = useState<EnterpriseSkill[]>([]);
  const [providers, setProviders] = useState<EnterpriseProvider[]>([]);
  const [bots, setBots] = useState<EnterpriseBot[]>([]);
  const [tenantFilter, setTenantFilter] = useState('');
  const [scopeFilter, setScopeFilter] = useState('');
  const [form, setForm] = useState(defaultForm);
  const [skillIDs, setSkillIDs] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const fetchData = useCallback(async () => {
    try {
      setLoading(true);
      setError('');
      const [tenantData, userData, skillData, providerData, botData] = await Promise.all([
        listEnterpriseTenants(),
        listEnterpriseUsers(tenantFilter ? { tenant_id: tenantFilter } : undefined),
        listEnterpriseSkills(),
        listEnterpriseProviders(),
        listEnterpriseBots(tenantFilter ? { tenant_id: tenantFilter } : undefined),
      ]);
      const nextBots = botData.bots || [];
      setTenants(tenantData.tenants || []);
      setUsers(userData.users || []);
      setSkills(skillData.skills || []);
      setProviders(providerData.providers || []);
      setBots(scopeFilter ? nextBots.filter((item) => (item.scope || 'tenant') === scopeFilter) : nextBots);
    } catch (e: any) {
      setError(e.message || '加载共享机器人失败');
    } finally {
      setLoading(false);
    }
  }, [scopeFilter, tenantFilter]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  async function handleCreateBot() {
    if (!form.name) {
      setError('机器人名称不能为空');
      return;
    }
    try {
      setSaving(true);
      setError('');
      await createEnterpriseBot({
        ...form,
        skill_ids: parseCSV(skillIDs),
      });
      setForm(defaultForm);
      setSkillIDs('');
      await fetchData();
    } catch (e: any) {
      setError(e.message || '创建机器人失败');
    } finally {
      setSaving(false);
    }
  }

  const publicCount = bots.filter((item) => (item.scope || 'tenant') === 'public').length;
  const tenantCount = bots.filter((item) => (item.scope || 'tenant') === 'tenant').length;
  const personalCount = bots.filter((item) => (item.scope || 'tenant') === 'personal').length;

  return (
    <div className="space-y-6 animate-fade-in">
      <EnterpriseHero
        eyebrow="共享机器人"
        title="把高频办公能力做成统一助手"
        description="机器人适合承接办公问答、报表整理、制度查询、表格处理等通用能力，并统一路由到企业允许接入的模型后端。"
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
        <DataPill label="机器人总数" value={bots.length} />
        <DataPill label="公共机器人" value={publicCount} />
        <DataPill label="租户机器人" value={tenantCount} />
        <DataPill label="个人机器人" value={personalCount} />
      </div>

      <div className="grid gap-6 xl:grid-cols-[0.92fr_1.08fr]">
        <EnterprisePanel
          title="创建共享机器人"
          description="将模型入口、技能组合和角色描述封装为统一助手，供团队或全员使用。"
          action={(
            <div className="inline-flex items-center gap-2 rounded-full border border-emerald-500/15 bg-emerald-500/10 px-3 py-1 text-xs font-medium text-emerald-700 dark:text-emerald-300">
              <Share2 size={13} /> 可共享访问
            </div>
          )}
        >
          <div className="grid gap-4 md:grid-cols-2">
            <Input
              label="机器人名称"
              value={form.name || ''}
              onChange={(e) => setForm((prev) => ({ ...prev, name: e.target.value }))}
              placeholder="办公助手"
            />
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">作用范围</label>
              <Select value={form.scope || 'tenant'} onChange={(e) => setForm((prev) => ({ ...prev, scope: e.target.value }))}>
                <option value="personal">personal</option>
                <option value="team">team</option>
                <option value="tenant">tenant</option>
                <option value="public">public</option>
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
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">归属用户</label>
              <Select value={form.owner_user_id || ''} onChange={(e) => setForm((prev) => ({ ...prev, owner_user_id: e.target.value }))}>
                <option value="">不绑定个人用户</option>
                {users
                  .filter((user) => !form.tenant_id || user.tenant_id === form.tenant_id)
                  .map((user) => (
                    <option key={user.id} value={user.id}>{user.display_name}</option>
                  ))}
              </Select>
            </div>
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">模型供应商</label>
              <Select value={form.provider_name || ''} onChange={(e) => setForm((prev) => ({ ...prev, provider_name: e.target.value }))}>
                <option value="">使用平台默认值</option>
                {providers.map((provider) => (
                  <option key={provider.id} value={provider.name}>{provider.display_name || provider.name}</option>
                ))}
              </Select>
            </div>
            <Input
              label="模型名称"
              value={form.model_name || ''}
              onChange={(e) => setForm((prev) => ({ ...prev, model_name: e.target.value }))}
              placeholder="claude-sonnet-4"
            />
            <Input
              label="关联技能 ID"
              value={skillIDs}
              onChange={(e) => setSkillIDs(e.target.value)}
              placeholder="skill_table_ops, skill_daily_report"
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
            <Textarea
              label="机器人说明"
              value={form.description || ''}
              onChange={(e) => setForm((prev) => ({ ...prev, description: e.target.value }))}
              placeholder="处理日常办公问题、总结文件、辅助表格填写，并按技能选择合适的工作流。"
              rows={4}
            />
          </div>

          <div className="mt-4 rounded-2xl border border-slate-200/80 bg-slate-50/80 px-4 py-4 dark:border-white/[0.06] dark:bg-white/[0.03]">
            <p className="text-xs uppercase tracking-[0.2em] text-slate-400">推荐技能</p>
            <div className="mt-3 flex flex-wrap gap-2">
              {skills.slice(0, 10).map((skill) => (
                <button
                  key={skill.id}
                  type="button"
                  onClick={() => {
                    const next = new Set(parseCSV(skillIDs));
                    next.add(skill.id);
                    setSkillIDs([...next].join(', '));
                  }}
                  className="rounded-full border border-slate-200/80 px-3 py-1 text-xs text-slate-600 transition hover:border-emerald-500/30 hover:text-emerald-700 dark:border-white/[0.08] dark:text-slate-300"
                >
                  {skill.display_name || skill.name}
                </button>
              ))}
              {skills.length === 0 && (
                <p className="text-sm text-slate-500 dark:text-slate-400">当前还没有已发布技能，可以先去技能工坊创建。</p>
              )}
            </div>
          </div>

          <div className="mt-4 flex items-center justify-between gap-3">
            <p className="text-xs text-slate-500 dark:text-slate-400">
              共享机器人更适合单一职责设计：一个机器人解决一类问题，避免提示词和技能链过度膨胀。
            </p>
            <Button onClick={handleCreateBot} loading={saving}>
              <Plus size={15} /> 创建机器人
            </Button>
          </div>
        </EnterprisePanel>

        <EnterprisePanel
          title="机器人目录"
          description="查看企业内部当前已有的机器人、模型入口和技能挂载情况。"
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
                <option value="personal">personal</option>
                <option value="team">team</option>
                <option value="tenant">tenant</option>
                <option value="public">public</option>
              </Select>
            </div>
          </div>

          <TinyTable>
            <thead className="bg-slate-50/90 text-left text-xs uppercase tracking-[0.18em] text-slate-400 dark:bg-white/[0.03]">
              <tr>
                <th className="px-4 py-3">机器人</th>
                <th className="px-4 py-3">范围</th>
                <th className="px-4 py-3">模型入口</th>
                <th className="px-4 py-3">技能数</th>
                <th className="px-4 py-3">更新时间</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-200/80 dark:divide-white/[0.06]">
              {bots.length === 0 ? (
                <tr>
                  <td colSpan={5} className="px-4 py-10 text-center text-sm text-slate-500 dark:text-slate-400">
                    还没有配置机器人。
                  </td>
                </tr>
              ) : bots.map((bot) => {
                const tenant = tenants.find((item) => item.id === bot.tenant_id);
                return (
                  <tr key={bot.id}>
                    <td className="px-4 py-3">
                      <div className="flex items-start gap-3">
                        <div className="mt-0.5 rounded-xl bg-emerald-500/10 p-2 text-emerald-600 dark:text-emerald-300">
                          <Bot size={14} />
                        </div>
                        <div>
                          <p className="text-sm font-medium text-slate-900 dark:text-white">{bot.name}</p>
                          <p className="text-xs text-slate-400">{tenant?.name || bot.tenant_id || '全局共享'}</p>
                        </div>
                      </div>
                    </td>
                    <td className="px-4 py-3 text-sm"><Badge variant="outline">{bot.scope || 'tenant'}</Badge></td>
                    <td className="px-4 py-3 text-sm text-slate-500 dark:text-slate-400">
                      {bot.provider_name || 'default'}
                      {bot.model_name ? ` / ${bot.model_name}` : ''}
                    </td>
                    <td className="px-4 py-3 text-sm text-slate-500 dark:text-slate-400">{(bot.skill_ids || []).length}</td>
                    <td className="px-4 py-3 text-sm text-slate-500 dark:text-slate-400">{formatTime(bot.updated_at)}</td>
                  </tr>
                );
              })}
            </tbody>
          </TinyTable>

          <div className="mt-4 grid gap-3 md:grid-cols-2">
            <div className="rounded-2xl border border-slate-200/80 bg-slate-50/80 p-4 dark:border-white/[0.06] dark:bg-white/[0.03]">
              <p className="text-xs uppercase tracking-[0.2em] text-slate-400">设计建议</p>
              <p className="mt-2 text-sm leading-6 text-slate-700 dark:text-slate-300">
                企业共享机器人适合沉淀固定提示词、固定技能集合和固定输出口径，减少每个用户重复配置。
              </p>
            </div>
            <div className="rounded-2xl border border-slate-200/80 bg-slate-50/80 p-4 dark:border-white/[0.06] dark:bg-white/[0.03]">
              <p className="text-xs uppercase tracking-[0.2em] text-slate-400">典型场景</p>
              <p className="mt-2 text-sm leading-6 text-slate-700 dark:text-slate-300">
                <Sparkles size={14} className="mr-2 inline text-emerald-500" />
                日常问答、周报生成、表格处理、制度查询、值班应答都适合先做成共享机器人。
              </p>
            </div>
          </div>
        </EnterprisePanel>
      </div>
    </div>
  );
}
