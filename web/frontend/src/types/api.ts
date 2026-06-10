export type ApiEnvelope<T> = {
  code: number;
  message: string;
  data?: T;
  trace_id?: string;
};

export type ApiResult<T> = {
  data: T;
  traceId?: string;
};

export type ApiErrorPayload = {
  message: string;
  traceId?: string;
  status?: number;
};

export type HealthStatus = {
  status: string;
  env?: string;
  mock?: boolean;
  rag?: {
    embedder_provider?: string;
    vector_store_provider?: string;
  };
};
