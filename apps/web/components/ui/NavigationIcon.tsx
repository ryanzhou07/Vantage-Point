export type NavigationIconName = 'overview' | 'banking' | 'receipts' | 'settings' | 'general' | 'appearance' | 'account' | 'connections';

const paths: Record<NavigationIconName, string> = {
  overview: 'M3 3h7v7H3z M14 3h7v7h-7z M3 14h7v7H3z M14 14h7v7h-7z',
  banking: 'M3 9h18 M4 9v11 M9 9v11 M15 9v11 M20 9v11 M2 21h20 M2 7l10-5 10 5z',
  receipts: 'M5 3h14v18l-3-2-4 2-4-2-3 2z M8 8h8 M8 12h8',
  settings: 'M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8 M10 2h4l1 3 3 1 3 3-1 3 1 3-3 3-3 1-1 3h-4l-1-3-3-1-3-3 1-3-1-3 3-3 3-1z',
  general: 'M4 6h16 M4 12h16 M4 18h16 M8 3v6 M16 9v6 M10 15v6',
  appearance: 'M12 3a9 9 0 1 0 9 9h-9z M12 3v9h9',
  account: 'M16 7a4 4 0 1 0-8 0 4 4 0 0 0 8 0 M4 21v-3a8 8 0 0 1 16 0v3',
  connections: 'M9 7l2-2a5 5 0 0 1 7 7l-2 2 M15 17l-2 2a5 5 0 0 1-7-7l2-2 M8 16l8-8',
};

export function NavigationIcon({ name }: { name: NavigationIconName }) {
  return <svg className="vp-nav-icon" width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"><path d={paths[name]} /></svg>;
}
