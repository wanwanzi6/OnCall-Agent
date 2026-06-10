import { Activity, Bot, Database, FileText, MessageSquare, Settings } from 'lucide-react';

export type PageKey = 'chat' | 'knowledge' | 'aiops' | 'reports' | 'settings';

const navItems: Array<{ key: PageKey; label: string; icon: typeof MessageSquare }> = [
  { key: 'chat', label: 'Chat', icon: MessageSquare },
  { key: 'knowledge', label: 'Knowledge', icon: Database },
  { key: 'aiops', label: 'AI Ops', icon: Activity },
  { key: 'reports', label: 'Reports', icon: FileText },
  { key: 'settings', label: 'Settings', icon: Settings },
];

type Props = {
  active: PageKey;
  onChange: (page: PageKey) => void;
};

export function Sidebar({ active, onChange }: Props) {
  return (
    <aside className="sidebar">
      <div className="brand">
        <Bot size={22} />
        <div>
          <strong>OnCall Agent</strong>
          <span>Ops Console</span>
        </div>
      </div>
      <nav>
        {navItems.map((item) => {
          const Icon = item.icon;
          return (
            <button
              key={item.key}
              className={active === item.key ? 'nav-item active' : 'nav-item'}
              type="button"
              onClick={() => onChange(item.key)}
            >
              <Icon size={18} />
              <span>{item.label}</span>
            </button>
          );
        })}
      </nav>
    </aside>
  );
}
