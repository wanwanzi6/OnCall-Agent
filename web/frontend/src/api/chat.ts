import { apiRequest, API_BASE_URL, ApiError, newTraceId } from './client';
import type { ChatAnswer, StreamChunk } from '../types/rag';

export async function sendChat(message: string) {
  return apiRequest<ChatAnswer>('/chat', {
    method: 'POST',
    body: JSON.stringify({ message }),
  });
}

export async function streamChat(
  message: string,
  onChunk: (chunk: StreamChunk) => void,
): Promise<{ traceId?: string }> {
  const traceId = newTraceId('web-chat-stream');
  const response = await fetch(`${API_BASE_URL}/chat/stream`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Trace-ID': traceId,
    },
    body: JSON.stringify({ message }),
  });
  const responseTraceId = response.headers.get('X-Trace-ID') ?? traceId;
  if (!response.ok || !response.body) {
    let detail = response.statusText;
    try {
      const payload = await response.json();
      detail = payload.message ?? detail;
      throw new ApiError({ message: detail, traceId: payload.trace_id ?? responseTraceId, status: response.status });
    } catch (error) {
      if (error instanceof ApiError) throw error;
      throw new ApiError({ message: detail, traceId: responseTraceId, status: response.status });
    }
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    const events = buffer.split('\n\n');
    buffer = events.pop() ?? '';
    for (const event of events) {
      const dataLine = event
        .split('\n')
        .find((line) => line.startsWith('data:'));
      if (!dataLine) continue;
      const raw = dataLine.slice(5).trim();
      if (!raw) continue;
      try {
        onChunk(JSON.parse(raw) as StreamChunk);
      } catch {
        onChunk({ delta: raw, done: false });
      }
    }
  }
  return { traceId: responseTraceId };
}
