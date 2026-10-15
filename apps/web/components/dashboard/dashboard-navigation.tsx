'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { destinations } from './navigation';

const linkStyle = 'flex min-h-12 items-center rounded-xl px-4 py-3 text-sm font-medium focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-teal-700';

export function DashboardNavigation() {
  const pathname = usePathname();

  function linkClass(href: string) {
    return `${linkStyle} ${pathname === href ? 'bg-slate-900 text-white' : 'text-slate-700 hover:bg-slate-100'}`;
  }

  return (
    <nav aria-label="Dashboard navigation" className="space-y-5">
      <Link href="/dashboard" aria-current={pathname === '/dashboard' ? 'page' : undefined} className={linkClass('/dashboard')}>
        Overview
      </Link>

      {(['Banking', 'Receipt splitting'] as const).map((group) => (
        <div key={group}>
          <p className="mb-2 px-4 text-xs font-semibold uppercase tracking-widest text-slate-500">{group}</p>
          <ul className="space-y-1">
            {destinations.filter((destination) => destination.group === group).map((destination) => {
              const href = `/dashboard/${destination.slug}`;

              return (
                <li key={destination.slug}>
                  <Link href={href} aria-current={pathname === href ? 'page' : undefined} className={linkClass(href)}>
                    {destination.label}
                  </Link>
                </li>
              );
            })}
          </ul>
        </div>
      ))}

      <Link href="/dashboard/settings" aria-current={pathname === '/dashboard/settings' ? 'page' : undefined} className={linkClass('/dashboard/settings')}>
        Settings
      </Link>
    </nav>
  );
}
