import type { ChatMessage } from '../types/rag';
import type { AIOpsAnalyzeResult, StoredReport } from '../types/aiops';

const CHAT_KEY = 'oncall-agent.chat.messages';
const REPORT_KEY = 'oncall-agent.aiops.reports';
const REPORT_LIMIT = 30;

export function loadChatMessages(): ChatMessage[] {
  return loadJson<ChatMessage[]>(CHAT_KEY, []);
}

export function saveChatMessages(messages: ChatMessage[]): void {
  localStorage.setItem(CHAT_KEY, JSON.stringify(messages.slice(-50)));
}

export function clearChatMessages(): void {
  localStorage.removeItem(CHAT_KEY);
}

export function loadReports(): StoredReport[] {
  return loadJson<StoredReport[]>(REPORT_KEY, []);
}

export function saveReports(reports: StoredReport[]): void {
  localStorage.setItem(REPORT_KEY, JSON.stringify(reports.slice(0, REPORT_LIMIT)));
}

export function addReport(result: AIOpsAnalyzeResult): StoredReport {
  const reports = loadReports();
  const firstAlert = result.alerts?.[0];
  const item: StoredReport = {
    id: `${Date.now()}-${result.trace_id ?? 'trace'}`,
    created_at: new Date().toISOString(),
    trace_id: result.trace_id,
    alert_summary: firstAlert ? `${firstAlert.name ?? '-'} / ${firstAlert.service ?? '-'}` : '无活跃告警',
    mode: result.mode,
    fallback_used: result.fallback_used,
    result,
  };
  saveReports([item, ...reports]);
  return item;
}

function loadJson<T>(key: string, fallback: T): T {
  try {
    const value = localStorage.getItem(key);
    return value ? (JSON.parse(value) as T) : fallback;
  } catch {
    return fallback;
  }
}
