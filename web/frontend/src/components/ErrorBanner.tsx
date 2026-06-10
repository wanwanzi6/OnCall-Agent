import { AlertTriangle } from 'lucide-react';
import type { ApiError } from '../api/client';
import { TraceId } from './TraceId';

type Props = {
  error?: Error | ApiError | null;
};

export function ErrorBanner({ error }: Props) {
  if (!error) return null;
  const traceId = 'traceId' in error ? error.traceId : undefined;
  return (
    <div className="error-banner" role="alert">
      <AlertTriangle size={18} />
      <div>
        <strong>{error.message || '请求失败'}</strong>
        <div>{traceId ? <TraceId value={traceId} /> : <span className="muted">trace_id: -</span>}</div>
      </div>
    </div>
  );
}
