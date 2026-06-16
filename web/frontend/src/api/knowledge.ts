import { apiRequest } from './client';
import type { DocumentItem, KnowledgeSearchResponse, UploadResult } from '../types/rag';

export const ALLOWED_UPLOAD_EXTS = ['.md', '.markdown', '.txt'];
export const DEFAULT_MAX_UPLOAD_BYTES = 10 * 1024 * 1024;

export function validateUploadFile(file: File, maxBytes = DEFAULT_MAX_UPLOAD_BYTES): string | undefined {
  const lower = file.name.toLowerCase();
  const allowed = ALLOWED_UPLOAD_EXTS.some((ext) => lower.endsWith(ext));
  if (!allowed) return '只允许上传 .md、.markdown、.txt 文件';
  if (file.size > maxBytes) return `文件不能超过 ${Math.round(maxBytes / 1024 / 1024)}MB`;
  return undefined;
}

export async function uploadKnowledgeFile(file: File) {
  const form = new FormData();
  form.append('file', file);
  return apiRequest<UploadResult>('/knowledge/upload', {
    method: 'POST',
    body: form,
  });
}

export async function listDocuments() {
  const result = await apiRequest<{ documents: DocumentItem[] }>('/knowledge/documents');
  return { ...result, data: result.data.documents ?? [] };
}

export async function deleteDocument(id: string) {
  return apiRequest<{ deleted: boolean }>(`/knowledge/documents/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  });
}

export async function searchKnowledge(query: string, topK: number) {
  const result = await apiRequest<KnowledgeSearchResponse>('/knowledge/search', {
    method: 'POST',
    body: JSON.stringify({ query, top_k: topK }),
  });
  return { ...result, data: { ...result.data, results: result.data.results ?? [] } };
}
