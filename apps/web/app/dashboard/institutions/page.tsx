import { ConnectionsPanel } from '../../../features/banking/ConnectionsPanel';

export default function InstitutionsPage() {
  return (
    <section className="space-y-6 rounded-2xl border border-slate-200 bg-white p-6 sm:p-10">
      <p className="text-sm font-semibold uppercase tracking-widest text-teal-800">Banking</p>
      <h1 className="text-3xl font-semibold tracking-tight">Institutions</h1>
      <p className="max-w-xl leading-relaxed text-slate-600">Connect and manage your financial institutions.</p>
      <ConnectionsPanel />
    </section>
  );
}
