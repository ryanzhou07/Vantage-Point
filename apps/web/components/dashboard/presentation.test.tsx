import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';
import { DashboardOverview } from './overview';
import { FeatureUnavailable } from './feature-unavailable';
import { DashboardNavigation } from './dashboard-navigation';
import { destinationFor, destinations } from './navigation';

vi.mock('next/navigation', () => ({ usePathname: () => '/dashboard/accounts' }));

it('covers only MVP destinations and rejects unknown sections', () => {
  expect(destinations.map((destination) => destination.slug)).toEqual(['institutions', 'accounts', 'transactions', 'recurring', 'receipts', 'receipt-history', 'receivables']);
  expect(destinationFor('forecast')).toBeUndefined();
});

it('exposes navigation links with the exact active page and keyboard focus styling', () => {
  const html = renderToStaticMarkup(<DashboardNavigation />);
  expect(html).toContain('aria-label="Dashboard navigation"');
  expect(html.match(/aria-current="page"/g)).toHaveLength(1);
  expect(html).toMatch(/<a[^>]*aria-current="page"[^>]*href="\/dashboard\/accounts"/);
  expect(html).toContain('focus-visible:');
});

it('overview links explain availability rather than implying functioning financial actions', () => {
  const html = renderToStaticMarkup(<DashboardOverview />);
  expect(html).toContain('Banking and Receipt tools are not available yet');
  for (const destination of destinations) expect(html).toContain(`href="/dashboard/${destination.slug}"`);
  expect(html).not.toMatch(/\$|€|£|Connect now|Upload receipt|Mark paid/);
});

describe('unimplemented destinations', () => {
  it.each(destinations)('$label honestly explains that no domain data is loaded', (destination) => {
    const html = renderToStaticMarkup(<FeatureUnavailable destination={destination} />);
    expect(html).toContain('This area has not been implemented');
    expect(html).toContain('data is loaded or displayed here');
    expect(html).toContain('href="/dashboard"');
    expect(html).not.toMatch(/<button|<input|\$|€|£/);
  });
});
