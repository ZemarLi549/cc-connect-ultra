import { useCallback, useEffect, useState } from 'react';
import { AlertCircle, Bot, Database, FolderKanban, RefreshCcw, Save, ShieldCheck, Sparkles, Users2 } from 'lucide-react';
import { Badge, Button, Input } from '@/components/ui';
import {
  getEnterpriseLeaderboard,
  getEnterpriseOverview,
  getEnterpriseSettings,
  listEnterpriseBots,
  listEnterpriseImports,
  listEnterpriseProviders,
  saveEnterpriseSettings,
  type EnterpriseAIOpsSettings,
  type EnterpriseBot,
  type EnterpriseImportJob,
  type EnterpriseLeaderboardEntry,
  type EnterpriseOverview as EnterpriseOverviewData,
  type EnterpriseProvider,
} from '@/api/enterprise';
import { DataPill, EnterpriseHero, EnterprisePanel, TinyTable } from './shared';
import { formatTime } from './utils';

function microsToCurrency(value?: number) {
  return ((value || 0) / 1_000_000).toFixed(2);
}

export default function EnterpriseOverview() {
  const [overview, setOverview] = useState<EnterpriseOverviewData | null>(null);
  const [settings, setSettings] = useState<EnterpriseAIOpsSettings | null>(null);
  const [leaderboard, setLeaderboard] = useState<EnterpriseLeaderboardEntry[]>([]);
  const [providers, setProviders] = useState<EnterpriseProvider[]>([]);
  const [bots, setBots] = useState<EnterpriseBot[]>([]);
  const [imports, setImports] = useState<EnterpriseImportJob[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const fetchData = useCallback(async () => {
    try {
      setLoading(true);
      setError('');
      const [overviewData, settingsData, leaderboardData, providerData, botData, importData] = await Promise.all([
        getEnterpriseOverview(),
        getEnterpriseSettings(),
        getEnterpriseLeaderboard({ group_by: 'user', limit: '6' }),
        listEnterpriseProviders(),
        listEnterpriseBots(),
        listEnterpriseImports(),
      ]);
      setOverview(overviewData);
      setSettings(settingsData);
      setLeaderboard(leaderboardData.leaderboard || []);
      setProviders(providerData.providers || []);
      setBots(botData.bots || []);
      setImports(importData.imports || []);
    } catch (e: any) {
      setError(e.message || '加载企业总览失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  async function handleSaveSettings() {
    if (!settings) return;
    try {
      setSaving(true);
      setError('');
      const saved = await saveEnterpriseSettings(settings);
      setSettings(saved);
    } catch (e: any) {
      setError(e.message || '保存企业设置失败');
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="space-y-6 animate-fade-in">
      <EnterpriseHero
        eyebrow="企业控制台"
        title="玄星Ops 企业 AI 控制面"
        description="一套公共的 Claude Code 运行底座，面向企业内部统一提供空间隔离、共享机器人、技能资产、RBAC 权限和 PG/Redis 数据治理。"
        actions={(
          <>
            <Button variant="secondary" onClick={fetchData} loading={loading}>
              <RefreshCcw size={15} /> 刷新
            </Button>
            <Button onClick={handleSaveSettings} loading={saving}>
              <Save size={15} /> 保存设置
            </Button>
          </>
        )}
      />

      {error && (
        <div className="flex items-center gap-2 rounded-2xl border border-red-500/20 bg-red-500/10 px-4 py-3 text-sm text-red-600 dark:text-red-300">
          <AlertCircle size={16} /> {error}
        </div>
      )}

      <div className="grid gap-3 md:grid-cols-3 xl:grid-cols-6">
        <DataPill label="租户" value={overview?.tenants_count ?? 0} />
        <DataPill label="用户" value={overview?.users_count ?? 0} />
        <DataPill label="空间" value={overview?.spaces_count ?? 0} />
        <DataPill label="角色" value={overview?.roles_count ?? 0} />
        <DataPill label="项目档案" value={overview?.projects_count ?? 0} />
        <DataPill label="累计 Tokens" value={(overview?.total_tokens ?? 0).toLocaleString()} />
      </div>

      <div className="grid gap-6 xl:grid-cols-[1.1fr_0.9fr]">
        <EnterprisePanel
          title="企业底座设置"
          description="这里保存企业统一的存储与集成参数。当前控制面数据已经支持统一落 PostgreSQL，Redis 用于 RBAC 有效权限缓存。"
        >
          <div className="grid gap-4 md:grid-cols-2">
            <Input
              label="组织名称"
              value={settings?.organization_name || ''}
              onChange={(e) => setSettings((prev) => ({ ...(prev || {}), organization_name: e.target.value }))}
              placeholder="玄星Ops 企业平台"
            />
            <Input
              label="默认项目名"
              value={settings?.default_project_name || ''}
              onChange={(e) => setSettings((prev) => ({ ...(prev || {}), default_project_name: e.target.value }))}
              placeholder="enterprise-runtime"
            />
            <Input
              label="空间根目录"
              value={settings?.default_space_base_dir || ''}
              onChange={(e) => setSettings((prev) => ({ ...(prev || {}), default_space_base_dir: e.target.value }))}
              placeholder="D:\\tenant-data\\spaces"
            />
            <Input
              label="PostgreSQL DSN"
              value={settings?.postgres?.dsn || ''}
              onChange={(e) => setSettings((prev) => ({
                ...(prev || {}),
                postgres: { ...(prev?.postgres || {}), driver: 'postgres', dsn: e.target.value },
              }))}
              placeholder="postgres://user:pass@127.0.0.1:5432/cc_enterprise"
            />
            <Input
              label="Redis 地址"
              value={settings?.redis?.addr || ''}
              onChange={(e) => setSettings((prev) => ({
                ...(prev || {}),
                redis: { ...(prev?.redis || {}), addr: e.target.value },
              }))}
              placeholder="127.0.0.1:6379"
            />
            <Input
              label="Cocoloop 工作区"
              value={settings?.cocoloop?.workspace || ''}
              onChange={(e) => setSettings((prev) => ({
                ...(prev || {}),
                cocoloop: { ...(prev?.cocoloop || {}), workspace: e.target.value },
              }))}
              placeholder="enterprise-shared-skills"
            />
          </div>

          <div className="mt-4 flex flex-wrap gap-2 text-xs">
            <Badge variant={settings?.postgres?.dsn ? 'success' : 'outline'}>
              <Database size={12} className="mr-1 inline" />
              {settings?.postgres?.dsn ? 'PostgreSQL 已启用' : 'PostgreSQL 未配置'}
            </Badge>
            <Badge variant={settings?.redis?.addr ? 'success' : 'outline'}>
              <ShieldCheck size={12} className="mr-1 inline" />
              {settings?.redis?.addr ? 'Redis 已启用' : 'Redis 未配置'}
            </Badge>
            <Badge variant={settings?.cocoloop?.workspace ? 'info' : 'outline'}>
              <Sparkles size={12} className="mr-1 inline" />
              Cocoloop 技能导入
            </Badge>
          </div>
        </EnterprisePanel>

        <EnterprisePanel
          title="活跃度排行榜"
          description="基于使用记录看当前最活跃的用户，方便预算、运营和资源分配。"
        >
          <div className="space-y-3">
            {leaderboard.length === 0 ? (
              <p className="text-sm text-slate-500 dark:text-slate-400">还没有使用记录。</p>
            ) : leaderboard.map((entry, index) => (
              <div
                key={`${entry.subject_id}-${index}`}
                className="flex items-center justify-between rounded-2xl border border-slate-200/70 bg-slate-50/80 px-4 py-3 dark:border-white/[0.06] dark:bg-white/[0.03]"
              >
                <div>
                  <p className="text-sm font-medium text-slate-900 dark:text-white">{entry.subject_name || entry.subject_id}</p>
                  <p className="text-xs uppercase tracking-[0.18em] text-slate-400">{entry.subject_type}</p>
                </div>
                <div className="text-right">
                  <p className="text-sm font-semibold text-emerald-600 dark:text-emerald-300">{entry.tokens.toLocaleString()} tok</p>
                  <p className="text-xs text-slate-400">{entry.requests} 次请求</p>
                </div>
              </div>
            ))}
          </div>
        </EnterprisePanel>
      </div>

      <div className="grid gap-6 xl:grid-cols-[1fr_1fr]">
        <EnterprisePanel title="共享机器人" description="适合日常办公、报表、值班、知识问答等统一能力入口。">
          <div className="space-y-3">
            {bots.length === 0 ? (
              <p className="text-sm text-slate-500 dark:text-slate-400">还没有配置共享机器人。</p>
            ) : bots.slice(0, 6).map((bot) => (
              <div key={bot.id} className="flex items-start justify-between rounded-2xl border border-slate-200/70 px-4 py-3 dark:border-white/[0.06]">
                <div className="space-y-1">
                  <p className="text-sm font-semibold text-slate-900 dark:text-white">
                    <Bot size={14} className="mr-2 inline text-emerald-500" />
                    {bot.name}
                  </p>
                  <p className="text-xs text-slate-500 dark:text-slate-400">{bot.description || '面向企业内部共享的助手能力。'}</p>
                </div>
                <div className="text-right text-xs text-slate-400">
                  <div>{bot.provider_name || 'claude-code'}</div>
                  <div>{bot.model_name || '默认模型'}</div>
                </div>
              </div>
            ))}
          </div>
        </EnterprisePanel>

        <EnterprisePanel title="模型供应目录" description="统一维护企业允许接入的模型与推理后端。">
          <div className="space-y-3">
            {providers.length === 0 ? (
              <p className="text-sm text-slate-500 dark:text-slate-400">还没有配置模型供应商。</p>
            ) : providers.slice(0, 6).map((provider) => (
              <div key={provider.id} className="flex items-center justify-between rounded-2xl border border-slate-200/70 px-4 py-3 dark:border-white/[0.06]">
                <div>
                  <p className="text-sm font-semibold text-slate-900 dark:text-white">{provider.display_name || provider.name}</p>
                  <p className="text-xs text-slate-500 dark:text-slate-400">{provider.provider_type || 'OpenAI 兼容接口'}</p>
                </div>
                <Badge variant="outline">{provider.default_model || provider.models?.[0] || '默认模型'}</Badge>
              </div>
            ))}
          </div>
        </EnterprisePanel>
      </div>

      <EnterprisePanel title="最近导入任务" description="追踪 Cocoloop 或其他来源的技能导入任务。">
        <TinyTable>
          <thead className="bg-slate-50/90 text-left text-xs uppercase tracking-[0.18em] text-slate-400 dark:bg-white/[0.03]">
            <tr>
              <th className="px-4 py-3">来源</th>
              <th className="px-4 py-3">类型</th>
              <th className="px-4 py-3">状态</th>
              <th className="px-4 py-3">导入技能数</th>
              <th className="px-4 py-3">更新时间</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-200/80 dark:divide-white/[0.06]">
            {imports.length === 0 ? (
              <tr>
                <td colSpan={5} className="px-4 py-8 text-center text-sm text-slate-500 dark:text-slate-400">
                  暂无导入任务。
                </td>
              </tr>
            ) : imports.slice(0, 8).map((item) => (
              <tr key={item.id}>
                <td className="px-4 py-3 text-sm font-medium text-slate-900 dark:text-white">{item.source_name || item.source_ref || item.id}</td>
                <td className="px-4 py-3 text-sm text-slate-500 dark:text-slate-400">{item.source_type || 'manual'}</td>
                <td className="px-4 py-3 text-sm text-slate-500 dark:text-slate-400">{item.status || 'queued'}</td>
                <td className="px-4 py-3 text-sm text-slate-500 dark:text-slate-400">{item.imported_skills ?? 0}</td>
                <td className="px-4 py-3 text-sm text-slate-500 dark:text-slate-400">{formatTime(item.updated_at)}</td>
              </tr>
            ))}
          </tbody>
        </TinyTable>
      </EnterprisePanel>

      <div className="grid gap-6 xl:grid-cols-[1fr_1fr]">
        <EnterprisePanel title="平台状态" description="从控制面视角快速判断企业空间与资产规模。">
          <div className="grid gap-3 md:grid-cols-2">
            <div className="rounded-2xl border border-slate-200/80 bg-slate-50/80 p-4 dark:border-white/[0.06] dark:bg-white/[0.03]">
              <p className="text-xs uppercase tracking-[0.18em] text-slate-400">空间与用户</p>
              <p className="mt-2 text-sm leading-6 text-slate-700 dark:text-slate-300">
                当前共 {overview?.spaces_count ?? 0} 个空间、{overview?.users_count ?? 0} 个用户，适合按“每人一个空间”方式进行运行时隔离。
              </p>
            </div>
            <div className="rounded-2xl border border-slate-200/80 bg-slate-50/80 p-4 dark:border-white/[0.06] dark:bg-white/[0.03]">
              <p className="text-xs uppercase tracking-[0.18em] text-slate-400">技能与机器人</p>
              <p className="mt-2 text-sm leading-6 text-slate-700 dark:text-slate-300">
                当前沉淀 {overview?.skills_count ?? 0} 个技能、{overview?.bots_count ?? 0} 个机器人，便于做公共办公入口与复用资产。
              </p>
            </div>
          </div>
        </EnterprisePanel>

        <EnterprisePanel title="成本概览" description="费用先按请求记录累计，适合后续扩展配额、预算和排行。">
          <div className="grid gap-3 md:grid-cols-2">
            <div className="rounded-2xl border border-slate-200/80 bg-slate-50/80 p-4 dark:border-white/[0.06] dark:bg-white/[0.03]">
              <p className="text-xs uppercase tracking-[0.18em] text-slate-400">总成本</p>
              <p className="mt-2 text-2xl font-semibold text-slate-950 dark:text-white">
                ${microsToCurrency(overview?.total_cost_micros)}
              </p>
            </div>
            <div className="rounded-2xl border border-slate-200/80 bg-slate-50/80 p-4 dark:border-white/[0.06] dark:bg-white/[0.03]">
              <p className="text-xs uppercase tracking-[0.18em] text-slate-400">请求记录</p>
              <p className="mt-2 text-2xl font-semibold text-slate-950 dark:text-white">
                {overview?.usage_count ?? 0}
              </p>
            </div>
          </div>
        </EnterprisePanel>
      </div>

      <div className="grid gap-3 md:grid-cols-4">
        <div className="rounded-2xl border border-slate-200/80 bg-slate-50/80 p-4 dark:border-white/[0.06] dark:bg-white/[0.03]">
          <p className="text-xs uppercase tracking-[0.2em] text-slate-400">空间隔离</p>
          <p className="mt-2 text-sm leading-6 text-slate-700 dark:text-slate-300">目录、模型入口、会话上下文都应按空间隔离，避免共享上下文串话。</p>
        </div>
        <div className="rounded-2xl border border-slate-200/80 bg-slate-50/80 p-4 dark:border-white/[0.06] dark:bg-white/[0.03]">
          <p className="text-xs uppercase tracking-[0.2em] text-slate-400">统一资产</p>
          <p className="mt-2 text-sm leading-6 text-slate-700 dark:text-slate-300">技能、机器人、模型供应商、项目档案适合统一沉淀到 PG 中进行治理。</p>
        </div>
        <div className="rounded-2xl border border-slate-200/80 bg-slate-50/80 p-4 dark:border-white/[0.06] dark:bg-white/[0.03]">
          <p className="text-xs uppercase tracking-[0.2em] text-slate-400">RBAC</p>
          <p className="mt-2 text-sm leading-6 text-slate-700 dark:text-slate-300">角色与绑定支持全局、租户、空间、项目四级作用域，便于精细化授权。</p>
        </div>
        <div className="rounded-2xl border border-slate-200/80 bg-slate-50/80 p-4 dark:border-white/[0.06] dark:bg-white/[0.03]">
          <p className="text-xs uppercase tracking-[0.2em] text-slate-400">运行态</p>
          <p className="mt-2 text-sm leading-6 text-slate-700 dark:text-slate-300">控制面已支持统一存储，当前运行态项目仍然会从本地配置启动并同步进控制面。</p>
        </div>
      </div>
    </div>
  );
}
