export function Brand({ compact = false }: { compact?: boolean }) {
  return (
    <div className="vp-brand">
      <span className="vp-brand-mark" aria-hidden="true">V</span>
      <span>Vantage-Point{!compact && <small>Personal finance, together.</small>}</span>
    </div>
  );
}
