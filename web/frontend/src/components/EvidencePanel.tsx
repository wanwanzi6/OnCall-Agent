import type { Evidence } from '../types/aiops';
import { formatDate } from '../utils/format';
import { StatusBadge } from './StatusBadge';

type Props = {
  evidence?: Evidence[];
};

export function EvidencePanel({ evidence = [] }: Props) {
  if (evidence.length === 0) return <div className="muted small">无 evidence</div>;
  const groups = evidence.reduce<Record<string, Evidence[]>>((acc, item) => {
    const key = item.type || 'unknown';
    acc[key] = acc[key] ?? [];
    acc[key].push(item);
    return acc;
  }, {});

  return (
    <div className="evidence-panel">
      {Object.entries(groups).map(([type, items]) => (
        <section className="evidence-group" key={type}>
          <div className="section-title">
            <h3>{type}</h3>
            <StatusBadge value={`${items.length}`} kind="mode" />
          </div>
          {items.map((item, index) => (
            <article className="evidence-item" key={item.id || `${type}-${index}`}>
              <div className="row-between">
                <strong>{item.source || '-'}</strong>
                <span className="muted small">{formatDate(item.created_at)}</span>
              </div>
              {item.query ? <div className="meta-line">{item.query}</div> : null}
              <p>{item.summary || '-'}</p>
              {item.samples?.length ? (
                <pre>{item.samples.slice(0, 6).join('\n')}</pre>
              ) : item.logs?.length ? (
                <pre>{item.logs.slice(0, 6).join('\n')}</pre>
              ) : null}
            </article>
          ))}
        </section>
      ))}
    </div>
  );
}
