import { Copy, Download, Play } from 'lucide-react';
import { useState } from 'react';
import { analyzeAIOps } from '../api/aiops';
import { CitationList } from '../components/CitationList';
import { EmptyState } from '../components/EmptyState';
import { ErrorBanner } from '../components/ErrorBanner';
import { EvidencePanel } from '../components/EvidencePanel';
import { MarkdownRenderer } from '../components/MarkdownRenderer';
import { PageHeader } from '../components/PageHeader';
import { StatusBadge } from '../components/StatusBadge';
import { StepTimeline } from '../components/StepTimeline';
import { TraceId } from '../components/TraceId';
import type { AIOpsAnalyzeResult } from '../types/aiops';
import { copyText, downloadText, formatDate } from '../utils/format';
import { addReport } from '../utils/storage';

export function AIOpsPage() {
  const [alertName, setAlertName] = useState('服务下线');
  const [service, setService] = useState('billing-service');
  const [modeHint, setModeHint] = useState('server-config');
  const [result, setResult] = useState<AIOpsAnalyzeResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  async function analyze() {
    setError(null);
    setLoading(true);
    try {
      const response = await analyzeAIOps({ alert_name: alertName, service });
      const data = { ...response.data, trace_id: response.data.trace_id || response.traceId };
      setResult(data);
      addReport(data);
    } catch (err) {
      setError(err instanceof Error ? err : new Error('aiops analyze failed'));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="page">
      <PageHeader
        title="AI Ops"
        description="触发告警分析并查看 workflow、evidence、citations 和 Markdown 报告。"
        actions={result ? <TraceId value={result.trace_id} /> : null}
      />
      <ErrorBanner error={error} />
      <section className="panel control-panel">
        <div className="form-row aiops-form">
          <label>
            <span>alert</span>
            <input value={alertName} onChange={(event) => setAlertName(event.target.value)} />
          </label>
          <label>
            <span>service</span>
            <input value={service} onChange={(event) => setService(event.target.value)} />
          </label>
          <label>
            <span>mode</span>
            <select value={modeHint} onChange={(event) => setModeHint(event.target.value)}>
              <option value="server-config">server-config</option>
              <option value="rule" disabled>rule by config</option>
              <option value="agent" disabled>agent by config</option>
            </select>
          </label>
          <button className="button primary" disabled={loading} type="button" onClick={() => void analyze()}>
            <Play size={16} /> {loading ? '分析中...' : '触发分析'}
          </button>
        </div>
        <div className="muted small">当前后端 `/api/aiops/analyze` 未暴露 mode 请求字段，页面展示服务端配置返回的 mode。</div>
      </section>

      {!result ? (
        <section className="panel">
          <EmptyState title="尚未触发分析" description="上传 SOP 后触发分析，可以看到 SOP citations、日志和指标证据。" />
        </section>
      ) : (
        <>
          {(result.fallback_used || result.steps?.some((step) => step.status === 'failed')) && (
            <div className="warning-banner">
              provider 或 agent 出现失败，报告已按当前 workflow 降级生成。
            </div>
          )}
          <section className="summary-strip">
            <div><span>mode</span><strong>{result.mode || '-'}</strong></div>
            <div><span>fallback</span><StatusBadge value={Boolean(result.fallback_used)} /></div>
            <div><span>alerts</span><strong>{result.alerts?.length ?? 0}</strong></div>
            <div><span>steps</span><strong>{result.steps?.length ?? 0}</strong></div>
            <div><span>evidence</span><strong>{result.evidence?.length ?? 0}</strong></div>
          </section>

          <section className="panel">
            <div className="section-title">
              <h2>Alerts</h2>
            </div>
            {result.alerts?.length ? (
              <div className="table-wrap">
                <table>
                  <thead>
                    <tr>
                      <th>name</th>
                      <th>service</th>
                      <th>severity</th>
                      <th>status</th>
                      <th>region</th>
                      <th>starts_at</th>
                    </tr>
                  </thead>
                  <tbody>
                    {result.alerts.map((alert, index) => (
                      <tr key={alert.id || index}>
                        <td>{alert.name || '-'}</td>
                        <td>{alert.service || '-'}</td>
                        <td><StatusBadge value={alert.severity} kind="severity" /></td>
                        <td><StatusBadge value={alert.status} /></td>
                        <td>{alert.region || '-'}</td>
                        <td>{formatDate(alert.starts_at)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <EmptyState title="无活跃告警" />
            )}
          </section>

          <section className="grid two">
            <div className="panel">
              <div className="section-title"><h2>Workflow Steps</h2></div>
              <StepTimeline steps={result.steps} />
            </div>
            <div className="panel">
              <div className="section-title"><h2>Citations</h2></div>
              <CitationList citations={result.citations} />
            </div>
          </section>

          <section className="panel">
            <div className="section-title"><h2>Evidence</h2></div>
            <EvidencePanel evidence={result.evidence} />
          </section>

          <section className="panel">
            <div className="section-title">
              <h2>Report</h2>
              <div className="inline-actions">
                <button className="button ghost" type="button" onClick={() => void copyText(result.report)}>
                  <Copy size={16} /> 复制
                </button>
                <button
                  className="button ghost"
                  type="button"
                  onClick={() => downloadText(`aiops-report-${result.trace_id || Date.now()}.md`, result.report)}
                >
                  <Download size={16} /> 下载
                </button>
              </div>
            </div>
            <MarkdownRenderer content={result.report} />
          </section>
        </>
      )}
    </div>
  );
}
