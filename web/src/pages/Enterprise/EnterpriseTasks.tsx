import { useCallback, useEffect, useMemo, useState } from 'react';
import { AlertCircle, CalendarClock, ListTodo, Plus, RefreshCcw, Target } from 'lucide-react';
import { Badge, Button, Input, Textarea } from '@/components/ui';
import {
  createEnterpriseTask,
  listEnterpriseSpaces,
  listEnterpriseTasks,
  listEnterpriseTenants,
  listEnterpriseUsers,
  type EnterpriseSpace,
  type EnterpriseTask,
  type EnterpriseTenant,
  type EnterpriseUser,
} from '@/api/enterprise';
import { DataPill, EnterpriseHero, EnterprisePanel, Select, TinyTable } from './shared';
import { formatTime, parseCSV } from './utils';

const defaultTaskForm: Partial<EnterpriseTask> = {
  tenant_id: '',
  space_id: '',
  owner_user_id: '',
  assignee_user_id: '',
  parent_task_id: '',
  title: '',
  description: '',
  task_type: 'task',
  priority: 'medium',
  status: 'todo',
  tags: [],
};

function toISOStringInput(value: string) {
  if (!value) return '';
  return value.slice(0, 16);
}

export default function EnterpriseTasks() {
  const [tenants, setTenants] = useState<EnterpriseTenant[]>([]);
  const [users, setUsers] = useState<EnterpriseUser[]>([]);
  const [spaces, setSpaces] = useState<EnterpriseSpace[]>([]);
  const [tasks, setTasks] = useState<EnterpriseTask[]>([]);
  const [tenantFilter, setTenantFilter] = useState('');
  const [spaceFilter, setSpaceFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [priorityFilter, setPriorityFilter] = useState('');
  const [query, setQuery] = useState('');
  const [taskForm, setTaskForm] = useState(defaultTaskForm);
  const [tagsInput, setTagsInput] = useState('');
  const [dueAtInput, setDueAtInput] = useState('');
  const [reminderAtInput, setReminderAtInput] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const fetchData = useCallback(async () => {
    try {
      setLoading(true);
      setError('');
      const [tenantData, userData, spaceData, taskData] = await Promise.all([
        listEnterpriseTenants(),
        listEnterpriseUsers(tenantFilter ? { tenant_id: tenantFilter } : undefined),
        listEnterpriseSpaces(tenantFilter ? { tenant_id: tenantFilter } : undefined),
        listEnterpriseTasks({
          tenant_id: tenantFilter || undefined,
          space_id: spaceFilter || undefined,
          status: statusFilter || undefined,
          priority: priorityFilter || undefined,
          q: query || undefined,
        }),
      ]);
      setTenants(tenantData.tenants || []);
      setUsers(userData.users || []);
      setSpaces(spaceData.spaces || []);
      setTasks(taskData.tasks || []);
    } catch (e: any) {
      setError(e.message || '加载任务中心失败');
    } finally {
      setLoading(false);
    }
  }, [priorityFilter, query, spaceFilter, statusFilter, tenantFilter]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const stats = useMemo(() => {
    return {
      total: tasks.length,
      todo: tasks.filter((item) => (item.status || 'todo') === 'todo').length,
      inProgress: tasks.filter((item) => item.status === 'in_progress').length,
      done: tasks.filter((item) => item.status === 'done').length,
    };
  }, [tasks]);

  async function handleCreateTask() {
    if (!taskForm.title) {
      setError('任务标题不能为空');
      return;
    }
    try {
      setSaving(true);
      setError('');
      await createEnterpriseTask({
        ...taskForm,
        tags: parseCSV(tagsInput),
        due_at: dueAtInput ? new Date(dueAtInput).toISOString() : undefined,
        reminder_at: reminderAtInput ? new Date(reminderAtInput).toISOString() : undefined,
      });
      setTaskForm(defaultTaskForm);
      setTagsInput('');
      setDueAtInput('');
      setReminderAtInput('');
      await fetchData();
    } catch (e: any) {
      setError(e.message || '创建任务失败');
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="space-y-6 animate-fade-in">
      <EnterpriseHero
        eyebrow="任务中心"
        title="企业任务、目标与提醒统一管理"
        description="第一版任务中心已经支持任务创建、状态流转、优先级、租户/空间隔离和基础筛选，适合作为后续 AI 执行与调度的主入口。"
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
        <DataPill label="任务总数" value={stats.total} />
        <DataPill label="待办" value={stats.todo} />
        <DataPill label="进行中" value={stats.inProgress} />
        <DataPill label="已完成" value={stats.done} />
      </div>

      <div className="grid gap-6 xl:grid-cols-[0.95fr_1.05fr]">
        <EnterprisePanel
          title="创建任务"
          description="支持 task、goal、reminder 三类任务，后续可继续扩展子任务、批量更新和调度联动。"
          action={(
            <div className="inline-flex items-center gap-2 rounded-full border border-emerald-500/15 bg-emerald-500/10 px-3 py-1 text-xs font-medium text-emerald-700 dark:text-emerald-300">
              <Target size={13} /> Task Center v1
            </div>
          )}
        >
          <div className="grid gap-4 md:grid-cols-2">
            <Input
              label="任务标题"
              value={taskForm.title || ''}
              onChange={(e) => setTaskForm((prev) => ({ ...prev, title: e.target.value }))}
              placeholder="处理生产告警复盘"
            />
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">任务类型</label>
              <Select value={taskForm.task_type || 'task'} onChange={(e) => setTaskForm((prev) => ({ ...prev, task_type: e.target.value }))}>
                <option value="task">task</option>
                <option value="goal">goal</option>
                <option value="reminder">reminder</option>
              </Select>
            </div>
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">所属租户</label>
              <Select value={taskForm.tenant_id || ''} onChange={(e) => setTaskForm((prev) => ({ ...prev, tenant_id: e.target.value }))}>
                <option value="">不绑定租户</option>
                {tenants.map((tenant) => (
                  <option key={tenant.id} value={tenant.id}>{tenant.name}</option>
                ))}
              </Select>
            </div>
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">所属空间</label>
              <Select value={taskForm.space_id || ''} onChange={(e) => setTaskForm((prev) => ({ ...prev, space_id: e.target.value }))}>
                <option value="">不绑定空间</option>
                {spaces
                  .filter((space) => !taskForm.tenant_id || space.tenant_id === taskForm.tenant_id)
                  .map((space) => (
                    <option key={space.id} value={space.id}>{space.name}</option>
                  ))}
              </Select>
            </div>
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">创建人</label>
              <Select value={taskForm.owner_user_id || ''} onChange={(e) => setTaskForm((prev) => ({ ...prev, owner_user_id: e.target.value }))}>
                <option value="">不指定</option>
                {users
                  .filter((user) => !taskForm.tenant_id || user.tenant_id === taskForm.tenant_id)
                  .map((user) => (
                    <option key={user.id} value={user.id}>{user.display_name}</option>
                  ))}
              </Select>
            </div>
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">指派给</label>
              <Select value={taskForm.assignee_user_id || ''} onChange={(e) => setTaskForm((prev) => ({ ...prev, assignee_user_id: e.target.value }))}>
                <option value="">不指定</option>
                {users
                  .filter((user) => !taskForm.tenant_id || user.tenant_id === taskForm.tenant_id)
                  .map((user) => (
                    <option key={user.id} value={user.id}>{user.display_name}</option>
                  ))}
              </Select>
            </div>
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">优先级</label>
              <Select value={taskForm.priority || 'medium'} onChange={(e) => setTaskForm((prev) => ({ ...prev, priority: e.target.value }))}>
                <option value="urgent">urgent</option>
                <option value="high">high</option>
                <option value="medium">medium</option>
                <option value="low">low</option>
                <option value="none">none</option>
              </Select>
            </div>
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">状态</label>
              <Select value={taskForm.status || 'todo'} onChange={(e) => setTaskForm((prev) => ({ ...prev, status: e.target.value }))}>
                <option value="todo">todo</option>
                <option value="in_progress">in_progress</option>
                <option value="done">done</option>
                <option value="cancelled">cancelled</option>
              </Select>
            </div>
            <Input
              label="标签"
              value={tagsInput}
              onChange={(e) => setTagsInput(e.target.value)}
              placeholder="ops, alert, weekly"
            />
            <Input
              label="截止时间"
              type="datetime-local"
              value={dueAtInput}
              onChange={(e) => setDueAtInput(e.target.value)}
            />
            <Input
              label="提醒时间"
              type="datetime-local"
              value={reminderAtInput}
              onChange={(e) => setReminderAtInput(e.target.value)}
            />
          </div>

          <div className="mt-4">
            <Textarea
              label="任务描述"
              value={taskForm.description || ''}
              onChange={(e) => setTaskForm((prev) => ({ ...prev, description: e.target.value }))}
              placeholder="描述任务目标、执行要求和上下文信息。"
              rows={4}
            />
          </div>

          <div className="mt-4 flex items-center justify-between gap-3">
            <p className="text-xs text-slate-500 dark:text-slate-400">
              下一步我会继续把子任务、批量更新、定时执行和 AI 自动跟进加进来，这一版先把主链路做实。
            </p>
            <Button onClick={handleCreateTask} loading={saving}>
              <Plus size={15} /> 创建任务
            </Button>
          </div>
        </EnterprisePanel>

        <EnterprisePanel title="任务清单" description="按租户、空间、状态、优先级和关键词查看当前任务。">
          <div className="mb-4 grid gap-3 md:grid-cols-5">
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">租户</label>
              <Select value={tenantFilter} onChange={(e) => setTenantFilter(e.target.value)}>
                <option value="">全部</option>
                {tenants.map((tenant) => (
                  <option key={tenant.id} value={tenant.id}>{tenant.name}</option>
                ))}
              </Select>
            </div>
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">空间</label>
              <Select value={spaceFilter} onChange={(e) => setSpaceFilter(e.target.value)}>
                <option value="">全部</option>
                {spaces
                  .filter((space) => !tenantFilter || space.tenant_id === tenantFilter)
                  .map((space) => (
                    <option key={space.id} value={space.id}>{space.name}</option>
                  ))}
              </Select>
            </div>
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">状态</label>
              <Select value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)}>
                <option value="">全部</option>
                <option value="todo">todo</option>
                <option value="in_progress">in_progress</option>
                <option value="done">done</option>
                <option value="cancelled">cancelled</option>
              </Select>
            </div>
            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">优先级</label>
              <Select value={priorityFilter} onChange={(e) => setPriorityFilter(e.target.value)}>
                <option value="">全部</option>
                <option value="urgent">urgent</option>
                <option value="high">high</option>
                <option value="medium">medium</option>
                <option value="low">low</option>
                <option value="none">none</option>
              </Select>
            </div>
            <Input
              label="关键词"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="搜索标题或描述"
            />
          </div>

          <TinyTable>
            <thead className="bg-slate-50/90 text-left text-xs uppercase tracking-[0.18em] text-slate-400 dark:bg-white/[0.03]">
              <tr>
                <th className="px-4 py-3">任务</th>
                <th className="px-4 py-3">类型</th>
                <th className="px-4 py-3">负责人</th>
                <th className="px-4 py-3">优先级 / 状态</th>
                <th className="px-4 py-3">截止 / 提醒</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-200/80 dark:divide-white/[0.06]">
              {tasks.length === 0 ? (
                <tr>
                  <td colSpan={5} className="px-4 py-10 text-center text-sm text-slate-500 dark:text-slate-400">
                    还没有任务记录。
                  </td>
                </tr>
              ) : tasks.map((task) => {
                const assignee = users.find((user) => user.id === task.assignee_user_id);
                return (
                  <tr key={task.id}>
                    <td className="px-4 py-3">
                      <div>
                        <p className="text-sm font-medium text-slate-900 dark:text-white">{task.title}</p>
                        <p className="text-xs text-slate-400">{task.description || '-'}</p>
                        {(task.tags || []).length > 0 && (
                          <div className="mt-2 flex flex-wrap gap-2">
                            {task.tags?.map((tag) => (
                              <Badge key={`${task.id}-${tag}`} variant="outline">{tag}</Badge>
                            ))}
                          </div>
                        )}
                      </div>
                    </td>
                    <td className="px-4 py-3 text-sm">
                      <div className="flex flex-wrap gap-2">
                        <Badge variant="outline">{task.task_type || 'task'}</Badge>
                        {(task.task_type || 'task') === 'goal' && <Target size={14} className="text-emerald-500" />}
                        {(task.task_type || 'task') === 'reminder' && <CalendarClock size={14} className="text-amber-500" />}
                        {(task.task_type || 'task') === 'task' && <ListTodo size={14} className="text-sky-500" />}
                      </div>
                    </td>
                    <td className="px-4 py-3 text-sm text-slate-500 dark:text-slate-400">{assignee?.display_name || task.assignee_user_id || '-'}</td>
                    <td className="px-4 py-3 text-sm">
                      <div className="flex flex-wrap gap-2">
                        <Badge variant={(task.priority || 'medium') === 'urgent' || (task.priority || 'medium') === 'high' ? 'warning' : 'outline'}>
                          {task.priority || 'medium'}
                        </Badge>
                        <Badge variant={(task.status || 'todo') === 'done' ? 'success' : 'outline'}>
                          {task.status || 'todo'}
                        </Badge>
                      </div>
                    </td>
                    <td className="px-4 py-3 text-sm text-slate-500 dark:text-slate-400">
                      <div>{formatTime(task.due_at)}</div>
                      <div className="text-xs text-slate-400">{formatTime(task.reminder_at)}</div>
                    </td>
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
