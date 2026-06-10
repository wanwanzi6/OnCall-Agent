import type { ReactNode } from 'react';
import { Sidebar, type PageKey } from './Sidebar';

type Props = {
  active: PageKey;
  onChange: (page: PageKey) => void;
  children: ReactNode;
};

export function Layout({ active, onChange, children }: Props) {
  return (
    <div className="app-shell">
      <Sidebar active={active} onChange={onChange} />
      <main className="main-panel">{children}</main>
    </div>
  );
}
