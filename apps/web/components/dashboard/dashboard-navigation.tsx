'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useState } from 'react';
import { NavigationIcon } from '../ui/NavigationIcon';
import { destinations } from './navigation';
import './navigation.css';

function NavigationGroup({ group, pathname }: { group: 'Banking' | 'Receipt splitting'; pathname: string }) {
  const [expanded, setExpanded] = useState(true);
  const id = group === 'Banking' ? 'banking-links' : 'receipt-links';

  return (
    <div className={`vp-nav-group ${expanded ? 'is-expanded' : ''}`}>
      <button className="vp-nav-row vp-nav-parent" type="button" aria-expanded={expanded} aria-controls={id} onClick={() => setExpanded(!expanded)}>
        <NavigationIcon name={group === 'Banking' ? 'banking' : 'receipts'} />
        <span>{group}</span>
        <svg className={`vp-nav-chevron ${expanded ? 'is-expanded' : ''}`} width="12" height="12" viewBox="0 0 12 12" aria-hidden="true"><path d="m4 2 4 4-4 4" fill="none" stroke="currentColor" strokeWidth="1.5" /></svg>
      </button>
      <ul id={id} className="vp-nav-children" hidden={!expanded}>
        {destinations.filter((destination) => destination.group === group).map((destination) => {
          const href = `/dashboard/${destination.slug}`;

          return (
            <li key={destination.slug}>
              <Link href={href} aria-current={pathname === href ? 'page' : undefined} className="vp-nav-row vp-nav-child">{destination.label}</Link>
            </li>
          );
        })}
      </ul>
    </div>
  );
}

export function DashboardNavigation() {
  const pathname = usePathname();

  return (
    <nav aria-label="Dashboard navigation" className="vp-dashboard-navigation">
      <Link href="/dashboard" aria-current={pathname === '/dashboard' ? 'page' : undefined} className="vp-nav-row">
        <NavigationIcon name="overview" /><span>Overview</span>
      </Link>
      <div className="vp-nav-section">
        <p className="vp-nav-caption">WORKSPACE</p>
        <NavigationGroup group="Banking" pathname={pathname} />
        <NavigationGroup group="Receipt splitting" pathname={pathname} />
      </div>
      <div className="vp-nav-footer">
        <Link href="/dashboard/settings" aria-current={pathname === '/dashboard/settings' ? 'page' : undefined} className="vp-nav-row">
          <NavigationIcon name="settings" /><span>Settings</span>
        </Link>
      </div>
    </nav>
  );
}
