import { useEffect, useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';
import { Server, Heart, ArrowRight, FolderKanban, Plus, Smartphone, Settings2, Trash2 } from 'lucide-react';
import { Card, Badge, Button, Input, Modal, EmptyState } from '@/components/ui';
import { listProjects, listAgentTypes, type ProjectSummary } from '@/api/projects';
import PlatformSetupQR from './PlatformSetupQR';
import PlatformManualForm from './PlatformManualForm';
import { platformMeta } from '@/lib/platformMeta';
import { restartSystem } from '@/api/status';

const AGENT_LABELS: Record<string, string> = {
  claudecode: 'Claude Code',
  codex: 'Codex',
  gemini: 'Gemini CLI',
  cursor: 'Cursor',
  devin: 'Devin',
  dify: 'Dify',
  acp: 'ACP (Generic)',
  opencode: 'OpenCode',
  qoder: 'Qoder',
  iflow: 'iFlow',
  kimi: 'Kimi',
  pi: 'Pi',
};

const FALLBACK_AGENT_KEYS = ['claudecode', 'codex', 'gemini', 'cursor', 'dify', 'devin', 'acp', 'opencode', 'qoder'];

type DifyInputRow = {
  id: string;
  key: string;
  value: string;
};

const makeInputRowId = () => `${Date.now()}-${Math.random().toString(16).slice(2, 8)}`;

const PLATFORM_OPTIONS: { key: string; label: string; color: string; qr?: boolean }[] = [
  { key: 'feishu', label: 'Feishu / Lark', color: 'bg-blue-50 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400', qr: true },
  { key: 'weixin', label: 'WeChat', color: 'bg-green-50 dark:bg-green-900/30 text-green-600 dark:text-green-400', qr: true },
  { key: 'telegram', label: 'Telegram', color: 'bg-sky-50 dark:bg-sky-900/30 text-sky-600 dark:text-sky-400' },
  { key: 'discord', label: 'Discord', color: 'bg-indigo-50 dark:bg-indigo-900/30 text-indigo-600 dark:text-indigo-400' },
  { key: 'slack', label: 'Slack', color: 'bg-purple-50 dark:bg-purple-900/30 text-purple-600 dark:text-purple-400' },
  { key: 'dingtalk', label: 'DingTalk', color: 'bg-orange-50 dark:bg-orange-900/30 text-orange-600 dark:text-orange-400' },
  { key: 'wecom', label: 'WeChat Work', color: 'bg-emerald-50 dark:bg-emerald-900/30 text-emerald-600 dark:text-emerald-400' },
  { key: 'qq', label: 'QQ (OneBot)', color: 'bg-cyan-50 dark:bg-cyan-900/30 text-cyan-600 dark:text-cyan-400' },
  { key: 'qqbot', label: 'QQ Bot (Official)', color: 'bg-cyan-50 dark:bg-cyan-900/30 text-cyan-600 dark:text-cyan-400' },
  { key: 'line', label: 'LINE', color: 'bg-lime-50 dark:bg-lime-900/30 text-lime-600 dark:text-lime-400' },
  { key: 'weibo', label: 'Weibo (微博)', color: 'bg-red-50 dark:bg-red-900/30 text-red-600 dark:text-red-400' },
];

export default function ProjectList() {
  const { t } = useTranslation();
  const [projects, setProjects] = useState<ProjectSummary[]>([]);
  const [loading, setLoading] = useState(true);

  // Add project wizard state
  const [showWizard, setShowWizard] = useState(false);
  const [wizStep, setWizStep] = useState<'name' | 'platform' | 'platform-mode' | 'qr' | 'form' | 'done'>('name');
  const [newProjName, setNewProjName] = useState('');
  const [newWorkDir, setNewWorkDir] = useState('');
  const [newAgentType, setNewAgentType] = useState('claudecode');
  const [newAgentOptions, setNewAgentOptions] = useState<Record<string, string>>({});
  const [difyInputs, setDifyInputs] = useState<DifyInputRow[]>([]);
  const [agentTypeOptions, setAgentTypeOptions] = useState<string[]>([]);
  const [selectedPlat, setSelectedPlat] = useState('');
  const [showRestartModal, setShowRestartModal] = useState(false);

  const fetch = useCallback(async () => {
    try {
      setLoading(true);
      const [projectsRes, agentTypesRes] = await Promise.allSettled([
        listProjects(),
        listAgentTypes(),
      ]);
      if (projectsRes.status === 'fulfilled') {
        setProjects(projectsRes.value.projects || []);
      }
      if (agentTypesRes.status === 'fulfilled') {
        const fromAPI = (agentTypesRes.value.agents || []).map((s) => s.trim()).filter(Boolean);
        setAgentTypeOptions(fromAPI.length > 0 ? fromAPI.sort() : FALLBACK_AGENT_KEYS);
      } else {
        setAgentTypeOptions(FALLBACK_AGENT_KEYS);
      }
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetch();
    const handler = () => fetch();
    window.addEventListener('cc:refresh', handler);
    return () => window.removeEventListener('cc:refresh', handler);
  }, [fetch]);

  const openWizard = () => {
    setShowWizard(true);
    setWizStep('name');
    setNewProjName('');
    setNewWorkDir('');
    setNewAgentType('claudecode');
    setNewAgentOptions({});
    setDifyInputs([]);
    setSelectedPlat('');
  };

  useEffect(() => {
    if (newAgentType !== 'dify') {
      setNewAgentOptions({});
      setDifyInputs([]);
      return;
    }
    setNewAgentOptions((prev) => ({
      base_url: prev.base_url || 'https://api.dify.ai/v1',
      api_key: prev.api_key || '',
      app_mode: prev.app_mode || 'advanced-chat',
      user: prev.user || `cc-connect:${newProjName || 'project'}`,
      query_input_key: prev.query_input_key || 'query',
    }));
    setDifyInputs((prev) => (prev.length > 0 ? prev : [{ id: makeInputRowId(), key: '', value: '' }]));
  }, [newAgentType, newProjName]);

  const setAgentOption = (key: string, value: string) => {
    setNewAgentOptions((prev) => ({ ...prev, [key]: value }));
  };

  const updateDifyInput = (id: string, patch: Partial<DifyInputRow>) => {
    setDifyInputs((prev) => prev.map((row) => (row.id === id ? { ...row, ...patch } : row)));
  };

  const addDifyInput = () => {
    setDifyInputs((prev) => [...prev, { id: makeInputRowId(), key: '', value: '' }]);
  };

  const removeDifyInput = (id: string) => {
    setDifyInputs((prev) => {
      if (prev.length <= 1) {
        return [{ id: makeInputRowId(), key: '', value: '' }];
      }
      return prev.filter((row) => row.id !== id);
    });
  };

  const buildWizardAgentOptions = (): Record<string, any> => {
    if (newAgentType !== 'dify') return {};
    const opts: Record<string, any> = {};
    const baseURL = (newAgentOptions.base_url || '').trim();
    const apiKey = (newAgentOptions.api_key || '').trim();
    const appMode = (newAgentOptions.app_mode || '').trim();
    const user = (newAgentOptions.user || '').trim();
    const queryInputKey = (newAgentOptions.query_input_key || '').trim();
    if (baseURL) opts.base_url = baseURL;
    if (apiKey) opts.api_key = apiKey;
    if (appMode) opts.app_mode = appMode;
    if (user) opts.user = user;
    if (queryInputKey) opts.query_input_key = queryInputKey;
    const inputs: Record<string, string> = {};
    for (const row of difyInputs) {
      const k = row.key.trim();
      if (!k) continue;
      inputs[k] = row.value;
    }
    if (Object.keys(inputs).length > 0) opts.inputs = inputs;
    return opts;
  };

  const agentOptionsForWizard = buildWizardAgentOptions();
  const effectiveAgentTypes = (agentTypeOptions.length > 0 ? agentTypeOptions : FALLBACK_AGENT_KEYS);
  const agentDropdownOptions = effectiveAgentTypes.map((key) => ({ key, label: AGENT_LABELS[key] || key }));

  const supportsQRPlatform = (type: string) => type === 'feishu' || type === 'lark' || type === 'weixin';
  const supportsManualPlatform = (type: string) => !!platformMeta[type];
  const hasBothPlatformModes = (type: string) => supportsQRPlatform(type) && supportsManualPlatform(type);

  const handlePlatformSelect = (key: string) => {
    setSelectedPlat(key);
    if (hasBothPlatformModes(key)) {
      setWizStep('platform-mode');
    } else if (supportsQRPlatform(key)) {
      setWizStep('qr');
    } else if (supportsManualPlatform(key)) {
      setWizStep('form');
    } else {
      setWizStep('done');
    }
  };

  const handleQRComplete = (restarted?: boolean) => {
    setShowWizard(false);
    if (restarted) {
      fetch();
      return;
    }
    setShowRestartModal(true);
  };

  const waitForService = (maxMs: number) =>
    new Promise<void>((resolve) => {
      const start = Date.now();
      const poll = () => {
        window.fetch('/api/v1/status')
          .then((r) => { if (r.ok) resolve(); else throw new Error(); })
          .catch(() => {
            if (Date.now() - start > maxMs) { resolve(); return; }
            setTimeout(poll, 500);
          });
      };
      setTimeout(poll, 1500);
    });

  if (loading && projects.length === 0) {
    return <div className="flex items-center justify-center h-64 text-gray-400 animate-pulse">Loading...</div>;
  }

  return (
    <div className="animate-fade-in space-y-4 ">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-bold text-gray-900 dark:text-white">{t('projects.title')}</h2>
        <Button size="sm" onClick={openWizard}>
          <Plus size={14} /> {t('setup.addProject', 'Add project')}
        </Button>
      </div>

      {projects.length === 0 ? (
        <EmptyState message={t('projects.noProjects')} icon={FolderKanban} />
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
          {projects.map((p) => (
            <Link key={p.name} to={`/projects/${p.name}`}>
              <Card hover className="h-full flex flex-col">
                <div className="flex items-start justify-between mb-3">
                  <div className="flex items-center gap-2">
                    <Server size={18} className="text-gray-400" />
                    <h3 className="font-semibold text-gray-900 dark:text-white">{p.name}</h3>
                  </div>
                  <ArrowRight size={16} className="text-gray-300 dark:text-gray-600" />
                </div>
                <div className="flex flex-wrap gap-1.5 mb-3">
                  <Badge variant="info">{p.agent_type}</Badge>
                  {p.platforms?.slice(0, 3).map((pl) => <Badge key={pl}>{pl}</Badge>)}
                  {(p.platforms?.length ?? 0) > 3 && (
                    <Badge>+{p.platforms!.length - 3}</Badge>
                  )}
                </div>
                <div className="flex items-center justify-between text-xs text-gray-500 dark:text-gray-400 mt-auto pt-3 border-t border-gray-100 dark:border-gray-800">
                  <span>{p.sessions_count} {t('nav.sessions').toLowerCase()}</span>
                  {p.heartbeat_enabled && (
                    <span className="flex items-center gap-1 text-emerald-500"><Heart size={12} /> {t('heartbeat.title')}</span>
                  )}
                </div>
              </Card>
            </Link>
          ))}
        </div>
      )}

      {/* Add Project Wizard Modal */}
      <Modal
        open={showWizard}
        onClose={() => setShowWizard(false)}
        title={t('setup.addProject', 'Add project')}
      >
        {wizStep === 'name' && (
          <div className="space-y-4 py-2">
            <Input
              label={t('setup.projectName', 'Project name')}
              value={newProjName}
              onChange={(e) => setNewProjName(e.target.value.replace(/[^a-zA-Z0-9_-]/g, ''))}
              placeholder="my-project"
              autoFocus
            />
            <Input
              label={t('setup.workDir', 'Working directory')}
              value={newWorkDir}
              onChange={(e) => setNewWorkDir(e.target.value)}
              placeholder="/path/to/project"
            />
            <div>
              <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">
                {t('setup.agentType', 'Agent type')}
              </label>
              <select
                value={newAgentType}
                onChange={(e) => setNewAgentType(e.target.value)}
                className="w-full px-3 py-2 text-sm rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-accent/50"
              >
                {agentDropdownOptions.map(a => (
                  <option key={a.key} value={a.key}>{a.label}</option>
                ))}
              </select>
            </div>
            {newAgentType === 'dify' && (
              <div className="space-y-3 rounded-lg border border-gray-200 dark:border-gray-700 p-3">
                <div className="text-xs font-medium text-gray-600 dark:text-gray-400">Dify Options</div>
                <Input
                  label="base_url"
                  value={newAgentOptions.base_url || ''}
                  onChange={(e) => setAgentOption('base_url', e.target.value)}
                  placeholder="https://api.dify.ai/v1"
                />
                <Input
                  label="api_key"
                  type="password"
                  value={newAgentOptions.api_key || ''}
                  onChange={(e) => setAgentOption('api_key', e.target.value)}
                  placeholder="app-***"
                />
                <div>
                  <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">app_mode</label>
                  <select
                    value={newAgentOptions.app_mode || 'advanced-chat'}
                    onChange={(e) => setAgentOption('app_mode', e.target.value)}
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
                  value={newAgentOptions.user || ''}
                  onChange={(e) => setAgentOption('user', e.target.value)}
                  placeholder="cc-connect:my-backend"
                />
                <Input
                  label="query_input_key"
                  value={newAgentOptions.query_input_key || ''}
                  onChange={(e) => setAgentOption('query_input_key', e.target.value)}
                  placeholder="query"
                />
                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <label className="block text-xs font-medium text-gray-600 dark:text-gray-400">inputs (dynamic)</label>
                    <Button size="sm" variant="secondary" onClick={addDifyInput}>
                      <Plus size={12} /> Add
                    </Button>
                  </div>
                  <div className="space-y-2">
                    {difyInputs.map((row) => (
                      <div key={row.id} className="grid grid-cols-[1fr_1fr_auto] gap-2 items-end">
                        <Input
                          label="key"
                          value={row.key}
                          onChange={(e) => updateDifyInput(row.id, { key: e.target.value })}
                          placeholder="tenant"
                        />
                        <Input
                          label="value"
                          value={row.value}
                          onChange={(e) => updateDifyInput(row.id, { value: e.target.value })}
                          placeholder="ops"
                        />
                        <button
                          type="button"
                          onClick={() => removeDifyInput(row.id)}
                          className="h-9 w-9 mb-[1px] rounded-lg border border-gray-300 dark:border-gray-700 text-gray-500 hover:text-red-500 hover:border-red-300"
                          aria-label="remove input row"
                        >
                          <Trash2 size={14} className="mx-auto" />
                        </button>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            )}
            <div className="flex justify-end gap-2 pt-2">
              <Button variant="secondary" onClick={() => setShowWizard(false)}>{t('common.cancel')}</Button>
              <Button disabled={!newProjName.trim() || !newWorkDir.trim()} onClick={() => setWizStep('platform')}>
                {t('setup.next', 'Next')}
              </Button>
            </div>
          </div>
        )}

        {wizStep === 'platform' && (
          <div className="space-y-3 py-2">
            <p className="text-sm text-gray-500 dark:text-gray-400 mb-2">
              {t('setup.choosePlatform', 'Choose a platform to connect:')}
            </p>
            <div className="grid grid-cols-2 gap-2 max-h-80 overflow-y-auto">
              {PLATFORM_OPTIONS.map(({ key, label, color, qr }) => (
                <button
                  key={key}
                  onClick={() => handlePlatformSelect(key)}
                  className="flex items-center gap-2.5 p-3 rounded-xl border border-gray-200 dark:border-gray-700 hover:border-accent/50 hover:bg-accent/5 transition-all text-left"
                >
                  <div className={`w-9 h-9 rounded-lg ${color} flex items-center justify-center shrink-0`}>
                    {qr ? <Smartphone size={16} /> : <Settings2 size={16} />}
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
            <div className="flex justify-start pt-2">
              <Button variant="secondary" size="sm" onClick={() => setWizStep('name')}>{t('common.back')}</Button>
            </div>
          </div>
        )}

        {wizStep === 'platform-mode' && selectedPlat && (
          <div className="space-y-3 py-2">
            <p className="text-sm text-gray-500 dark:text-gray-400 mb-2">
              {t('setup.chooseConnectionMode', 'Choose connection mode:')}
            </p>
            <div className="grid grid-cols-1 gap-2">
              <button
                onClick={() => setWizStep('qr')}
                className="flex items-center justify-between p-3 rounded-xl border border-gray-200 dark:border-gray-700 hover:border-accent/50 hover:bg-accent/5 transition-all text-left"
              >
                <div>
                  <div className="text-sm font-medium text-gray-900 dark:text-white">{t('setup.scanToConnect', 'Scan QR code')}</div>
                  <div className="text-[11px] text-gray-400">{t('setup.scanModeHint', 'Connect quickly by scanning on phone')}</div>
                </div>
                <Smartphone size={16} className="text-gray-400" />
              </button>
              <button
                onClick={() => setWizStep('form')}
                className="flex items-center justify-between p-3 rounded-xl border border-gray-200 dark:border-gray-700 hover:border-accent/50 hover:bg-accent/5 transition-all text-left"
              >
                <div>
                  <div className="text-sm font-medium text-gray-900 dark:text-white">{t('setup.manualSetup', 'Manual setup')}</div>
                  <div className="text-[11px] text-gray-400">{t('setup.manualModeHint', 'Fill in app credentials manually')}</div>
                </div>
                <Settings2 size={16} className="text-gray-400" />
              </button>
            </div>
            <div className="flex justify-start pt-2">
              <Button variant="secondary" size="sm" onClick={() => setWizStep('platform')}>{t('common.back')}</Button>
            </div>
          </div>
        )}

        {wizStep === 'qr' && supportsQRPlatform(selectedPlat) && (
          <PlatformSetupQR
            platformType={selectedPlat as 'feishu' | 'weixin'}
            projectName={newProjName}
            workDir={newWorkDir}
            agentType={newAgentType}
            agentOptions={agentOptionsForWizard}
            onComplete={handleQRComplete}
            onCancel={() => setWizStep(hasBothPlatformModes(selectedPlat) ? 'platform-mode' : 'platform')}
          />
        )}

        {wizStep === 'form' && platformMeta[selectedPlat] && (
          <PlatformManualForm
            platformType={selectedPlat}
            projectName={newProjName}
            workDir={newWorkDir}
            agentType={newAgentType}
            agentOptions={agentOptionsForWizard}
            onComplete={() => {
              setShowWizard(false);
              setShowRestartModal(true);
            }}
            onCancel={() => setWizStep(hasBothPlatformModes(selectedPlat) ? 'platform-mode' : 'platform')}
          />
        )}

        {wizStep === 'done' && !supportsQRPlatform(selectedPlat) && (
          <div className="space-y-4 py-4 text-center">
            <Settings2 size={40} className="mx-auto text-gray-400" />
            <p className="text-sm text-gray-600 dark:text-gray-400">
              {t('setup.manualHint', 'For {{platform}}, please configure credentials in config.toml or via the project detail page after creating the project.', { platform: PLATFORM_OPTIONS.find(o => o.key === selectedPlat)?.label || selectedPlat })}
            </p>
            <div className="flex justify-center gap-2">
              <Button variant="secondary" onClick={() => setWizStep('platform')}>{t('common.back')}</Button>
            </div>
          </div>
        )}
      </Modal>

      <Modal open={showRestartModal} onClose={() => setShowRestartModal(false)} title={t('setup.restartRequired', 'Restart required')}>
        <div className="space-y-4 py-2">
          <p className="text-sm text-gray-600 dark:text-gray-400">
            {t('setup.restartHint', 'Restart the service for the new platform to take effect.')}
          </p>
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => { setShowRestartModal(false); setTimeout(fetch, 300); }}>
              {t('setup.later', 'Later')}
            </Button>
            <Button onClick={async () => { await restartSystem(); setShowRestartModal(false); await waitForService(8000); await fetch(); }}>
              {t('setup.restartNow', 'Restart now')}
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
