import { apiRequest } from './client';
import type { HealthStatus } from '../types/api';

export function getHealth() {
  return apiRequest<HealthStatus>('/health');
}
