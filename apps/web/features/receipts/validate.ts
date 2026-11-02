import { money, type Draft } from './types';

const whole = (n: number) => Number.isInteger(n) && n >= 0;

// Plain-language list of what still blocks "confirm". The backend re-validates; this only guides the owner.
export function missingFields(d: Draft): string[] {
  const out: string[] = [];
  if (!d.merchant.trim()) out.push('Merchant name is empty.');
  if (!/^\d{4}-\d{2}-\d{2}$/.test(d.date)) out.push('Receipt date is missing or not in YYYY-MM-DD format.');
  if (d.currency !== 'USD') out.push('Currency must be USD.');
  if (d.items.length === 0) out.push('Add at least one item.');
  d.items.forEach((line, i) => {
    if (!line.name.trim()) out.push(`Item ${i + 1} needs a name.`);
    if (!Number.isInteger(line.quantity) || line.quantity < 1) out.push(`Item ${i + 1} needs a quantity of 1 or more.`);
    if (!whole(line.amount_minor)) out.push(`Item ${i + 1} needs a line amount in cents.`);
  });
  d.participants.forEach((p, i) => { if (!p.name.trim()) out.push(`Participant ${i + 1} needs a name (or remove them).`); });
  const amounts: [string, number][] = [['Subtotal', d.subtotal_minor], ['Tax', d.tax_minor], ['Tip', d.tip_minor], ['Receipt total', d.total_minor]];
  for (const [label, value] of amounts) if (!whole(value)) out.push(`${label} needs an amount in cents (0 or more).`);
  if (d.discount_minor !== 0) out.push('Receipt-level discount must be 0. Fold it into the item amounts first.');
  if (!d.tax_method) out.push('Choose how tax is split.');
  if (!d.tip_method) out.push('Choose how tip is split.');
  if (out.length) return out;
  const items = d.items.reduce((n, l) => n + l.amount_minor, 0);
  if (items !== d.subtotal_minor) out.push(`Items add up to ${money(items)} but the subtotal is ${money(d.subtotal_minor)}.`);
  if (d.subtotal_minor + d.tax_minor + d.tip_minor !== d.total_minor) out.push(`Subtotal + tax + tip is ${money(d.subtotal_minor + d.tax_minor + d.tip_minor)} but the receipt total is ${money(d.total_minor)}.`);
  return out;
}
