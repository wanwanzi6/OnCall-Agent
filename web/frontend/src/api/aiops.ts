import { apiRequest } from './client';
import type { AIOpsAnalyzeRequest, AIOpsAnalyzeResult } from '../types/aiops';

export async function analyzeAIOps(payload: AIOpsAnalyzeRequest) {
  return apiRequest<AIOpsAnalyzeResult>('/aiops/analyze', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}
