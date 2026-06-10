import { Copy, Download, Trash2 } from 'lucide-react';
import { useMemo, useState } from 'react';
import { deleteLocalReport, listLocalReports } from '../api/reports';
import { EmptyState } from '../components/EmptyState';
import { MarkdownRenderer } from '../components/MarkdownRenderer';
import { PageHeader } from '../components/PageHeader';
import { StatusBadge } from '../components/StatusBadge';
import { TraceId } from '../components/TraceId';
import type { StoredReport } from '../types/aiops';
import { copyText, downloadText, formatDate } from '../utils/format';

export function ReportsPage() {
  const [reports, setReports] = useState<StoredReport[]>(() => listLocalReports());
  const [selectedId, setSelectedId] = useState<string | undefined>(reports[0]?.id);
  const selected = useMemo(() => reports.find((item) => item.id === selectedId) ?? reports[0], [reports, selectedId]);

  function remove(id: string) {
    const next = deleteLocalReport(id);
    setReports(next);
    if (selectedId === id) setSelectedId(next[0]?.id);
  }

  return (
    <div className="page">
      <PageHeader title="Reports" description="本地保存最近 AI Ops 分析报告。" actions={selected ? <TraceId value={selected.trace_id} /> : null} />
      <section className="grid reports-grid">
        <div className="panel">
          <div className="section-title">
            <h2>历史报告</h2>
            <span className="muted">{reports.length} reports</span>
          </div>
          {reports.length === 0 ? (
            <EmptyState title="暂无历史报告" description="AI Ops 分析完成后会自动保存到浏览器 localStorage。" />
          ) : (
            <div className="report-list">
              {reports.map((report) => (
                <button
                  className={selected?.id === report.id ? 'report-row active' : 'report-row'}
                  key={report.id}
                  type="button"
                  onClick={() => setSelectedId(report.id)}
                >
                  <strong>{report.alert_summary}</strong>
                  <span>{formatDate(report.created_at)}</span>
                  <span>{report.mode || '-'} · fallback {report.fallback_used ? 'yes' : 'no'}</span>
                </button>
              ))}
            </div>
          )}
        </div>
        <div className="panel">
          {!selected ? (
            <EmptyState title="请选择报告" />
          ) : (
            <>
              <div className="section-title">
                <div>
                  <h2>{selected.alert_summary}</h2>
                  <div className="meta-line">{formatDate(selected.created_at)}</div>
                </div>
                <div className="inline-actions">
                  <StatusBadge value={selected.mode} kind="mode" />
                  <button className="button ghost" type="button" onClick={() => void copyText(selected.result.report)}>
                    <Copy size={16} /> 复制
                  </button>
                  <button
                    className="button ghost"
                    type="button"
                    onClick={() => downloadText(`aiops-report-${selected.trace_id || selected.id}.md`, selected.result.report)}
                  >
                    <Download size={16} /> 下载
                  </button>
                  <button className="icon-button danger" type="button" title="删除" onClick={() => remove(selected.id)}>
                    <Trash2 size={16} />
                  </button>
                </div>
              </div>
              <MarkdownRenderer content={selected.result.report} />
            </>
          )}
        </div>
      </section>
    </div>
  );
}
