import { RefreshCw } from 'lucide-react';
import { useEffect, useState } from 'react';
import { API_BASE_URL } from '../api/client';
import { getHealth } from '../api/health';
import { ErrorBanner } from '../components/ErrorBanner';
import { PageHeader } from '../components/PageHeader';
import { StatusBadge } from '../components/StatusBadge';
import { TraceId } from '../components/TraceId';
import type { HealthStatus } from '../types/api';

export function SettingsPage() {
  const [health, setHealth] = useState<HealthStatus | null>(null);
  const [traceId, setTraceId] = useState<string | undefined>();
  const [error, setError] = useState<Error | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    void refresh();
  }, []);

  async function refresh() {
    setError(null);
    setLoading(true);
    try {
      const response = await getHealth();
      setHealth(response.data);
      setTraceId(response.traceId);
    } catch (err) {
      setError(err instanceof Error ? err : new Error('health check failed'));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="page">
      <PageHeader
        title="Settings"
        description="只展示非敏感运行状态和前端配置。"
        actions={
          <>
            <TraceId value={traceId} />
            <button className="button ghost" disabled={loading} type="button" onClick={() => void refresh()}>
              <RefreshCw size={16} /> 刷新
            </button>
          </>
        }
      />
      <ErrorBanner error={error} />
      <section className="panel settings-list">
        <div><span>API base URL</span><code>{API_BASE_URL}</code></div>
        <div><span>frontend version</span><code>0.1.0</code></div>
        <div><span>health</span><StatusBadge value={health?.status ?? 'unknown'} /></div>
        <div><span>env</span><code>{health?.env ?? '-'}</code></div>
        <div><span>mock</span><StatusBadge value={Boolean(health?.mock)} /></div>
        <div><span>rag embedder</span><code>{health?.rag?.embedder_provider ?? '-'}</code></div>
        <div><span>vector store</span><code>{health?.rag?.vector_store_provider ?? '-'}</code></div>
      </section>
    </div>
  );
}
