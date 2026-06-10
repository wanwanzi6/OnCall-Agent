import type { Citation } from './rag';

export type Alert = {
  id?: string;
  name?: string;
  service?: string;
  severity?: string;
  status?: string;
  description?: string;
  region?: string;
  labels?: Record<string, string>;
  starts_at?: string;
};

export type WorkflowStep = {
  name?: string;
  status?: string;
  summary?: string;
  error?: string;
  started_at?: string;
  ended_at?: string;
  trace_id?: string;
};

export type Evidence = {
  id?: string;
  type?: string;
  source?: string;
  query?: string;
  summary?: string;
  samples?: string[];
  metadata?: Record<string, string>;
  created_at?: string;
  logs?: string[];
};

export type AIOpsAnalyzeResult = {
  trace_id?: string;
  report: string;
  alerts: Alert[];
  steps: WorkflowStep[];
  evidence: Evidence[];
  citations: Citation[];
  mode?: string;
  fallback_used?: boolean;
};

export type AIOpsAnalyzeRequest = {
  alert_name?: string;
  service?: string;
};

export type StoredReport = {
  id: string;
  created_at: string;
  trace_id?: string;
  alert_summary: string;
  mode?: string;
  fallback_used?: boolean;
  result: AIOpsAnalyzeResult;
};
