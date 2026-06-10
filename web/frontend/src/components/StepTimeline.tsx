import type { WorkflowStep } from '../types/aiops';
import { formatDuration } from '../utils/format';
import { StatusBadge } from './StatusBadge';

type Props = {
  steps?: WorkflowStep[];
};

export function StepTimeline({ steps = [] }: Props) {
  if (steps.length === 0) return <div className="muted small">无 workflow steps</div>;
  return (
    <div className="timeline">
      {steps.map((step, index) => (
        <div className="timeline-item" key={`${step.name}-${index}`}>
          <div className="timeline-dot" />
          <div className="timeline-content">
            <div className="row-between">
              <strong>{step.name || `step-${index + 1}`}</strong>
              <StatusBadge value={step.status} />
            </div>
            <p>{step.summary || '-'}</p>
            <div className="meta-line">duration: {formatDuration(step.started_at, step.ended_at)}</div>
            {step.error ? <pre className="inline-error">{step.error}</pre> : null}
          </div>
        </div>
      ))}
    </div>
  );
}
