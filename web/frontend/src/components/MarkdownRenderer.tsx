import DOMPurify from 'dompurify';
import { marked } from 'marked';

type Props = {
  content?: string;
};

marked.setOptions({
  gfm: true,
  breaks: true,
});

export function MarkdownRenderer({ content = '' }: Props) {
  const html = DOMPurify.sanitize(marked.parse(content || '') as string, {
    USE_PROFILES: { html: true },
  });
  return <div className="markdown-body" dangerouslySetInnerHTML={{ __html: html }} />;
}
