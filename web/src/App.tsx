import { Routes, Route, Navigate } from 'react-router-dom';
import { useAuthStore } from '@/store/auth';
import Layout from '@/components/Layout/Layout';
import Login from '@/pages/Login';
import Dashboard from '@/pages/Dashboard';
import ProjectList from '@/pages/Projects/ProjectList';
import ProjectDetail from '@/pages/Projects/ProjectDetail';
import ChatList from '@/pages/Chat/ChatList';
import ChatView from '@/pages/Chat/ChatView';
import CronList from '@/pages/Cron/CronList';
import SystemConfig from '@/pages/System/Config';
import ProviderList from '@/pages/Providers/ProviderList';
import SkillList from '@/pages/Skills/SkillList';
import EnterpriseOverview from '@/pages/Enterprise/EnterpriseOverview';
import EnterpriseSpaces from '@/pages/Enterprise/EnterpriseSpaces';
import EnterpriseBots from '@/pages/Enterprise/EnterpriseBots';
import EnterpriseSkills from '@/pages/Enterprise/EnterpriseSkills';
import EnterpriseAnalytics from '@/pages/Enterprise/EnterpriseAnalytics';
import EnterpriseRBAC from '@/pages/Enterprise/EnterpriseRBAC';
import EnterpriseProjects from '@/pages/Enterprise/EnterpriseProjects';
import EnterpriseTasks from '@/pages/Enterprise/EnterpriseTasks';

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  if (!isAuthenticated) return <Navigate to="/login" replace />;
  return <>{children}</>;
}

export default function App() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);

  return (
    <Routes>
      <Route path="/login" element={isAuthenticated ? <Navigate to="/" replace /> : <Login />} />
      <Route element={<ProtectedRoute><Layout /></ProtectedRoute>}>
        <Route index element={<Dashboard />} />
        <Route path="projects" element={<ProjectList />} />
        <Route path="projects/:name" element={<ProjectDetail />} />
        <Route path="providers" element={<ProviderList />} />
        <Route path="skills" element={<SkillList />} />
        <Route path="enterprise" element={<EnterpriseOverview />} />
        <Route path="enterprise/spaces" element={<EnterpriseSpaces />} />
        <Route path="enterprise/bots" element={<EnterpriseBots />} />
        <Route path="enterprise/skills" element={<EnterpriseSkills />} />
        <Route path="enterprise/tasks" element={<EnterpriseTasks />} />
        <Route path="enterprise/rbac" element={<EnterpriseRBAC />} />
        <Route path="enterprise/projects" element={<EnterpriseProjects />} />
        <Route path="enterprise/analytics" element={<EnterpriseAnalytics />} />
        <Route path="chat" element={<ChatList />} />
        <Route path="chat/:name" element={<ChatView />} />
        <Route path="cron" element={<CronList />} />
        <Route path="system" element={<SystemConfig />} />
      </Route>
    </Routes>
  );
}
