type Props = {
  value?: string | boolean;
  kind?: 'status' | 'severity' | 'mode';
};

export function StatusBadge({ value, kind = 'status' }: Props) {
  const label = typeof value === 'boolean' ? (value ? 'yes' : 'no') : value || '-';
  const normalized = String(label).toLowerCase();
  const className = ['badge', `badge-${kind}`, `badge-${normalized}`].join(' ');
  return <span className={className}>{label}</span>;
}
