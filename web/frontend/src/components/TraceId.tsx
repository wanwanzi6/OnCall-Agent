import { Copy } from 'lucide-react';
import { copyText } from '../utils/format';

type Props = {
  value?: string;
};

export function TraceId({ value }: Props) {
  if (!value) return <span className="muted">trace_id: -</span>;
  const short = value.length > 18 ? `${value.slice(0, 10)}...${value.slice(-6)}` : value;
  return (
    <button className="trace-id" type="button" title={value} onClick={() => void copyText(value)}>
      <span>trace_id: {short}</span>
      <Copy size={14} />
    </button>
  );
}
