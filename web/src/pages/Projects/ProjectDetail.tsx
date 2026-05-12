import { useEffect, useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { useParams, Link, useNavigate } from 'react-router-dom';
import {
  ArrowLeft, Plug, Heart, Settings, Layers, Zap, Pause, Play,
  Trash2, Plus, Check, Clock, ExternalLink, Link2,
} from 'lucide-react';
import { Card, Badge, Button, Input, Modal, EmptyState } from '@/components/ui';
import { getProject, updateProject, deleteProject, listAgentTypes, lookupFeishuIds, type ProjectDetail as ProjectDetailType } from '@/api/projects';
import { listProviders, addProvider, removeProvider, activateProvider, type Provider, listGlobalProviders, type GlobalProvider, saveProviderRefs } from '@/api/providers';
import { getHeartbeat, pauseHeartbeat, resumeHeartbeat, triggerHeartbeat, setHeartbeatInterval, type HeartbeatStatus } from '@/api/heartbeat';
import { restartSystem } from '@/api/status';
import { formatTime, cn } from '@/lib/utils';
import PlatformSetupQR from './PlatformSetupQR';
import PlatformManualForm from './PlatformManualForm';
import { platformMeta, type FieldDef } from '@/lib/platformMeta';

const PLATFORM_OPTIONS: { key: string; label: string; color: string; abbr: string; qr?: boolean }[] = [
  { key: 'feishu', label: 'Feishu / Lark', abbr: 'FS', color: 'bg-blue-50 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400', qr: true },
  { key: 'weixin', label: 'WeChat', abbr: 'WX', color: 'bg-green-50 dark:bg-green-900/30 text-green-600 dark:text-green-400', qr: true },
  { key: 'telegram', label: 'Telegram', abbr: 'TG', color: 'bg-sky-50 dark:bg-sky-900/30 text-sky-600 dark:text-sky-400' },
  { key: 'discord', label: 'Discord', abbr: 'DC', color: 'bg-indigo-50 dark:bg-indigo-900/30 text-indigo-600 dark:text-indigo-400' },
  { key: 'slack', label: 'Slack', abbr: 'SK', color: 'bg-purple-50 dark:bg-purple-900/30 text-purple-600 dark:text-purple-400' },
  { key: 'dingtalk', label: 'DingTalk', abbr: 'DT', color: 'bg-orange-50 dark:bg-orange-900/30 text-orange-600 dark:text-orange-400' },
  { key: 'wecom', label: 'WeChat Work', abbr: 'WC', color: 'bg-emerald-50 dark:bg-emerald-900/30 text-emerald-600 dark:text-emerald-400' },
  { key: 'qq', label: 'QQ (OneBot)', abbr: 'QQ', color: 'bg-cyan-50 dark:bg-cyan-900/30 text-cyan-600 dark:text-cyan-400' },
  { key: 'qqbot', label: 'QQ Bot (Official)', abbr: 'QB', color: 'bg-cyan-50 dark:bg-cyan-900/30 text-cyan-600 dark:text-cyan-400' },
  { key: 'line', label: 'LINE', abbr: 'LN', color: 'bg-lime-50 dark:bg-lime-900/30 text-lime-600 dark:text-lime-400' },
  { key: 'weibo', label: 'Weibo (微博)', abbr: 'WB', color: 'bg-red-50 dark:bg-red-900/30 text-red-600 dark:text-red-400' },
];

const supportsQRPlatform = (type: string) => type === 'feishu' || type === 'lark' || type === 'weixin';
const supportsManualPlatform = (type: string) => !!platformMeta[type];
const hasBothPlatformModes = (type: string) => supportsQRPlatform(type) && supportsManualPlatform(type);

type Tab = 'overview' | 'providers' | 'heartbeat' | 'settings';
type OptionValueType = 'string' | 'number' | 'boolean' | 'json';
type OptionRow = {
  id: string;
  key: string;
  type: OptionValueType;
  value: string;
};
type PlatformOptionEditor = {
  index: number;
  name: string;
  type: string;
  rows: OptionRow[];
};
type DifyInputRow = {
  id: string;
  key: string;
  value: string;
};

const makeEntryId = () => `${Date.now()}-${Math.random().toString(16).slice(2, 8)}`;
const DIFY_RESERVED_OPTION_KEYS = new Set(['base_url', 'api_key', 'app_mode', 'user', 'query_input_key', 'inputs']);
const platformControlKey = (platformType: string, platformIndex: number) => `${platformType}#${platformIndex}`;

function inferOptionType(value: any): OptionValueType {
  if (typeof value === 'number') return 'number';
  if (typeof value === 'boolean') return 'boolean';
  if (value !== null && typeof value === 'object') return 'json';
  return 'string';
}

function optionMapToRows(source?: Record<string, any>): OptionRow[] {
  if (!source) return [];
  return Object.entries(source).map(([key, value]) => {
    const typ = inferOptionType(value);
    return {
      id: makeEntryId(),
      key,
      type: typ,
      value: typ === 'json' ? JSON.stringify(value) : String(value),
    };
  });
}

function fieldTypeToOptionType(fieldType?: FieldDef['type']): OptionValueType {
  if (fieldType === 'number') return 'number';
  if (fieldType === 'boolean') return 'boolean';
  return 'string';
}

function findPlatformField(platformType: string, key: string): FieldDef | undefined {
  return platformMeta[platformType]?.fields.find(f => f.key === key);
}

function mergeTemplateRows(platformType: string, rows: OptionRow[]): OptionRow[] {
  const meta = platformMeta[platformType];
  if (!meta) return rows;
  const byKey = new Map(rows.map(r => [r.key, r]));
  const merged: OptionRow[] = [];
  for (const field of meta.fields) {
    const existing = byKey.get(field.key);
    if (existing) {
      merged.push(existing);
      byKey.delete(field.key);
      continue;
    }
    merged.push({
      id: makeEntryId(),
      key: field.key,
      type: fieldTypeToOptionType(field.type),
      value: '',
    });
  }
  for (const row of rows) {
    if (byKey.has(row.key)) {
      merged.push(row);
      byKey.delete(row.key);
    }
  }
  return merged;
}

function sortRowsByTemplate(platformType: string, rows: OptionRow[]): OptionRow[] {
  const meta = platformMeta[platformType];
  if (!meta) return rows;
  const order = new Map(meta.fields.map((f, i) => [f.key, i]));
  return [...rows].sort((a, b) => {
    const ai = order.has(a.key) ? order.get(a.key)! : Number.MAX_SAFE_INTEGER;
    const bi = order.has(b.key) ? order.get(b.key)! : Number.MAX_SAFE_INTEGER;
    if (ai !== bi) return ai - bi;
    return a.key.localeCompare(b.key);
  });
}

function rowsToOptionMap(rows: OptionRow[]): Record<string, any> {
  const out: Record<string, any> = {};
  for (const row of rows) {
    const key = row.key.trim();
    if (!key) continue;
    const raw = row.value.trim();
    if (row.type === 'string' && raw === '') continue;
    if (row.type === 'number') {
      if (raw === '') continue;
      const n = Number(raw);
      if (!Number.isNaN(n)) out[key] = n;
      continue;
    }
    if (row.type === 'boolean') {
      if (raw === '') continue;
      out[key] = raw.toLowerCase() === 'true';
      continue;
    }
    if (row.type === 'json') {
      if (!raw) continue;
      try {
        out[key] = JSON.parse(raw);
      } catch {
        out[key] = raw;
      }
      continue;
    }
    out[key] = row.value;
  }
  return out;
}

function ensureDifyOptionRows(rows: OptionRow[], projectName?: string): OptionRow[] {
  const next = [...rows];
  const findIndexByKey = (key: string) => next.findIndex(r => r.key.trim() === key);
  const ensure = (key: string, type: OptionValueType, defaultValue: string) => {
    const idx = findIndexByKey(key);
    if (idx >= 0) {
      const current = next[idx];
      next[idx] = { ...current, key, type };
      return;
    }
    next.push({ id: makeEntryId(), key, type, value: defaultValue });
  };
  ensure('base_url', 'string', 'https://api.dify.ai/v1');
  ensure('api_key', 'string', '');
  ensure('app_mode', 'string', 'advanced-chat');
  ensure('user', 'string', `cc-connect:${projectName || 'project'}`);
  ensure('query_input_key', 'string', 'query');
  ensure('inputs', 'json', '{}');
  return next;
}

function parseDifyInputs(raw: string): Record<string, string> {
  const trimmed = raw.trim();
  if (!trimmed) return {};
  try {
    const parsed = JSON.parse(trimmed);
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return {};
    }
    const out: Record<string, string> = {};
    for (const [k, v] of Object.entries(parsed as Record<string, any>)) {
      const key = String(k).trim();
      if (!key) continue;
      out[key] = typeof v === 'string' ? v : JSON.stringify(v);
    }
    return out;
  } catch {
    return {};
  }
}

function buildDifyInputRows(rows: OptionRow[]): DifyInputRow[] {
  const inputsRow = rows.find(r => r.key.trim() === 'inputs');
  if (!inputsRow) {
    return [{ id: makeEntryId(), key: '', value: '' }];
  }
  const map = parseDifyInputs(inputsRow.value);
  const entries = Object.entries(map);
  if (entries.length === 0) {
    return [{ id: makeEntryId(), key: '', value: '' }];
  }
  return entries.map(([key, value]) => ({ id: makeEntryId(), key, value }));
}

function difyInputsToMap(rows: DifyInputRow[]): Record<string, string> {
  const out: Record<string, string> = {};
  for (const row of rows) {
    const key = row.key.trim();
    if (!key) continue;
    out[key] = row.value;
  }
  return out;
}

function normalizeJSONValue(value: any): any {
  if (Array.isArray(value)) {
    return value.map(normalizeJSONValue);
  }
  if (value && typeof value === 'object') {
    const sortedKeys = Object.keys(value).sort();
    const out: Record<string, any> = {};
    for (const key of sortedKeys) {
      out[key] = normalizeJSONValue(value[key]);
    }
    return out;
  }
  return value;
}

function jsonEqual(a: any, b: any): boolean {
  return JSON.stringify(normalizeJSONValue(a)) === JSON.stringify(normalizeJSONValue(b));
}

export default function ProjectDetail() {
  const { t } = useTranslation();
  const { name } = useParams<{ name: string }>();
  const [tab, setTab] = useState<Tab>('overview');
  const [project, setProject] = useState<ProjectDetailType | null>(null);
  const [providers, setProviders] = useState<Provider[]>([]);
  const [activeProvider, setActiveProvider] = useState('');
  const [heartbeat, setHeartbeatState] = useState<HeartbeatStatus | null>(null);
  const [loading, setLoading] = useState(true);

  // Settings form
  const [language, setLanguage] = useState('');
  const [adminFrom, setAdminFrom] = useState('');
  const [disabledCmds, setDisabledCmds] = useState('');
  const [workDir, setWorkDir] = useState('');
  const [agentMode, setAgentMode] = useState('');
  const [showCtxIndicator, setShowCtxIndicator] = useState(true);
  const [replyFooter, setReplyFooter] = useState(true);
  const [injectSender, setInjectSender] = useState(false);
  const [platformAllowFrom, setPlatformAllowFrom] = useState<Record<string, string>>({});
  const [platformAllowChat, setPlatformAllowChat] = useState<Record<string, string>>({});
  const [agentOptionRows, setAgentOptionRows] = useState<OptionRow[]>([]);
  const [difyInputs, setDifyInputs] = useState<DifyInputRow[]>([]);
  const [platformOptionEditors, setPlatformOptionEditors] = useState<PlatformOptionEditor[]>([]);
  const [removedPlatformIndexes, setRemovedPlatformIndexes] = useState<number[]>([]);
  const [saving, setSaving] = useState(false);
  const [lookupQuery, setLookupQuery] = useState('');
  const [lookupLoading, setLookupLoading] = useState(false);
  const [lookupUsers, setLookupUsers] = useState<Array<{ open_id: string; name?: string; en_name?: string; nickname?: string }>>([]);
  const [lookupChats, setLookupChats] = useState<Array<{ chat_id: string; name?: string }>>([]);
  const [lookupWarnings, setLookupWarnings] = useState<string[]>([]);

  // Agent type
  const [agentTypes, setAgentTypes] = useState<string[]>([]);
  const [selectedAgentType, setSelectedAgentType] = useState('');
  const isDifyAgent = selectedAgentType.trim().toLowerCase() === 'dify';

  // Global providers & refs
  const [globalProviders, setGlobalProviders] = useState<GlobalProvider[]>([]);
  const [providerRefs, setProviderRefs] = useState<string[]>([]);
  const [savingRefs, setSavingRefs] = useState(false);

  // Add provider modal
  const [showAddProvider, setShowAddProvider] = useState(false);
  const [addMode, setAddMode] = useState<'pick' | 'custom'>('pick');
  const [newProvider, setNewProvider] = useState({ name: '', api_key: '', base_url: '', model: '' });

  // Interval modal
  const [showInterval, setShowInterval] = useState(false);
  const [newInterval, setNewInterval] = useState('30');

  // Add platform
  const [showAddPlatform, setShowAddPlatform] = useState(false);
  const [addPlatType, setAddPlatType] = useState('');
  const [addPlatMode, setAddPlatMode] = useState<'qr' | 'manual' | ''>('');
  const [showRestartModal, setShowRestartModal] = useState(false);

  // Delete project
  const navigate = useNavigate();
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const handleDeleteProject = async () => {
    if (!name) return;
    setDeleting(true);
    try {
      const res = await deleteProject(name);
      setShowDeleteConfirm(false);
      if (res.restart_required && window.confirm(t('setup.restartAfterDelete'))) {
        await restartSystem();
        // Wait for service to come back up before navigating
        await waitForService(8000);
      }
      navigate('/projects');
    } catch (e: any) {
      alert(e?.message || String(e));
    } finally {
      setDeleting(false);
    }
  };

  const waitForService = (maxMs: number) =>
    new Promise<void>((resolve) => {
      const start = Date.now();
      const poll = () => {
        fetch('/api/v1/status')
          .then((r) => { if (r.ok) resolve(); else throw new Error(); })
          .catch(() => {
            if (Date.now() - start > maxMs) { resolve(); return; }
            setTimeout(poll, 500);
          });
      };
      setTimeout(poll, 1500);
    });

  const fetchAll = useCallback(async () => {
    if (!name) return;
    try {
      setLoading(true);
      const [proj, provs, hb, gp, at] = await Promise.allSettled([
        getProject(name),
        listProviders(name),
        getHeartbeat(name),
        listGlobalProviders(),
        listAgentTypes(),
      ]);
      if (proj.status === 'fulfilled') {
        setProject(proj.value);
        setLanguage(proj.value.settings?.language || '');
        setAdminFrom(proj.value.settings?.admin_from || '');
        setDisabledCmds(proj.value.settings?.disabled_commands?.join(', ') || '');
        setWorkDir(proj.value.work_dir || '');
        setAgentMode(proj.value.agent_mode || 'default');
        setSelectedAgentType(proj.value.agent_type || '');
        setShowCtxIndicator(proj.value.show_context_indicator !== false);
        setReplyFooter(proj.value.reply_footer !== false);
        setInjectSender(proj.value.inject_sender === true);
        setProviderRefs(proj.value.provider_refs || []);
        const agentRows = optionMapToRows(proj.value.agent_options || {});
        const normalizedAgentRows = (proj.value.agent_type || '').toLowerCase() === 'dify'
          ? ensureDifyOptionRows(agentRows, proj.value.name || name)
          : agentRows;
        setAgentOptionRows(normalizedAgentRows);
        setDifyInputs(buildDifyInputRows(normalizedAgentRows));
        setPlatformOptionEditors((proj.value.platform_configs || []).map(pc => ({
          index: pc.index ?? 0,
          name: pc.name || '',
          type: pc.type,
          rows: mergeTemplateRows(pc.type, optionMapToRows(pc.options || {})),
        })));
        setRemovedPlatformIndexes([]);
        const afMap: Record<string, string> = {};
        const acMap: Record<string, string> = {};
        proj.value.platform_configs?.forEach(pc => {
          const key = platformControlKey(pc.type, pc.index ?? 0);
          if (pc.allow_from !== undefined) afMap[key] = pc.allow_from;
          if (pc.options && typeof pc.options.allow_chat === 'string') {
            acMap[key] = pc.options.allow_chat;
          }
        });
        setPlatformAllowFrom(afMap);
        setPlatformAllowChat(acMap);
      }
      if (provs.status === 'fulfilled') {
        setProviders(provs.value.providers || []);
        setActiveProvider(provs.value.active_provider || '');
      }
      if (hb.status === 'fulfilled') {
        const hbVal = hb.value;
        setHeartbeatState(hbVal?.enabled ? hbVal : null);
      }
      if (gp.status === 'fulfilled') {
        setGlobalProviders(gp.value.providers || []);
      }
      if (at.status === 'fulfilled') {
        setAgentTypes((at.value.agents || []).sort());
      }
    } finally {
      setLoading(false);
    }
  }, [name]);

  useEffect(() => {
    fetchAll();
    const handler = () => fetchAll();
    window.addEventListener('cc:refresh', handler);
    return () => window.removeEventListener('cc:refresh', handler);
  }, [fetchAll]);

  useEffect(() => {
    if (!isDifyAgent) {
      setDifyInputs([]);
      return;
    }
    const ensured = ensureDifyOptionRows(agentOptionRows, name || project?.name);
    setAgentOptionRows(ensured);
    setDifyInputs(buildDifyInputRows(ensured));
    // intentionally runs when switching agent type/project only
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isDifyAgent, name]);

  const hasFeishuLookup = !!project?.platform_configs?.some((pc) => {
    const typ = (pc.type || '').toLowerCase();
    return typ === 'feishu' || typ === 'lark';
  });

  const handleFeishuLookup = async () => {
    if (!name || !lookupQuery.trim()) return;
    setLookupLoading(true);
    try {
      const res = await lookupFeishuIds(name, lookupQuery.trim(), 20);
      setLookupUsers(res.users || []);
      setLookupChats(res.chats || []);
      setLookupWarnings(res.warnings || []);
    } catch (e: any) {
      setLookupUsers([]);
      setLookupChats([]);
      setLookupWarnings([e?.message || String(e)]);
    } finally {
      setLookupLoading(false);
    }
  };

  const handleSaveSettings = async () => {
    if (!name) return;
    setSaving(true);
    try {
      const agentTypeChanged = project && selectedAgentType !== project.agent_type;
      const currentAgentOptions = rowsToOptionMap(agentOptionRows);
      const originalAgentOptions = project?.agent_options || {};
      const agentOptionsChanged = !jsonEqual(currentAgentOptions, originalAgentOptions);
      const activePlatformConfigs = (project?.platform_configs || [])
        .filter((pc) => !removedPlatformIndexes.includes(pc.index ?? -1));
      const editorByIndex = new Map(platformOptionEditors.map((p) => [p.index, p]));
      const platformOptionUpdates = activePlatformConfigs.map((pc) => {
        const index = pc.index ?? 0;
        const key = platformControlKey(pc.type, index);
        const editor = editorByIndex.get(index);
        const options = editor ? rowsToOptionMap(editor.rows) : { ...(pc.options || {}) };
        const platformName = editor ? editor.name.trim() : (pc.name || '').trim();
        const allowFrom = platformAllowFrom[key];
        if (allowFrom !== undefined) options.allow_from = allowFrom.trim();
        const allowChat = platformAllowChat[key];
        if (allowChat !== undefined) {
          const trimmed = allowChat.trim();
          if (trimmed === '') {
            delete options.allow_chat;
          } else {
            options.allow_chat = trimmed;
          }
        }
        return { index, name: platformName, options, originalName: (pc.name || '').trim(), original: pc.options || {} };
      }).filter((u) => !jsonEqual(u.options, u.original) || u.name !== u.originalName)
        .map(({ index, name: platformName, options }) => ({ index, name: platformName, options }));

      const payload: any = {
        language,
        admin_from: adminFrom,
        disabled_commands: disabledCmds.split(',').map(s => s.trim()).filter(Boolean),
        work_dir: workDir,
        mode: agentMode,
        ...(agentTypeChanged ? { agent_type: selectedAgentType } : {}),
        show_context_indicator: showCtxIndicator,
        reply_footer: replyFooter,
        inject_sender: injectSender,
      };
      if (agentOptionsChanged) payload.agent_options = currentAgentOptions;
      if (platformOptionUpdates.length > 0) payload.platform_option_updates = platformOptionUpdates;
      if (removedPlatformIndexes.length > 0) payload.remove_platform_indexes = removedPlatformIndexes;

      const res = await updateProject(name, payload);
      if (res && (res as any).restart_required) {
        setShowRestartModal(true);
        return;
      }
      await fetchAll();
    } finally {
      setSaving(false);
    }
  };

  const handleAddProvider = async () => {
    if (!name || !newProvider.name) return;
    await addProvider(name, newProvider);
    setShowAddProvider(false);
    setNewProvider({ name: '', api_key: '', base_url: '', model: '' });
    fetchAll();
  };

  const handleSetInterval = async () => {
    if (!name) return;
    await setHeartbeatInterval(name, parseInt(newInterval));
    setShowInterval(false);
    fetchAll();
  };

  const newOptionRow = (): OptionRow => ({ id: makeEntryId(), key: '', type: 'string', value: '' });

  const addAgentOptionRow = () => setAgentOptionRows(prev => [...prev, newOptionRow()]);
  const updateAgentOptionRow = (id: string, patch: Partial<OptionRow>) => {
    setAgentOptionRows(prev => prev.map(r => (r.id === id ? { ...r, ...patch } : r)));
  };
  const removeAgentOptionRow = (id: string) => {
    setAgentOptionRows(prev => prev.filter(r => r.id !== id));
  };

  const addPlatformOptionRow = (index: number) => {
    setPlatformOptionEditors(prev => prev.map(p => (p.index === index ? { ...p, rows: [...p.rows, newOptionRow()] } : p)));
  };
  const updatePlatformEditor = (index: number, patch: Partial<PlatformOptionEditor>) => {
    setPlatformOptionEditors(prev => prev.map(p => (p.index === index ? { ...p, ...patch } : p)));
  };
  const updatePlatformOptionRow = (index: number, id: string, patch: Partial<OptionRow>) => {
    setPlatformOptionEditors(prev => prev.map(p => (
      p.index === index ? { ...p, rows: p.rows.map(r => (r.id === id ? { ...r, ...patch } : r)) } : p
    )));
  };
  const removePlatformOptionRow = (index: number, id: string) => {
    setPlatformOptionEditors(prev => prev.map(p => (
      p.index === index ? { ...p, rows: p.rows.filter(r => r.id !== id) } : p
    )));
  };
  const removePlatformFromProject = (index: number) => {
    setRemovedPlatformIndexes(prev => (prev.includes(index) ? prev : [...prev, index]));
    setPlatformOptionEditors(prev => prev.filter(p => p.index !== index));
  };
  const applyPlatformTemplate = (index: number, platformType: string) => {
    setPlatformOptionEditors(prev => prev.map(p => (
      p.index === index ? { ...p, rows: mergeTemplateRows(platformType, p.rows) } : p
    )));
  };
  const initPlatformOptionEditors = () => {
    if (!project?.platform_configs) return;
    setPlatformOptionEditors(
      project.platform_configs
        .filter(pc => !removedPlatformIndexes.includes(pc.index ?? -1))
        .map(pc => ({
          index: pc.index ?? 0,
          name: pc.name || '',
          type: pc.type,
          rows: mergeTemplateRows(pc.type, optionMapToRows(pc.options || {})),
        }))
    );
  };

  const upsertAgentOptionByKey = (key: string, value: string, type: OptionValueType = 'string') => {
    setAgentOptionRows((prev) => {
      const idx = prev.findIndex((row) => row.key.trim() === key);
      if (idx >= 0) {
        const next = [...prev];
        next[idx] = { ...next[idx], key, type, value };
        return next;
      }
      return [...prev, { id: makeEntryId(), key, type, value }];
    });
  };

  const getAgentOptionValue = (key: string, fallback = '') => {
    const row = agentOptionRows.find((r) => r.key.trim() === key);
    return row ? row.value : fallback;
  };

  const syncDifyInputs = (rows: DifyInputRow[]) => {
    const inputsMap = difyInputsToMap(rows);
    upsertAgentOptionByKey('inputs', JSON.stringify(inputsMap), 'json');
  };
  const addDifyInputRow = () => {
    setDifyInputs((prev) => [...prev, { id: makeEntryId(), key: '', value: '' }]);
  };
  const updateDifyInputRow = (id: string, patch: Partial<DifyInputRow>) => {
    setDifyInputs((prev) => {
      const next = prev.map((row) => (row.id === id ? { ...row, ...patch } : row));
      syncDifyInputs(next);
      return next;
    });
  };
  const removeDifyInputRow = (id: string) => {
    setDifyInputs((prev) => {
      const filtered = prev.filter((row) => row.id !== id);
      const next = filtered.length > 0 ? filtered : [{ id: makeEntryId(), key: '', value: '' }];
      syncDifyInputs(next);
      return next;
    });
  };
  const visibleAgentOptionRows = isDifyAgent
    ? agentOptionRows.filter((row) => !DIFY_RESERVED_OPTION_KEYS.has(row.key.trim()))
    : agentOptionRows;

  const tabs: { key: Tab; icon: React.ElementType }[] = [
    { key: 'overview', icon: Layers },
    { key: 'providers', icon: Zap },
    { key: 'heartbeat', icon: Heart },
    { key: 'settings', icon: Settings },
  ];

  if (loading && !project) {
    return <div className="flex items-center justify-center h-64 text-gray-400 animate-pulse">Loading...</div>;
  }

  return (
    <div className="space-y-6 animate-fade-in ">
      {/* Back + title */}
      <div className="flex items-center gap-3">
        <Link to="/projects" className="p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors">
          <ArrowLeft size={18} className="text-gray-400" />
        </Link>
        <h2 className="text-xl font-bold text-gray-900 dark:text-white">{name}</h2>
        {project && <Badge variant="info">{project.agent_type}</Badge>}
      </div>

      {/* Tabs */}
      <div className="flex gap-2">
        {tabs.map(({ key, icon: Icon }) => (
          <button
            key={key}
            onClick={() => setTab(key)}
            className={cn(
              'flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-all',
              tab === key
                ? 'bg-gray-900 dark:bg-gray-700 text-white shadow-md'
                : 'bg-gray-100 dark:bg-gray-800 text-gray-500 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700'
            )}
          >
            <Icon size={16} />
            {t(`projects.tabs.${key}`)}
          </button>
        ))}
      </div>

      {/* Tab content */}
      {tab === 'overview' && project && (
        <div className="space-y-4">
          <Card>
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-sm font-semibold text-gray-900 dark:text-white">{t('projects.platforms')}</h3>
              <Button size="sm" onClick={() => { setShowAddPlatform(true); setAddPlatType(''); }}>
                <Plus size={14} /> {t('setup.addPlatform', 'Add platform')}
              </Button>
            </div>
            <div className="flex flex-wrap gap-2">
              {(project.platform_configs && project.platform_configs.length > 0
                ? project.platform_configs.map((pc, idx) => {
                    const connected = project.platforms?.[idx]?.connected ?? true;
                    const label = (pc.name || '').trim() ? `${pc.name!.trim()} · ${pc.type}` : pc.type;
                    return (
                      <Badge key={`${pc.type}-${idx}`} variant={connected ? 'success' : 'danger'}>
                        <Plug size={12} className="mr-1" /> {label} {connected ? '✓' : '✗'}
                      </Badge>
                    );
                  })
                : project.platforms?.map((p, idx) => (
                    <Badge key={`${p.type}-${idx}`} variant={p.connected ? 'success' : 'danger'}>
                      <Plug size={12} className="mr-1" /> {(p.name || '').trim() ? `${p.name!.trim()} · ${p.type}` : p.type} {p.connected ? '✓' : '✗'}
                    </Badge>
                  )))}
            </div>
          </Card>
          <Card>
            <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-3">{t('sessions.title')}</h3>
            <p className="text-sm text-gray-500 dark:text-gray-400">
              {project.sessions_count} {t('nav.sessions').toLowerCase()}
            </p>
            {project.active_session_keys?.length > 0 && (
              <div className="mt-2 flex flex-wrap gap-1">
                {project.active_session_keys.map((k) => (
                  <Badge key={k} variant="default">{k}</Badge>
                ))}
              </div>
            )}
          </Card>
        </div>
      )}

      {tab === 'providers' && (() => {
        const globalNames = new Set(globalProviders.map(g => g.name));
        const isGlobal = (pName: string) => globalNames.has(pName) && providerRefs.includes(pName);
        const currentAgentType = project?.agent_type || selectedAgentType || '';
        const unlinkedGlobals = globalProviders.filter(g =>
          !providerRefs.includes(g.name) &&
          (!g.agent_types?.length || g.agent_types.includes(currentAgentType))
        );
        return (
        <div className="space-y-4">
          {/* Header */}
          <div className="flex justify-between items-center">
            <h3 className="text-sm font-semibold text-gray-900 dark:text-white">{t('providers.title')}</h3>
            <Button size="sm" onClick={() => { setAddMode('pick'); setShowAddProvider(true); }}><Plus size={14} /> {t('providers.add')}</Button>
          </div>

          {/* Unified provider list */}
          {providers.length === 0 ? (
            <Card>
              <div className="py-6 text-center">
                <Plug size={32} className="mx-auto text-gray-300 dark:text-gray-600 mb-2" />
                <p className="text-sm text-gray-500 dark:text-gray-400">{t('providers.emptyProject', 'No providers configured for this project.')}</p>
                <p className="text-xs text-gray-400 dark:text-gray-500 mt-1">{t('providers.emptyProjectHint', 'Link a global provider or add a custom one.')}</p>
              </div>
            </Card>
          ) : (
            <div className="space-y-2">
              {providers.map((p) => (
                <div
                  key={p.name}
                  className={cn(
                    'flex items-center justify-between px-4 py-3 rounded-xl border transition-all',
                    p.active
                      ? 'border-emerald-200 dark:border-emerald-500/20 bg-emerald-50/50 dark:bg-emerald-900/10'
                      : 'border-gray-200 dark:border-gray-700/60 bg-white dark:bg-gray-800/40',
                  )}
                >
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium text-gray-900 dark:text-white">{p.name}</span>
                      {p.active && <Badge variant="success">{t('providers.active')}</Badge>}
                      {isGlobal(p.name) && (
                        <Link to="/providers" className="inline-flex items-center gap-0.5 text-[10px] text-gray-400 hover:text-accent transition-colors">
                          <Link2 size={10} /> {t('providers.global', 'global')}
                        </Link>
                      )}
                    </div>
                    <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5 truncate">
                      {p.model}{p.base_url ? ` · ${p.base_url}` : ''}
                    </p>
                  </div>
                  <div className="flex items-center gap-1.5 shrink-0 ml-3">
                    {!p.active && (
                      <Button size="sm" variant="ghost" onClick={() => { activateProvider(name!, p.name).then(fetchAll); }}>
                        <Zap size={14} /> {t('providers.activate')}
                      </Button>
                    )}
                    {!p.active && (
                      isGlobal(p.name) ? (
                        <Button
                          size="sm"
                          variant="ghost"
                          className="text-gray-400 hover:text-red-500"
                          onClick={async () => {
                            const next = providerRefs.filter(r => r !== p.name);
                            setSavingRefs(true);
                            try {
                              await saveProviderRefs(name!, next);
                              await fetchAll();
                            } finally { setSavingRefs(false); }
                          }}
                        >
                          <Trash2 size={14} />
                        </Button>
                      ) : (
                        <Button size="sm" variant="ghost" className="text-gray-400 hover:text-red-500" onClick={() => { removeProvider(name!, p.name).then(fetchAll); }}>
                          <Trash2 size={14} />
                        </Button>
                      )
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}

          {/* Add Provider Modal */}
          <Modal open={showAddProvider} onClose={() => setShowAddProvider(false)} title={t('providers.add')}>
            <div className="space-y-4">
              {/* Toggle */}
              <div className="flex rounded-lg bg-gray-100 dark:bg-gray-800 p-0.5">
                <button
                  className={cn('flex-1 px-3 py-1.5 rounded-md text-xs font-medium transition-all', addMode === 'pick' ? 'bg-white dark:bg-gray-700 text-gray-900 dark:text-white shadow-sm' : 'text-gray-500')}
                  onClick={() => setAddMode('pick')}
                >{t('providers.linkGlobal', 'Link global')}</button>
                <button
                  className={cn('flex-1 px-3 py-1.5 rounded-md text-xs font-medium transition-all', addMode === 'custom' ? 'bg-white dark:bg-gray-700 text-gray-900 dark:text-white shadow-sm' : 'text-gray-500')}
                  onClick={() => setAddMode('custom')}
                >{t('providers.addCustom', 'Add custom')}</button>
              </div>

              {addMode === 'pick' ? (
                unlinkedGlobals.length === 0 ? (
                  <div className="py-4 text-center">
                    <p className="text-sm text-gray-500 dark:text-gray-400">{t('providers.allLinked', 'All global providers are already linked.')}</p>
                    <Link to="/providers" className="inline-flex items-center gap-1 mt-2 text-xs text-accent hover:underline">
                      {t('providers.manageGlobal', 'Manage global providers')} <ExternalLink size={11} />
                    </Link>
                  </div>
                ) : (
                  <div className="space-y-1.5 max-h-64 overflow-y-auto">
                    {unlinkedGlobals.map(gp => (
                      <button
                        key={gp.name}
                        className="flex items-center justify-between w-full px-3 py-2.5 rounded-lg border border-gray-200 dark:border-gray-700 hover:border-accent/40 hover:bg-accent/5 transition-all text-left"
                        onClick={async () => {
                          const next = [...providerRefs, gp.name];
                          setSavingRefs(true);
                          try {
                            await saveProviderRefs(name!, next);
                            await fetchAll();
                          } finally { setSavingRefs(false); }
                          setShowAddProvider(false);
                        }}
                      >
                        <div className="min-w-0">
                          <div className="text-sm font-medium text-gray-900 dark:text-white">{gp.name}</div>
                          <div className="text-xs text-gray-500 dark:text-gray-400 truncate">{gp.model}{gp.base_url ? ` · ${gp.base_url}` : ''}</div>
                        </div>
                        <Plus size={16} className="shrink-0 text-gray-400" />
                      </button>
                    ))}
                  </div>
                )
              ) : (
                <div className="space-y-3">
                  <Input label={t('providers.name')} value={newProvider.name} onChange={(e) => setNewProvider({...newProvider, name: e.target.value})} />
                  <Input label="API Key" type="password" value={newProvider.api_key} onChange={(e) => setNewProvider({...newProvider, api_key: e.target.value})} />
                  <Input label={t('providers.baseUrl')} value={newProvider.base_url} onChange={(e) => setNewProvider({...newProvider, base_url: e.target.value})} placeholder="https://api.example.com" />
                  <Input label={t('providers.model')} value={newProvider.model} onChange={(e) => setNewProvider({...newProvider, model: e.target.value})} />
                  <div className="flex justify-end gap-2 pt-1">
                    <Button variant="secondary" onClick={() => setShowAddProvider(false)}>{t('common.cancel')}</Button>
                    <Button onClick={handleAddProvider}>{t('providers.add')}</Button>
                  </div>
                </div>
              )}
            </div>
          </Modal>
        </div>
        );
      })()}

      {tab === 'heartbeat' && (
        <div className="space-y-4">
          {!heartbeat ? (
            <EmptyState message={t('heartbeat.notEnabled', 'Heartbeat is not configured for this project. Add [heartbeat] section in config.toml to enable.')} />
          ) : (
            <>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <Card><p className="text-xs text-gray-500">{t('heartbeat.status')}</p><p className="text-lg font-bold text-gray-900 dark:text-white mt-1">{heartbeat.paused ? t('heartbeat.paused') : t('heartbeat.running')}</p></Card>
                <Card><p className="text-xs text-gray-500">{t('heartbeat.interval')}</p><p className="text-lg font-bold text-gray-900 dark:text-white mt-1">{heartbeat.interval_mins}m</p></Card>
                <Card><p className="text-xs text-gray-500">{t('heartbeat.runCount')}</p><p className="text-lg font-bold text-gray-900 dark:text-white mt-1">{heartbeat.run_count}</p></Card>
                <Card><p className="text-xs text-gray-500">{t('heartbeat.errorCount')}</p><p className="text-lg font-bold text-gray-900 dark:text-white mt-1">{heartbeat.error_count}</p></Card>
              </div>
              <Card>
                <div className="space-y-2 text-sm">
                  <p className="text-gray-500">{t('heartbeat.lastRun')}: <span className="text-gray-900 dark:text-white">{formatTime(heartbeat.last_run)}</span></p>
                  <p className="text-gray-500">{t('heartbeat.skippedBusy')}: <span className="text-gray-900 dark:text-white">{heartbeat.skipped_busy}</span></p>
                  {heartbeat.last_error && <p className="text-red-500">{heartbeat.last_error}</p>}
                </div>
              </Card>
              <div className="flex gap-2">
                {heartbeat.paused ? (
                  <Button onClick={() => { resumeHeartbeat(name!).then(fetchAll); }}><Play size={14} /> {t('heartbeat.resume')}</Button>
                ) : (
                  <Button variant="secondary" onClick={() => { pauseHeartbeat(name!).then(fetchAll); }}><Pause size={14} /> {t('heartbeat.pause')}</Button>
                )}
                <Button variant="secondary" onClick={() => { triggerHeartbeat(name!).then(fetchAll); }}><Heart size={14} /> {t('heartbeat.trigger')}</Button>
                <Button variant="secondary" onClick={() => setShowInterval(true)}><Clock size={14} /> {t('heartbeat.setInterval')}</Button>
              </div>
            </>
          )}
          <Modal open={showInterval} onClose={() => setShowInterval(false)} title={t('heartbeat.setInterval')}>
            <div className="space-y-3">
              <Input label={`${t('heartbeat.interval')} (min)`} type="number" value={newInterval} onChange={(e) => setNewInterval(e.target.value)} />
              <div className="flex justify-end gap-2 pt-2">
                <Button variant="secondary" onClick={() => setShowInterval(false)}>{t('common.cancel')}</Button>
                <Button onClick={handleSetInterval}>{t('common.save')}</Button>
              </div>
            </div>
          </Modal>
        </div>
      )}

      {tab === 'settings' && project && (
        <div className="space-y-4">
        {/* Agent settings */}
        <Card>
          <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-4">{t('projects.agentSettings', 'Agent')}</h3>
          <div className="space-y-4 max-w-lg">
            <div>
              <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">
                {t('projects.agentType', 'Agent type')}
              </label>
              <select
                value={selectedAgentType}
                onChange={(e) => setSelectedAgentType(e.target.value)}
                className="w-full px-3 py-2 text-sm rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-accent/50"
              >
                {agentTypes.map(a => <option key={a} value={a}>{a}</option>)}
                {selectedAgentType && !agentTypes.includes(selectedAgentType) && (
                  <option value={selectedAgentType}>{selectedAgentType}</option>
                )}
              </select>
              {selectedAgentType !== project.agent_type && (
                <p className="text-[11px] text-amber-500 mt-1">{t('projects.agentTypeChangeHint', 'Changing agent type requires restart. Incompatible providers will be removed.')}</p>
              )}
            </div>
            <Input label={t('projects.workDir', 'Working directory')} value={workDir} onChange={(e) => setWorkDir(e.target.value)} placeholder="/path/to/project" />
            <div>
              <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">
                {t('projects.agentMode', 'Permission mode')}
              </label>
              <select
                value={agentMode}
                onChange={(e) => setAgentMode(e.target.value)}
                className="w-full px-3 py-2 text-sm rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-accent/50"
              >
                <option value="default">default</option>
                <option value="acceptEdits">acceptEdits (edit)</option>
                <option value="plan">plan</option>
                <option value="bypassPermissions">bypassPermissions (yolo)</option>
                <option value="dontAsk">dontAsk</option>
              </select>
            </div>
          </div>
        </Card>

        {/* General settings */}
        <Card>
          <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-4">{t('projects.generalSettings', 'General')}</h3>
          <div className="space-y-4 max-w-lg">
            <div className="flex items-center justify-between">
              <div>
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300">{t('projects.showCtxIndicator', 'Context indicator')}</label>
                <p className="text-[11px] text-gray-400 mt-0.5">{t('projects.showCtxIndicatorHint', 'Show [ctx: ~N%] suffix on replies')}</p>
              </div>
              <button
                onClick={() => setShowCtxIndicator(!showCtxIndicator)}
                className={cn('w-10 h-6 rounded-full transition-colors', showCtxIndicator ? 'bg-accent' : 'bg-gray-300 dark:bg-gray-700')}
              >
                <div className={cn('w-4 h-4 bg-white rounded-full transition-transform mx-1', showCtxIndicator ? 'translate-x-4' : 'translate-x-0')} />
              </button>
            </div>
            <div className="flex items-center justify-between">
              <div>
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300">{t('projects.replyFooter', 'Reply footer')}</label>
                <p className="text-[11px] text-gray-400 mt-0.5">{t('projects.replyFooterHint', 'Append model/usage metadata to replies')}</p>
              </div>
              <button
                onClick={() => setReplyFooter(!replyFooter)}
                className={cn('w-10 h-6 rounded-full transition-colors', replyFooter ? 'bg-accent' : 'bg-gray-300 dark:bg-gray-700')}
              >
                <div className={cn('w-4 h-4 bg-white rounded-full transition-transform mx-1', replyFooter ? 'translate-x-4' : 'translate-x-0')} />
              </button>
            </div>
            <div className="flex items-center justify-between">
              <div>
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300">{t('projects.injectSender', 'Inject sender')}</label>
                <p className="text-[11px] text-gray-400 mt-0.5">{t('projects.injectSenderHint', 'Prepend sender identity to messages sent to agent')}</p>
              </div>
              <button
                onClick={() => setInjectSender(!injectSender)}
                className={cn('w-10 h-6 rounded-full transition-colors', injectSender ? 'bg-accent' : 'bg-gray-300 dark:bg-gray-700')}
              >
                <div className={cn('w-4 h-4 bg-white rounded-full transition-transform mx-1', injectSender ? 'translate-x-4' : 'translate-x-0')} />
              </button>
            </div>
            <Input label={t('projects.language')} value={language} onChange={(e) => setLanguage(e.target.value)} placeholder="en, zh, ja..." />
            <Input label={t('projects.adminFrom')} value={adminFrom} onChange={(e) => setAdminFrom(e.target.value)} placeholder="user1,user2 or *" />
            <Input label={t('projects.disabledCommands')} value={disabledCmds} onChange={(e) => setDisabledCmds(e.target.value)} placeholder="restart, upgrade, cron" />
          </div>
        </Card>

        <Card>
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-semibold text-gray-900 dark:text-white">{t('projects.dynamicConfig', 'Dynamic config')}</h3>
            <Button size="sm" variant="secondary" onClick={addAgentOptionRow}>
              <Plus size={13} /> {t('projects.addAgentOption', 'Add agent option')}
            </Button>
          </div>
          <p className="text-xs text-gray-500 dark:text-gray-400 mb-3">
            {t('projects.dynamicConfigHint', 'Edit [[projects]] -> [projects.agent.options] and [projects.platforms.options] directly. Saving these fields requires restart.')}
          </p>
          {isDifyAgent && (
            <div className="mb-4 rounded-lg border border-gray-200 dark:border-gray-700 p-3 space-y-3">
              <p className="text-xs font-medium text-gray-600 dark:text-gray-400">Dify options</p>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                <Input
                  label="base_url *"
                  value={getAgentOptionValue('base_url', 'https://api.dify.ai/v1')}
                  onChange={(e) => upsertAgentOptionByKey('base_url', e.target.value)}
                  placeholder="https://api.dify.ai/v1"
                />
                <Input
                  label="api_key *"
                  type="password"
                  value={getAgentOptionValue('api_key')}
                  onChange={(e) => upsertAgentOptionByKey('api_key', e.target.value)}
                  placeholder="app-***"
                />
              </div>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
                <div>
                  <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">app_mode</label>
                  <select
                    value={getAgentOptionValue('app_mode', 'advanced-chat')}
                    onChange={(e) => upsertAgentOptionByKey('app_mode', e.target.value)}
                    className="w-full px-3 py-2 text-sm rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-accent/50"
                  >
                    <option value="advanced-chat">advanced-chat</option>
                    <option value="chat">chat</option>
                    <option value="completion">completion</option>
                    <option value="workflow">workflow</option>
                  </select>
                </div>
                <Input
                  label="user"
                  value={getAgentOptionValue('user', `cc-connect:${name || 'project'}`)}
                  onChange={(e) => upsertAgentOptionByKey('user', e.target.value)}
                  placeholder="cc-connect:my-backend"
                />
                <Input
                  label="query_input_key"
                  value={getAgentOptionValue('query_input_key', 'query')}
                  onChange={(e) => upsertAgentOptionByKey('query_input_key', e.target.value)}
                  placeholder="query"
                />
              </div>
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <p className="text-xs font-medium text-gray-600 dark:text-gray-400">inputs (dynamic)</p>
                  <Button size="sm" variant="secondary" onClick={addDifyInputRow}>
                    <Plus size={12} /> {t('common.add', 'Add')}
                  </Button>
                </div>
                <div className="space-y-2">
                  {difyInputs.map((row) => (
                    <div key={row.id} className="grid grid-cols-[1fr_1fr_auto] gap-2 items-end">
                      <Input
                        label="key"
                        value={row.key}
                        onChange={(e) => updateDifyInputRow(row.id, { key: e.target.value })}
                        placeholder="tenant"
                      />
                      <Input
                        label="value"
                        value={row.value}
                        onChange={(e) => updateDifyInputRow(row.id, { value: e.target.value })}
                        placeholder="ops"
                      />
                      <button
                        type="button"
                        onClick={() => removeDifyInputRow(row.id)}
                        className="h-9 w-9 mb-[1px] rounded-lg border border-gray-300 dark:border-gray-700 text-gray-500 hover:text-red-500 hover:border-red-300"
                        aria-label="remove dify input row"
                      >
                        <Trash2 size={14} className="mx-auto" />
                      </button>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}
          <div className="space-y-2 mb-4">
            <p className="text-xs font-medium text-gray-600 dark:text-gray-400">{t('projects.agentOptions', 'Agent options')}</p>
            {visibleAgentOptionRows.length === 0 ? (
              <p className="text-xs text-gray-400">{t('projects.noAgentOptions', 'No options. Click "Add agent option".')}</p>
            ) : visibleAgentOptionRows.map(row => (
              <div key={row.id} className="grid grid-cols-12 gap-2 items-center">
                <input
                  value={row.key}
                  onChange={(e) => updateAgentOptionRow(row.id, { key: e.target.value })}
                  placeholder="key"
                  className="col-span-4 px-3 py-2 text-sm rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
                />
                <select
                  value={row.type}
                  onChange={(e) => updateAgentOptionRow(row.id, { type: e.target.value as OptionValueType })}
                  className="col-span-3 px-3 py-2 text-sm rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
                >
                  <option value="string">string</option>
                  <option value="number">number</option>
                  <option value="boolean">boolean</option>
                  <option value="json">json</option>
                </select>
                {row.type === 'boolean' ? (
                  <select
                    value={row.value.toLowerCase() === 'true' ? 'true' : 'false'}
                    onChange={(e) => updateAgentOptionRow(row.id, { value: e.target.value })}
                    className="col-span-4 px-3 py-2 text-sm rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
                  >
                    <option value="true">true</option>
                    <option value="false">false</option>
                  </select>
                ) : (
                  <input
                    value={row.value}
                    onChange={(e) => updateAgentOptionRow(row.id, { value: e.target.value })}
                    placeholder={row.type === 'json' ? '{"k":"v"}' : 'value'}
                    className="col-span-4 px-3 py-2 text-sm rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
                  />
                )}
                <Button size="sm" variant="ghost" className="col-span-1 text-red-500" onClick={() => removeAgentOptionRow(row.id)}>
                  <Trash2 size={14} />
                </Button>
              </div>
            ))}
          </div>
          <div className="space-y-3">
            <p className="text-xs font-medium text-gray-600 dark:text-gray-400">{t('projects.platformOptions', 'Platform options')}</p>
            {platformOptionEditors.length === 0 ? (
              <div className="rounded-lg border border-dashed border-gray-300 dark:border-gray-700 p-3">
                <p className="text-xs text-gray-400 mb-2">{t('projects.noPlatformOptions', 'No platform option blocks available.')}</p>
                {(project.platform_configs?.length || 0) > 0 && (
                  <Button size="sm" variant="secondary" onClick={initPlatformOptionEditors}>
                    <Plus size={12} /> {t('projects.initPlatformOptions', 'Initialize platform options')}
                  </Button>
                )}
              </div>
            ) : platformOptionEditors.map(p => {
              const rows = sortRowsByTemplate(p.type, p.rows);
              const hasTemplate = !!platformMeta[p.type];
              const platformLabel = p.name.trim() ? `${p.name.trim()} · ${p.type}` : `${p.type} #${p.index}`;
              return (
              <div key={`${p.type}-${p.index}`} className="rounded-lg border border-gray-200 dark:border-gray-700 p-3 space-y-2">
                <div className="flex items-center justify-between">
                  <p className="text-sm font-medium text-gray-800 dark:text-gray-200">{platformLabel}</p>
                  <div className="flex items-center gap-2">
                    {hasTemplate && (
                      <Button size="sm" variant="secondary" onClick={() => applyPlatformTemplate(p.index, p.type)}>
                        {t('projects.useTemplateFields', 'Use template fields')}
                      </Button>
                    )}
                    <Button size="sm" variant="secondary" onClick={() => addPlatformOptionRow(p.index)}>
                      <Plus size={12} /> {t('common.add', 'Add')}
                    </Button>
                    <Button size="sm" variant="ghost" className="text-red-500" onClick={() => removePlatformFromProject(p.index)}>
                      <Trash2 size={13} /> {t('common.delete')}
                    </Button>
                  </div>
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">
                    {t('projects.platformDisplayName', 'Bot display name')}
                  </label>
                  <input
                    value={p.name}
                    onChange={(e) => updatePlatformEditor(p.index, { name: e.target.value })}
                    placeholder={t('projects.platformDisplayNamePlaceholder', 'e.g. Ops Bot')}
                    className="w-full px-3 py-2 text-sm rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
                  />
                </div>
                {rows.length === 0 ? (
                  <p className="text-xs text-gray-400">{t('projects.noPlatformOptionEntries', 'No options. Click Add.')}</p>
                ) : rows.map(row => {
                  const fieldDef = findPlatformField(p.type, row.key);
                  return (
                  <div key={row.id} className="grid grid-cols-12 gap-2 items-center">
                    <input
                      value={row.key}
                      onChange={(e) => updatePlatformOptionRow(p.index, row.id, { key: e.target.value })}
                      placeholder={fieldDef?.key || 'key'}
                      className="col-span-4 px-3 py-2 text-sm rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
                    />
                    <select
                      value={row.type}
                      onChange={(e) => updatePlatformOptionRow(p.index, row.id, { type: e.target.value as OptionValueType })}
                      className="col-span-3 px-3 py-2 text-sm rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
                    >
                      <option value="string">string</option>
                      <option value="number">number</option>
                      <option value="boolean">boolean</option>
                      <option value="json">json</option>
                    </select>
                    {row.type === 'boolean' ? (
                      <select
                        value={row.value.toLowerCase() === 'true' ? 'true' : 'false'}
                        onChange={(e) => updatePlatformOptionRow(p.index, row.id, { value: e.target.value })}
                        className="col-span-4 px-3 py-2 text-sm rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
                      >
                        <option value="true">true</option>
                        <option value="false">false</option>
                      </select>
                    ) : (
                      <input
                        value={row.value}
                        onChange={(e) => updatePlatformOptionRow(p.index, row.id, { value: e.target.value })}
                        placeholder={row.type === 'json' ? '{"k":"v"}' : (fieldDef?.placeholder || 'value')}
                        className="col-span-4 px-3 py-2 text-sm rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
                      />
                    )}
                    <Button size="sm" variant="ghost" className="col-span-1 text-red-500" onClick={() => removePlatformOptionRow(p.index, row.id)}>
                      <Trash2 size={14} />
                    </Button>
                  </div>
                )})}
              </div>
            )})}
          </div>
        </Card>

        {/* Per-platform access control */}
        {project.platform_configs && project.platform_configs.length > 0 && (
          <Card>
            <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-4">{t('projects.platformAccess', 'Platform access control')}</h3>
            <div className="space-y-3 max-w-lg">
              {project.platform_configs
                .filter(pc => !removedPlatformIndexes.includes(pc.index ?? -1))
                .map(pc => {
                  const idx = pc.index ?? 0;
                  const key = platformControlKey(pc.type, idx);
                  const typ = (pc.type || '').toLowerCase();
                  const canSetChat = typ === 'feishu' || typ === 'lark';
                  const platformLabel = (pc.name || '').trim() ? `${pc.name!.trim()} · ${pc.type}` : `${pc.type} #${idx}`;
                  return (
                    <div key={`${pc.type}-${idx}`} className="rounded-lg border border-gray-200 dark:border-gray-700 p-3 space-y-2">
                      <p className="text-xs font-medium text-gray-600 dark:text-gray-400">{platformLabel}</p>
                      <Input
                        label={t('fields.allowFrom')}
                        value={platformAllowFrom[key] ?? pc.allow_from ?? ''}
                        onChange={(e) => setPlatformAllowFrom(prev => ({ ...prev, [key]: e.target.value }))}
                        placeholder="user_open_id_1,user_open_id_2 or *"
                      />
                      {canSetChat && (
                        <Input
                          label={t('fields.allowChat', 'Allowed chats')}
                          value={platformAllowChat[key] ?? (typeof pc.options?.allow_chat === 'string' ? pc.options.allow_chat : '')}
                          onChange={(e) => setPlatformAllowChat(prev => ({ ...prev, [key]: e.target.value }))}
                          placeholder="oc_xxx,oc_yyy"
                        />
                      )}
                    </div>
                  );
                })}
            </div>
            {hasFeishuLookup && (
              <div className="mt-5 pt-4 border-t border-gray-200 dark:border-gray-700">
                <p className="text-sm font-medium text-gray-800 dark:text-gray-200 mb-2">
                  {t('projects.feishuLookup', 'Feishu 人/群 ID 查询')}
                </p>
                <p className="text-xs text-gray-500 dark:text-gray-400 mb-3">
                  {t('projects.feishuLookupHint', '输入人名或群名，查询 open_id / chat_id，用于 allow_from / allow_chat。')}
                </p>
                <div className="flex gap-2 max-w-xl">
                  <input
                    value={lookupQuery}
                    onChange={(e) => setLookupQuery(e.target.value)}
                    onKeyDown={(e) => { if (e.key === 'Enter') handleFeishuLookup(); }}
                    placeholder={t('projects.feishuLookupPlaceholder', '例如：张三 / 运维群')}
                    className="flex-1 px-3 py-2 text-sm rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
                  />
                  <Button size="sm" onClick={handleFeishuLookup} loading={lookupLoading}>
                    {t('common.search', 'Search')}
                  </Button>
                </div>
                {lookupWarnings.length > 0 && (
                  <div className="mt-3 space-y-1">
                    {lookupWarnings.map((w, i) => (
                      <p key={`${w}-${i}`} className="text-xs text-amber-600 dark:text-amber-400">{w}</p>
                    ))}
                  </div>
                )}
                {(lookupUsers.length > 0 || lookupChats.length > 0) && (
                  <div className="mt-3 space-y-3">
                    {lookupUsers.length > 0 && (
                      <div>
                        <p className="text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">Users</p>
                        <div className="space-y-1">
                          {lookupUsers.map((u) => (
                            <div key={u.open_id} className="text-xs rounded border border-gray-200 dark:border-gray-700 px-2 py-1 bg-gray-50/70 dark:bg-gray-900/40">
                              <span className="font-medium text-gray-800 dark:text-gray-100">{u.name || u.en_name || u.nickname || '(unknown)'}</span>
                              <span className="text-gray-500 dark:text-gray-400 ml-2">{u.open_id}</span>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                    {lookupChats.length > 0 && (
                      <div>
                        <p className="text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">Chats</p>
                        <div className="space-y-1">
                          {lookupChats.map((c) => (
                            <div key={c.chat_id} className="text-xs rounded border border-gray-200 dark:border-gray-700 px-2 py-1 bg-gray-50/70 dark:bg-gray-900/40">
                              <span className="font-medium text-gray-800 dark:text-gray-100">{c.name || '(unnamed chat)'}</span>
                              <span className="text-gray-500 dark:text-gray-400 ml-2">{c.chat_id}</span>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                )}
              </div>
            )}
          </Card>
        )}

        <div className="max-w-lg">
          <Button loading={saving} onClick={handleSaveSettings}>{t('common.save')}</Button>
        </div>
        <Card>
          <h3 className="text-sm font-semibold text-red-600 dark:text-red-400 mb-3">{t('projects.dangerZone', 'Danger Zone')}</h3>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-700 dark:text-gray-300">{t('projects.deleteTitle')}</p>
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">{t('projects.deleteHint', 'Remove this project from config. Requires restart.')}</p>
            </div>
            <Button variant="danger" size="sm" onClick={() => setShowDeleteConfirm(true)}>
              <Trash2 size={14} /> {t('common.delete')}
            </Button>
          </div>
        </Card>
        </div>
      )}

      {/* Delete confirmation */}
      <Modal open={showDeleteConfirm} onClose={() => setShowDeleteConfirm(false)} title={t('projects.deleteTitle')}>
        <div className="space-y-4 py-2">
          <p className="text-sm text-gray-600 dark:text-gray-400">
            {t('projects.deleteConfirm', { name })}
          </p>
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setShowDeleteConfirm(false)}>{t('common.cancel')}</Button>
            <Button variant="danger" onClick={handleDeleteProject} disabled={deleting}>
              {deleting ? t('common.deleting', 'Deleting...') : t('common.delete')}
            </Button>
          </div>
        </div>
      </Modal>

      {/* Add Platform Modal */}
      <Modal
        open={showAddPlatform}
        onClose={() => { setShowAddPlatform(false); setAddPlatType(''); setAddPlatMode(''); }}
        title={t('setup.addPlatform', 'Add platform')}
      >
        {!addPlatType ? (
          <div className="space-y-3 py-2">
            <p className="text-sm text-gray-500 dark:text-gray-400 mb-2">
              {t('setup.choosePlatform', 'Choose a platform to connect:')}
            </p>
            <div className="grid grid-cols-2 gap-2 max-h-80 overflow-y-auto">
              {PLATFORM_OPTIONS.map(({ key, label, color, qr, abbr }) => (
                <button
                  key={key}
                  onClick={() => {
                    setAddPlatType(key);
                    if (hasBothPlatformModes(key)) {
                      setAddPlatMode('');
                    } else if (supportsQRPlatform(key)) {
                      setAddPlatMode('qr');
                    } else if (supportsManualPlatform(key)) {
                      setAddPlatMode('manual');
                    } else {
                      setAddPlatMode('');
                    }
                  }}
                  className="flex items-center gap-2.5 p-3 rounded-xl border border-gray-200 dark:border-gray-700 hover:border-accent/50 hover:bg-accent/5 transition-all text-left"
                >
                  <div className={`w-9 h-9 rounded-lg ${color} flex items-center justify-center shrink-0 font-bold text-xs`}>
                    {abbr}
                  </div>
                  <div className="min-w-0">
                    <div className="text-sm font-medium text-gray-900 dark:text-white truncate">{label}</div>
                    <div className="text-[11px] text-gray-400">
                      {hasBothPlatformModes(key)
                        ? t('setup.scanOrManual', 'QR or manual')
                        : qr ? t('setup.scanToConnect', 'Scan QR code') : t('setup.manualSetup', 'Manual setup')}
                    </div>
                  </div>
                </button>
              ))}
            </div>
          </div>
        ) : addPlatMode === '' && hasBothPlatformModes(addPlatType) ? (
          <div className="space-y-3 py-2">
            <p className="text-sm text-gray-500 dark:text-gray-400 mb-2">
              {t('setup.chooseConnectionMode', 'Choose connection mode:')}
            </p>
            <div className="grid grid-cols-1 gap-2">
              <button
                onClick={() => setAddPlatMode('qr')}
                className="flex items-center justify-between p-3 rounded-xl border border-gray-200 dark:border-gray-700 hover:border-accent/50 hover:bg-accent/5 transition-all text-left"
              >
                <div>
                  <div className="text-sm font-medium text-gray-900 dark:text-white">{t('setup.scanToConnect', 'Scan QR code')}</div>
                  <div className="text-[11px] text-gray-400">{t('setup.scanModeHint', 'Connect quickly by scanning on phone')}</div>
                </div>
                <Settings size={16} className="text-gray-400" />
              </button>
              <button
                onClick={() => setAddPlatMode('manual')}
                className="flex items-center justify-between p-3 rounded-xl border border-gray-200 dark:border-gray-700 hover:border-accent/50 hover:bg-accent/5 transition-all text-left"
              >
                <div>
                  <div className="text-sm font-medium text-gray-900 dark:text-white">{t('setup.manualSetup', 'Manual setup')}</div>
                  <div className="text-[11px] text-gray-400">{t('setup.manualModeHint', 'Fill in app credentials manually')}</div>
                </div>
                <Plug size={16} className="text-gray-400" />
              </button>
            </div>
            <div className="flex justify-start pt-2">
              <Button variant="secondary" onClick={() => { setAddPlatType(''); setAddPlatMode(''); }}>
                {t('common.back')}
              </Button>
            </div>
          </div>
        ) : addPlatMode === 'qr' && supportsQRPlatform(addPlatType) ? (
          <PlatformSetupQR
            platformType={addPlatType as 'feishu' | 'weixin'}
            projectName={name!}
            onComplete={(restarted) => {
              setShowAddPlatform(false);
              setAddPlatType('');
              setAddPlatMode('');
              if (!restarted) {
                setShowRestartModal(true);
              } else {
                fetchAll();
              }
            }}
            onCancel={() => {
              if (hasBothPlatformModes(addPlatType)) {
                setAddPlatMode('');
              } else {
                setAddPlatType('');
              }
            }}
          />
        ) : addPlatMode === 'manual' && platformMeta[addPlatType] ? (
          <PlatformManualForm
            platformType={addPlatType}
            projectName={name!}
            onComplete={() => {
              setShowAddPlatform(false);
              setAddPlatType('');
              setAddPlatMode('');
              setShowRestartModal(true);
            }}
            onCancel={() => {
              if (hasBothPlatformModes(addPlatType)) {
                setAddPlatMode('');
              } else {
                setAddPlatType('');
              }
            }}
          />
        ) : (
          <div className="space-y-4 py-4 text-center">
            <p className="text-sm text-gray-600 dark:text-gray-400">
              {t('setup.manualHint', 'For {{platform}}, please configure credentials in config.toml and restart the service.', { platform: PLATFORM_OPTIONS.find(o => o.key === addPlatType)?.label || addPlatType })}
            </p>
            <Button variant="secondary" onClick={() => { setAddPlatType(''); setAddPlatMode(''); }}>{t('common.back')}</Button>
          </div>
        )}
      </Modal>

      {/* Restart Required Modal */}
      <Modal open={showRestartModal} onClose={() => setShowRestartModal(false)} title={t('setup.restartRequired', 'Restart required')}>
        <div className="space-y-4 py-2">
          <p className="text-sm text-gray-600 dark:text-gray-400">
            {t('setup.restartHint', 'Restart the service for the new platform to take effect.')}
          </p>
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => { setShowRestartModal(false); setTimeout(fetchAll, 300); }}>
              {t('setup.later', 'Later')}
            </Button>
            <Button onClick={async () => { await restartSystem(); setShowRestartModal(false); await waitForService(8000); await fetchAll(); }}>
              {t('setup.restartNow', 'Restart now')}
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
