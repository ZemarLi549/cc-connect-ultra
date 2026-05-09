import { useCallback, useEffect, useState } from 'react';
import { AlertCircle, BarChart3, DatabaseZap, RefreshCcw, Trophy } from 'lucide-react';
import { Badge, Button } from '@/components/ui';
import {
  getEnterpriseLeaderboard,
  getEnterpriseOverview,
  listEnterpriseProviders,
  listEnterpriseUsage,
  type EnterpriseLeaderboardEntry,
  type EnterpriseOverview,
  type EnterpriseProvider,
  type EnterpriseUsageRecord,
} from '@/api/enterprise';
import { DataPill, EnterpriseHero, EnterprisePanel, Select, TinyTable } from './shared';
import { formatTime } from './utils';

function microsToCurrency(value: number) {
  return (value / 1_000_000).toFixed(2);
}

export default function EnterpriseAnalytics() {
  const [overview, setOverview] = useState<EnterpriseOverview | null>(null);
  const [providers, setProviders] = useState<EnterpriseProvider[]>([]);
  const [usage, setUsage] = useState<EnterpriseUsageRecord[]>([]);
  const [leaderboard, setLeaderboard] = useState<EnterpriseLeaderboardEntry[]>([]);
  const [providerFilter, setProviderFilter] = useState('');
  const [groupBy, setGroupBy] = useState('user');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const fetchData = useCallback(async () => {
    try {
      setLoading(true);
      setError('');
      const [overviewData, providerData, usageData, leaderboardData] = await Promise.all([
        getEnterpriseOverview(),
        listEnterpriseProviders(),
        listEnterpriseUsage({ provider: providerFilter || undefined, limit: '100' }),
        getEnterpriseLeaderboard({ group_by: groupBy, limit: '10' }),
      ]);
      setOverview(overviewData);
      setProviders(providerData.providers || []);
      setUsage(usageData.usage || []);
      setLeaderboard(leaderboardData.leaderboard || []);
    } catch (e: any) {
      setError(e.message || '加载统计分析失败');
    } finally {
      setLoading(false);
    }
  }, [groupBy, providerFilter]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const totalPromptTokens = usage.reduce((sum, item) => sum + (item.prompt_tokens || 0), 0);
  const totalCompletionTokens = usage.reduce((sum, item) => sum + (item.completion_tokens || 0), 0);
  const totalLatency = usage.reduce((sum, item) => sum + (item.latency_ms || 0), 0);
  const averageLatency = usage.length > 0 ? Math.round(totalLatency / usage.length) : 0;

  const providerBreakdown = providers
    .map((provider) => {
      const records = usage.filter((item) => item.provider_name === provider.name);
      return {
        name: provider.display_name || provider.name,
        requests: records.length,
        tokens: records.reduce((sum, item) => sum + (item.total_tokens || 0), 0),
      };
    })
    .filter((item) => item.requests > 0)
    .sort((a, b) => b.tokens - a.tokens);

  const modelBreakdown = Object.entries(
    usage.reduce<Record<string, { requests: number; tokens: number }>>((acc, item) => {
      const key = item.model_name || 'default';
      if (!acc[key]) {
        acc[key] = { requests: 0, tokens: 0 };
      }
      acc[key].requests += 1;
      acc[key].tokens += item.total_tokens || 0;
      return acc;
    }, {}),
  )
    .map(([name, stats]) => ({ name, ...stats }))
    .sort((a, b) => b.tokens - a.tokens);

  return (
    <div className="space-y-6 animate-fade-in">
      <EnterpriseHero
        eyebrow="统计分析"
        title="从用量、成本和排行看企业 AI 运行状态"
        description="使用记录已经纳入企业控制面，可按用户、空间、租户维度观察请求量、Token、成本和模型分布。"
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

      <div className="grid gap-3 md:grid-cols-5">
        <DataPill label="请求事件" value={overview?.usage_count ?? 0} />
        <DataPill label="总 Tokens" value={(overview?.total_tokens ?? 0).toLocaleString()} />
        <DataPill label="输入 Tokens" value={totalPromptTokens.toLocaleString()} />
        <DataPill label="输出 Tokens" value={totalCompletionTokens.toLocaleString()} />
        <DataPill label="平均延迟" value={`${averageLatency} ms`} />
      </div>

      <div className="grid gap-6 xl:grid-cols-[0.88fr_1.12fr]">
        <EnterprisePanel
          title="筛选与排行榜"
          description="按供应商过滤记录，并在用户、空间、租户三种维度切换排行。"
          action={(
            <div className="inline-flex items-center gap-2 rounded-full border border-emerald-500/15 bg-emerald-500/10 px-3 py-1 text-xs font-medium text-emerald-700 dark:text-emerald-300">
              <Trophy size={13} /> 实时排行
            </div>
          )}
        >
          <div className="grid gap-3 md:grid-cols-2">
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">供应商筛选</label>
              <Select value={providerFilter} onChange={(e) => setProviderFilter(e.target.value)}>
                <option value="">全部供应商</option>
                {providers.map((provider) => (
                  <option key={provider.id} value={provider.name}>{provider.display_name || provider.name}</option>
                ))}
              </Select>
            </div>
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">排行维度</label>
              <Select value={groupBy} onChange={(e) => setGroupBy(e.target.value)}>
                <option value="user">user</option>
                <option value="space">space</option>
                <option value="tenant">tenant</option>
              </Select>
            </div>
          </div>

          <div className="mt-4 space-y-3">
            {leaderboard.length === 0 ? (
              <p className="text-sm text-slate-500 dark:text-slate-400">暂无用量记录。</p>
            ) : leaderboard.map((entry) => (
              <div key={`${entry.subject_type}-${entry.subject_id}`} className="flex items-center justify-between rounded-2xl border border-slate-200/80 bg-slate-50/80 px-4 py-3 dark:border-white/[0.06] dark:bg-white/[0.03]">
                <div>
                  <p className="text-sm font-medium text-slate-900 dark:text-white">{entry.subject_name || entry.subject_id}</p>
                  <p className="text-xs uppercase tracking-[0.18em] text-slate-400">{entry.subject_type}</p>
                </div>
                <div className="text-right">
                  <p className="text-sm font-semibold text-emerald-600 dark:text-emerald-300">{entry.tokens.toLocaleString()} tok</p>
                  <p className="text-xs text-slate-400">{entry.requests} 次 / ${microsToCurrency(entry.cost_micros)}</p>
                </div>
              </div>
            ))}
          </div>
        </EnterprisePanel>

        <EnterprisePanel
          title="供应商与模型分布"
          description="观察请求在不同推理后端上的分布，为配额、路由和成本治理提供依据。"
        >
          <div className="grid gap-4 md:grid-cols-2">
            <div className="rounded-2xl border border-slate-200/80 bg-slate-50/80 p-4 dark:border-white/[0.06] dark:bg-white/[0.03]">
              <div className="mb-3 flex items-center gap-2 text-xs uppercase tracking-[0.18em] text-slate-400">
                <DatabaseZap size={13} /> 供应商分布
              </div>
              <div className="space-y-3">
                {providerBreakdown.length === 0 ? (
                  <p className="text-sm text-slate-500 dark:text-slate-400">暂无供应商调用记录。</p>
                ) : providerBreakdown.map((item) => (
                  <div key={item.name} className="flex items-center justify-between">
                    <div className="min-w-0">
                      <p className="truncate text-sm font-medium text-slate-900 dark:text-white">{item.name}</p>
                      <p className="text-xs text-slate-400">{item.requests} 次请求</p>
                    </div>
                    <p className="text-sm text-emerald-600 dark:text-emerald-300">{item.tokens.toLocaleString()} tok</p>
                  </div>
                ))}
              </div>
            </div>

            <div className="rounded-2xl border border-slate-200/80 bg-slate-50/80 p-4 dark:border-white/[0.06] dark:bg-white/[0.03]">
              <div className="mb-3 flex items-center gap-2 text-xs uppercase tracking-[0.18em] text-slate-400">
                <BarChart3 size={13} /> 模型分布
              </div>
              <div className="space-y-3">
                {modelBreakdown.length === 0 ? (
                  <p className="text-sm text-slate-500 dark:text-slate-400">暂无模型调用记录。</p>
                ) : modelBreakdown.slice(0, 8).map((item) => (
                  <div key={item.name} className="flex items-center justify-between">
                    <div className="min-w-0">
                      <p className="truncate text-sm font-medium text-slate-900 dark:text-white">{item.name}</p>
                      <p className="text-xs text-slate-400">{item.requests} 次请求</p>
                    </div>
                    <p className="text-sm text-emerald-600 dark:text-emerald-300">{item.tokens.toLocaleString()} tok</p>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </EnterprisePanel>
      </div>

      <EnterprisePanel
        title="最近请求记录"
        description="展示最新的使用事件，适合做预算观察、模型排障和调用追踪。"
      >
        <TinyTable>
          <thead className="bg-slate-50/90 text-left text-xs uppercase tracking-[0.18em] text-slate-400 dark:bg-white/[0.03]">
            <tr>
              <th className="px-4 py-3">时间</th>
              <th className="px-4 py-3">供应商 / 模型</th>
              <th className="px-4 py-3">请求类型</th>
              <th className="px-4 py-3">Tokens</th>
              <th className="px-4 py-3">成本</th>
              <th className="px-4 py-3">延迟</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-200/80 dark:divide-white/[0.06]">
            {usage.length === 0 ? (
              <tr>
                <td colSpan={6} className="px-4 py-10 text-center text-sm text-slate-500 dark:text-slate-400">
                  暂无请求记录。
                </td>
              </tr>
            ) : usage.map((item) => (
              <tr key={item.id}>
                <td className="px-4 py-3 text-sm text-slate-500 dark:text-slate-400">{formatTime(item.occurred_at)}</td>
                <td className="px-4 py-3 text-sm text-slate-500 dark:text-slate-400">
                  {item.provider_name || 'default'}
                  {item.model_name ? ` / ${item.model_name}` : ''}
                </td>
                <td className="px-4 py-3 text-sm"><Badge variant="outline">{item.request_kind || 'chat'}</Badge></td>
                <td className="px-4 py-3 text-sm text-slate-500 dark:text-slate-400">{(item.total_tokens || 0).toLocaleString()}</td>
                <td className="px-4 py-3 text-sm text-slate-500 dark:text-slate-400">${microsToCurrency(item.cost_micros || 0)}</td>
                <td className="px-4 py-3 text-sm text-slate-500 dark:text-slate-400">{item.latency_ms || 0} ms</td>
              </tr>
            ))}
          </tbody>
        </TinyTable>
      </EnterprisePanel>
    </div>
  );
}
