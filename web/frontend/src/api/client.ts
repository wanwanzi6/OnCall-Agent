import type { ApiEnvelope, ApiErrorPayload, ApiResult } from '../types/api';

export const API_BASE_URL =
  (import.meta.env.VITE_API_BASE_URL as string | undefined)?.replace(/\/$/, '') ?? 'http://localhost:8080/api';

export class ApiError extends Error {
  traceId?: string;
  status?: number;

  constructor(payload: ApiErrorPayload) {
    super(payload.message);
    this.name = 'ApiError';
    this.traceId = payload.traceId;
    this.status = payload.status;
  }
}

export function newTraceId(prefix = 'web'): string {
  if (crypto.randomUUID) return `${prefix}-${crypto.randomUUID()}`;
  return `${prefix}-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

export async function apiRequest<T>(
  path: string,
  options: RequestInit = {},
  traceId = newTraceId(),
): Promise<ApiResult<T>> {
  const headers = new Headers(options.headers);
  headers.set('X-Trace-ID', traceId);
  if (!(options.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...options,
    headers,
  });
  const responseTraceId = response.headers.get('X-Trace-ID') ?? traceId;
  const envelope = (await readJson(response)) as ApiEnvelope<T> | undefined;
  const finalTraceId = envelope?.trace_id ?? responseTraceId;

  if (!response.ok || !envelope || envelope.code !== 0) {
    throw new ApiError({
      message: envelope?.message || response.statusText || 'request failed',
      traceId: finalTraceId,
      status: response.status,
    });
  }
  return { data: (envelope.data ?? {}) as T, traceId: finalTraceId };
}

async function readJson(response: Response): Promise<unknown> {
  const text = await response.text();
  if (!text) return undefined;
  try {
    return JSON.parse(text);
  } catch {
    throw new ApiError({
      message: 'response is not valid JSON',
      traceId: response.headers.get('X-Trace-ID') ?? undefined,
      status: response.status,
    });
  }
}
