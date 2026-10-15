import Link from 'next/link';
import { destinations } from './navigation';

export function DashboardOverview() {
  return (
    <div className="space-y-8">
      <div className="space-y-3">
        <p className="text-sm font-semibold uppercase tracking-widest text-teal-800">Overview</p>
        <h1 className="text-3xl font-semibold tracking-tight sm:text-4xl">Your financial workspace</h1>
        <p className="max-w-2xl leading-relaxed text-slate-600">One place for your accounts and shared expenses. Explore the workspace below; Banking and Receipt tools are not available yet.</p>
      </div>

      {(['Banking', 'Receipt splitting'] as const).map((group) => (
        <section key={group} aria-label={group} className="space-y-4">
          <h2 className="text-xl font-semibold">{group}</h2>
          <div className="grid gap-4 sm:grid-cols-2">
            {destinations.filter((destination) => destination.group === group).map((destination) => (
              <Link key={destination.slug} href={`/dashboard/${destination.slug}`} className="flex flex-col items-start gap-3 rounded-2xl border border-slate-200 bg-white p-6 hover:border-teal-700 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-teal-700">
                <span className="rounded-full bg-slate-100 px-3 py-1 text-xs font-medium text-slate-600">Not available yet</span>
                <h3 className="text-lg font-semibold">{destination.label}</h3>
                <p className="text-sm leading-relaxed text-slate-600">{destination.description}</p>
                <span className="mt-auto pt-2 text-sm font-semibold text-teal-800">View availability <span aria-hidden="true">→</span></span>
              </Link>
            ))}
          </div>
        </section>
      ))}
    </div>
  );
}
