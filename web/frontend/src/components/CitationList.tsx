import { useState } from 'react';
import type { Citation } from '../types/rag';
import { compactText } from '../utils/format';
import { StatusBadge } from './StatusBadge';

type Props = {
  citations?: Citation[];
};

export function CitationList({ citations = [] }: Props) {
  const [expanded, setExpanded] = useState<Record<string, boolean>>({});
  if (citations.length === 0) {
    return <div className="muted small">无 citations</div>;
  }
  return (
    <div className="citation-list">
      {citations.map((citation, index) => {
        const key = citation.chunk_id || `${citation.document_id}-${index}`;
        const open = expanded[key];
        return (
          <article className="citation-item" key={key}>
            <button
              className="citation-head"
              type="button"
              onClick={() => setExpanded((prev) => ({ ...prev, [key]: !prev[key] }))}
            >
              <span>{citation.source || citation.title_path || citation.document_id || 'unknown source'}</span>
              {typeof citation.score === 'number' ? <StatusBadge value={citation.score.toFixed(3)} kind="mode" /> : null}
            </button>
            <div className="citation-meta">
              {citation.title_path ? <span>{citation.title_path}</span> : null}
              {citation.document_id ? <span>doc: {citation.document_id}</span> : null}
            </div>
            <p>{open ? citation.content : compactText(citation.content ?? '', 180)}</p>
          </article>
        );
      })}
    </div>
  );
}
