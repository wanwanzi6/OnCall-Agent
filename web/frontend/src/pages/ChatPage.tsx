import { Send, Trash2 } from 'lucide-react';
import { useEffect, useState } from 'react';
import { sendChat, streamChat } from '../api/chat';
import { ApiError } from '../api/client';
import { CitationList } from '../components/CitationList';
import { EmptyState } from '../components/EmptyState';
import { ErrorBanner } from '../components/ErrorBanner';
import { PageHeader } from '../components/PageHeader';
import { TraceId } from '../components/TraceId';
import type { ChatMessage } from '../types/rag';
import { clearChatMessages, loadChatMessages, saveChatMessages } from '../utils/storage';

export function ChatPage() {
  const [messages, setMessages] = useState<ChatMessage[]>(() => loadChatMessages());
  const [input, setInput] = useState('服务下线告警应该怎么处理？');
  const [loading, setLoading] = useState(false);
  const [streaming, setStreaming] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => saveChatMessages(messages), [messages]);

  async function submit() {
    const question = input.trim();
    if (!question || loading) return;
    setError(null);
    setLoading(true);
    const userMessage: ChatMessage = {
      id: crypto.randomUUID(),
      role: 'user',
      content: question,
      createdAt: new Date().toISOString(),
    };
    const assistantId = crypto.randomUUID();
    setMessages((prev) => [
      ...prev,
      userMessage,
      { id: assistantId, role: 'assistant', content: '', citations: [], createdAt: new Date().toISOString() },
    ]);
    setInput('');

    try {
      if (streaming) {
        let answer = '';
        const result = await streamChat(question, (chunk) => {
          if (chunk.delta) {
            answer += answer ? ` ${chunk.delta}` : chunk.delta;
            setMessages((prev) => prev.map((m) => (m.id === assistantId ? { ...m, content: answer } : m)));
          }
        });
        const full = await sendChat(question);
        setMessages((prev) =>
          prev.map((m) =>
            m.id === assistantId
              ? { ...m, content: full.data.answer || answer, citations: full.data.citations ?? [], traceId: full.traceId ?? result.traceId }
              : m,
          ),
        );
      } else {
        const result = await sendChat(question);
        setMessages((prev) =>
          prev.map((m) =>
            m.id === assistantId
              ? { ...m, content: result.data.answer, citations: result.data.citations ?? [], traceId: result.traceId }
              : m,
          ),
        );
      }
    } catch (err) {
      setError(err instanceof Error ? err : new Error('chat failed'));
      setMessages((prev) => prev.filter((m) => m.id !== assistantId));
    } finally {
      setLoading(false);
    }
  }

  function clear() {
    clearChatMessages();
    setMessages([]);
  }

  return (
    <div className="page">
      <PageHeader
        title="Chat"
        description="基于已索引 SOP 的 RAG 问答。"
        actions={
          <button className="button ghost" type="button" onClick={clear}>
            <Trash2 size={16} /> 清空
          </button>
        }
      />
      <ErrorBanner error={error} />
      <section className="panel chat-panel">
        {messages.length === 0 ? (
          <EmptyState title="暂无对话" description="先上传 SOP，再询问告警处理或排障步骤。" />
        ) : (
          <div className="message-list">
            {messages.map((message) => (
              <article className={`message ${message.role}`} key={message.id}>
                <div className="message-role">{message.role === 'user' ? 'You' : 'Assistant'}</div>
                <p>{message.content || (loading ? '分析中...' : '')}</p>
                {message.traceId ? <TraceId value={message.traceId} /> : null}
                {message.role === 'assistant' ? <CitationList citations={message.citations} /> : null}
              </article>
            ))}
          </div>
        )}
      </section>
      <section className="composer">
        <label className="toggle">
          <input checked={streaming} type="checkbox" onChange={(event) => setStreaming(event.target.checked)} />
          <span>Stream</span>
        </label>
        <textarea
          value={input}
          placeholder="输入排障问题..."
          onChange={(event) => setInput(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === 'Enter' && (event.ctrlKey || event.metaKey)) void submit();
          }}
        />
        <button className="button primary" disabled={loading || !input.trim()} type="button" onClick={() => void submit()}>
          <Send size={16} /> 发送
        </button>
      </section>
      {error instanceof ApiError ? null : null}
    </div>
  );
}
