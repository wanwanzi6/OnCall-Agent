import { useState } from 'react';
import { Layout } from './components/Layout';
import type { PageKey } from './components/Sidebar';
import { AIOpsPage } from './pages/AIOpsPage';
import { ChatPage } from './pages/ChatPage';
import { KnowledgePage } from './pages/KnowledgePage';
import { ReportsPage } from './pages/ReportsPage';
import { SettingsPage } from './pages/SettingsPage';

export default function App() {
  const [page, setPage] = useState<PageKey>('knowledge');
  return (
    <Layout active={page} onChange={setPage}>
      {page === 'chat' && <ChatPage />}
      {page === 'knowledge' && <KnowledgePage />}
      {page === 'aiops' && <AIOpsPage />}
      {page === 'reports' && <ReportsPage />}
      {page === 'settings' && <SettingsPage />}
    </Layout>
  );
}
