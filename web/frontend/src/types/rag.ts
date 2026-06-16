export type Citation = {
  chunk_id?: string;
  document_id?: string;
  source?: string;
  score?: number;
  content?: string;
  title_path?: string;
};

export type AgentPlanStep = {
  id?: string;
  name?: string;
  tool?: string;
  args?: Record<string, string>;
  rationale?: string;
  status?: string;
  depends_on?: string[];
  error?: string;
};

export type AgentPlan = {
  goal?: string;
  status?: string;
  steps?: AgentPlanStep[];
  created_at?: string;
  updated_at?: string;
};

export type AgentIteration = {
  index?: number;
  phase?: string;
  step_id?: string;
  tool?: string;
  observation?: string;
  replan_reason?: string;
  started_at?: string;
  ended_at?: string;
};

export type WorkflowStep = {
  name?: string;
  tool?: string;
  status?: string;
  summary?: string;
  error?: string;
  started_at?: string;
  ended_at?: string;
  trace_id?: string;
};

export type ChatMessage = {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  citations?: Citation[];
  plan?: AgentPlan;
  iterations?: AgentIteration[];
  steps?: WorkflowStep[];
  traceId?: string;
  createdAt: string;
};

export type ChatAnswer = {
  trace_id?: string;
  answer: string;
  sources: string[];
  citations: Citation[];
  mock?: boolean;
  plan?: AgentPlan;
  iterations?: AgentIteration[];
  steps?: WorkflowStep[];
};

export type StreamChunk = {
  index?: number;
  delta?: string;
  done?: boolean;
};

export type DocumentItem = {
  id: string;
  name: string;
  path?: string;
  metadata?: Record<string, string>;
  created_at?: string;
  chunk_count?: number;
};

export type SearchResult = {
  score: number;
  source?: string;
  title_path?: string;
  chunk: {
    id: string;
    document_id: string;
    content: string;
    index?: number;
    metadata?: Record<string, string>;
  };
};

export type UploadResult = {
  trace_id?: string;
  file_name: string;
  file_type: string;
  chunk_count: number;
  doc_id: string;
  next_steps?: string[];
  mock?: boolean;
  plan?: AgentPlan;
  iterations?: AgentIteration[];
  steps?: WorkflowStep[];
};

export type KnowledgeSearchResponse = {
  trace_id?: string;
  results: SearchResult[];
  plan?: AgentPlan;
  iterations?: AgentIteration[];
  steps?: WorkflowStep[];
};
