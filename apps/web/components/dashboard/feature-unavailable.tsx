import Link from 'next/link';
import type { DashboardDestination } from './navigation';

export function FeatureUnavailable({ destination }: { destination: DashboardDestination }) {
  return (
    <section className="space-y-6 rounded-2xl border border-slate-200 bg-white p-6 sm:p-10">
      <p className="text-sm font-semibold uppercase tracking-widest text-teal-800">{destination.group}</p>
      <h1 className="text-3xl font-semibold tracking-tight">{destination.label}</h1>
      <p className="max-w-xl leading-relaxed text-slate-600">{destination.description}</p>
      <div className="space-y-2 rounded-xl bg-slate-50 p-5">
        <h2 className="font-semibold">Not available yet</h2>
        <p className="max-w-xl text-sm leading-relaxed text-slate-600">This area has not been implemented. No {destination.group === 'Banking' ? 'banking' : 'receipt'} data is loaded or displayed here.</p>
      </div>
      <Link href="/dashboard" className="inline-flex min-h-12 items-center font-semibold text-teal-800 underline underline-offset-4 focus-visible:outline-2 focus-visible:outline-teal-700">Back to overview</Link>
    </section>
  );
}
