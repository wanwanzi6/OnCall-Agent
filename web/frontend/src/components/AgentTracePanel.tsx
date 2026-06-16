import type { AgentIteration, AgentPlan, WorkflowStep } from '../types/rag';
import { formatDuration } from '../utils/format';
import { StatusBadge } from './StatusBadge';

type Props = {
  plan?: AgentPlan;
  iterations?: AgentIteration[];
  steps?: WorkflowStep[];
  replanReason?: string;
};

export function AgentTracePanel({ plan, iterations = [], steps = [], replanReason }: Props) {
  const hasTrace = plan || iterations.length > 0 || steps.length > 0 || replanReason;
  if (!hasTrace) return <div className="muted small">无 agent trace</div>;

  return (
    <div className="agent-trace">
      {plan ? (
        <div className="trace-block">
          <div className="row-between">
            <strong>{plan.goal || 'Agent Plan'}</strong>
            <StatusBadge value={plan.status} />
          </div>
          <div className="trace-step-list">
            {(plan.steps ?? []).map((step, index) => (
              <div className="trace-step" key={step.id || `${step.name}-${index}`}>
                <span>{step.name || step.id || `step-${index + 1}`}</span>
                <span className="muted">{step.tool || '-'}</span>
                <StatusBadge value={step.status} />
              </div>
            ))}
          </div>
        </div>
      ) : null}

      {replanReason ? (
        <div className="trace-block">
          <strong>Replan</strong>
          <p>{replanReason}</p>
        </div>
      ) : null}

      {iterations.length > 0 ? (
        <div className="trace-block">
          <strong>Iterations</strong>
          <div className="trace-step-list">
            {iterations.map((item, index) => (
              <div className="trace-row" key={`${item.phase}-${item.index ?? index}`}>
                <span>{item.index ?? index + 1}</span>
                <span>{item.phase || '-'}</span>
                <span className="muted">{item.tool || item.step_id || '-'}</span>
                <span>{item.observation || '-'}</span>
              </div>
            ))}
          </div>
        </div>
      ) : null}

      {steps.length > 0 ? (
        <div className="trace-block">
          <strong>Steps</strong>
          <div className="trace-step-list">
            {steps.map((step, index) => (
              <div className="trace-row" key={`${step.name}-${index}`}>
                <span>{step.name || `step-${index + 1}`}</span>
                <StatusBadge value={step.status} />
                <span className="muted">{formatDuration(step.started_at, step.ended_at)}</span>
                <span>{step.summary || step.error || '-'}</span>
              </div>
            ))}
          </div>
        </div>
      ) : null}
    </div>
  );
}

