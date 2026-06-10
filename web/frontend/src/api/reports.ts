import { loadReports, saveReports } from '../utils/storage';
import type { StoredReport } from '../types/aiops';

export function listLocalReports(): StoredReport[] {
  return loadReports();
}

export function deleteLocalReport(id: string): StoredReport[] {
  const next = loadReports().filter((report) => report.id !== id);
  saveReports(next);
  return next;
}
