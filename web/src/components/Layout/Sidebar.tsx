import { NavLink } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  LayoutDashboard,
  FolderKanban,
  MessageSquare,
  Clock,
  Settings,
  ChevronLeft,
  ChevronRight,
  Plug,
  Puzzle,
  Building2,
  Bot,
  Sparkles,
  BarChart3,
  Shield,
  FolderTree,
  ListChecks,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { useState } from 'react';

const navItems = [
  { key: 'dashboard', path: '/', icon: LayoutDashboard },
  { key: 'enterprise', path: '/enterprise', icon: Building2, label: '\u4f01\u4e1a\u5e73\u53f0' },
  { key: 'spaces', path: '/enterprise/spaces', icon: FolderKanban, label: '\u7528\u6237\u7a7a\u95f4' },
  { key: 'bots', path: '/enterprise/bots', icon: Bot, label: '\u5171\u4eab\u673a\u5668\u4eba' },
  { key: 'skillStudio', path: '/enterprise/skills', icon: Sparkles, label: '\u6280\u80fd\u5de5\u574a' },
  { key: 'taskCenter', path: '/enterprise/tasks', icon: ListChecks, label: '\u4efb\u52a1\u4e2d\u5fc3' },
  { key: 'rbac', path: '/enterprise/rbac', icon: Shield, label: '\u89d2\u8272\u6743\u9650' },
  { key: 'enterpriseProjects', path: '/enterprise/projects', icon: FolderTree, label: '\u9879\u76ee\u6863\u6848' },
  { key: 'analytics', path: '/enterprise/analytics', icon: BarChart3, label: '\u7edf\u8ba1\u5206\u6790' },
  { key: 'projects', path: '/projects', icon: FolderKanban },
  { key: 'providers', path: '/providers', icon: Plug },
  { key: 'skills', path: '/skills', icon: Puzzle },
  { key: 'chat', path: '/chat', icon: MessageSquare },
  { key: 'cron', path: '/cron', icon: Clock },
  { key: 'system', path: '/system', icon: Settings },
];

export default function Sidebar() {
  const { t } = useTranslation();
  const [collapsed, setCollapsed] = useState(false);

  return (
    <aside
      className={cn(
        'h-screen flex flex-col border-r transition-all duration-300 ease-out',
        'bg-white/75 backdrop-blur-xl border-gray-200/80',
        'dark:bg-[rgba(0,0,0,0.85)] dark:backdrop-blur-xl dark:border-white/[0.08]',
        collapsed ? 'w-16' : 'w-56',
      )}
    >
      <div
        className={cn(
          'flex items-center px-4 h-14 border-b transition-colors shrink-0',
          'border-gray-200/80 dark:border-white/[0.08]',
          collapsed ? 'justify-center' : 'gap-0',
        )}
      >
        {collapsed ? (
          <span className="text-base font-bold tracking-tighter text-gray-900 dark:text-white">
            {'\u7384\u661f'}
          </span>
        ) : (
          <span className="text-base font-bold tracking-tight text-gray-900 dark:text-white">
            {'\u7384\u661f'}<span className="text-accent">Ops</span>
          </span>
        )}
      </div>

      <nav className="flex-1 py-4 space-y-1 px-2 overflow-y-auto">
        {navItems.map(({ key, path, icon: Icon, label }) => (
          <NavLink
            key={key}
            to={path}
            end={path === '/'}
            className={({ isActive }) =>
              cn(
                'flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-all duration-200',
                isActive
                  ? 'bg-accent/12 text-accent ring-1 ring-accent/25'
                  : 'text-gray-600 dark:text-gray-400 hover:bg-gray-100/80 dark:hover:bg-white/[0.06] hover:text-gray-900 dark:hover:text-white',
              )
            }
          >
            <Icon size={18} className="shrink-0" />
            {!collapsed && <span>{label || t(`nav.${key}`)}</span>}
          </NavLink>
        ))}
      </nav>

      <div className={cn('border-t p-2', 'border-gray-200/80 dark:border-white/[0.08]')}>
        <button
          type="button"
          onClick={() => setCollapsed(!collapsed)}
          className={cn(
            'flex items-center justify-center w-full px-3 py-2 rounded-xl transition-colors duration-200',
            'text-gray-400 hover:bg-gray-100/80 dark:hover:bg-white/[0.06]',
          )}
        >
          {collapsed ? <ChevronRight size={18} /> : <ChevronLeft size={18} />}
        </button>
      </div>
    </aside>
  );
}
