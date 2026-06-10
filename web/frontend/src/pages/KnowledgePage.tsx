import { RefreshCw, Search, Trash2, Upload } from 'lucide-react';
import { useEffect, useState } from 'react';
import { deleteDocument, listDocuments, searchKnowledge, uploadKnowledgeFile, validateUploadFile } from '../api/knowledge';
import { EmptyState } from '../components/EmptyState';
import { ErrorBanner } from '../components/ErrorBanner';
import { PageHeader } from '../components/PageHeader';
import { TraceId } from '../components/TraceId';
import type { DocumentItem, SearchResult } from '../types/rag';
import { compactText, formatDate } from '../utils/format';

export function KnowledgePage() {
  const [docs, setDocs] = useState<DocumentItem[]>([]);
  const [results, setResults] = useState<SearchResult[]>([]);
  const [query, setQuery] = useState('服务下线 panic restart_count');
  const [topK, setTopK] = useState(3);
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [traceId, setTraceId] = useState<string | undefined>();
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    void refresh();
  }, []);

  async function refresh() {
    setError(null);
    setLoading(true);
    try {
      const response = await listDocuments();
      setDocs(response.data);
      setTraceId(response.traceId);
    } catch (err) {
      setError(err instanceof Error ? err : new Error('load documents failed'));
    } finally {
      setLoading(false);
    }
  }

  async function upload(file?: File) {
    if (!file) return;
    const validation = validateUploadFile(file);
    if (validation) {
      setError(new Error(validation));
      return;
    }
    setError(null);
    setUploading(true);
    try {
      const response = await uploadKnowledgeFile(file);
      setTraceId(response.traceId);
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err : new Error('upload failed'));
    } finally {
      setUploading(false);
    }
  }

  async function remove(id: string) {
    setError(null);
    try {
      const response = await deleteDocument(id);
      setTraceId(response.traceId);
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err : new Error('delete failed'));
    }
  }

  async function search() {
    if (!query.trim()) return;
    setError(null);
    setLoading(true);
    try {
      const response = await searchKnowledge(query, topK);
      setResults(response.data);
      setTraceId(response.traceId);
    } catch (err) {
      setError(err instanceof Error ? err : new Error('search failed'));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="page">
      <PageHeader
        title="Knowledge"
        description="上传 SOP 文档、查看索引状态并手动检索。"
        actions={
          <>
            <TraceId value={traceId} />
            <button className="button ghost" disabled={loading} type="button" onClick={() => void refresh()}>
              <RefreshCw size={16} /> 刷新
            </button>
          </>
        }
      />
      <ErrorBanner error={error} />
      <section className="grid two">
        <div className="panel">
          <div className="section-title">
            <h2>上传 SOP</h2>
          </div>
          <label className="upload-box">
            <Upload size={22} />
            <span>{uploading ? '上传并索引中...' : '选择 .md / .markdown / .txt 文件'}</span>
            <input
              accept=".md,.markdown,.txt"
              disabled={uploading}
              type="file"
              onChange={(event) => void upload(event.target.files?.[0])}
            />
          </label>
        </div>
        <div className="panel">
          <div className="section-title">
            <h2>手动搜索</h2>
          </div>
          <div className="form-row">
            <input value={query} onChange={(event) => setQuery(event.target.value)} />
            <input min={1} max={10} type="number" value={topK} onChange={(event) => setTopK(Number(event.target.value))} />
            <button className="button primary" disabled={loading || !query.trim()} type="button" onClick={() => void search()}>
              <Search size={16} /> 搜索
            </button>
          </div>
        </div>
      </section>

      <section className="panel">
        <div className="section-title">
          <h2>文档索引</h2>
          <span className="muted">{docs.length} documents</span>
        </div>
        {docs.length === 0 ? (
          <EmptyState title="暂无文档" description="上传 SOP 后会在这里看到 document_id 和索引状态。" />
        ) : (
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>name</th>
                  <th>document_id</th>
                  <th>chunk_count</th>
                  <th>created_at</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {docs.map((doc) => (
                  <tr key={doc.id}>
                    <td>{doc.name}</td>
                    <td><code>{doc.id}</code></td>
                    <td>{doc.chunk_count ?? '-'}</td>
                    <td>{formatDate(doc.created_at)}</td>
                    <td>
                      <button className="icon-button" type="button" title="删除" onClick={() => void remove(doc.id)}>
                        <Trash2 size={16} />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <section className="panel">
        <div className="section-title">
          <h2>搜索结果</h2>
          <span className="muted">{results.length} chunks</span>
        </div>
        {results.length === 0 ? (
          <EmptyState title="暂无结果" description="输入问题或关键字检索已上传 SOP。" />
        ) : (
          <div className="result-list">
            {results.map((item) => (
              <article className="result-item" key={item.chunk.id}>
                <div className="row-between">
                  <strong>{item.source || item.chunk.metadata?.source_file || 'unknown source'}</strong>
                  <span>score {item.score.toFixed(3)}</span>
                </div>
                <div className="meta-line">{item.title_path || item.chunk.metadata?.title_path || '-'}</div>
                <p>{compactText(item.chunk.content, 260)}</p>
              </article>
            ))}
          </div>
        )}
      </section>
    </div>
  );
}
