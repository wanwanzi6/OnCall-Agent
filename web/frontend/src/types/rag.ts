export type Citation = {
  chunk_id?: string;
  document_id?: string;
  source?: string;
  score?: number;
  content?: string;
  title_path?: string;
};

export type ChatMessage = {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  citations?: Citation[];
  traceId?: string;
  createdAt: string;
};

export type ChatAnswer = {
  answer: string;
  sources: string[];
  citations: Citation[];
  mock?: boolean;
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
  file_name: string;
  file_type: string;
  chunk_count: number;
  doc_id: string;
  next_steps?: string[];
  mock?: boolean;
};
