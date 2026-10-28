import type { ReactNode } from 'react';
import { Surface } from '../ui/Surface';
import { Button } from '../ui/Button';

export function SettingsCard({ title, description, children }: {
  title: string;
  description: string;
  children: ReactNode;
}) {
  return (
    <Surface className="settings-card">
      <h3>{title}</h3>
      <p className="settings-description">{description}</p>
      <div className="settings-card-body">{children}</div>
    </Surface>
  );
}

export function PreviewField({ id, label, value = '', placeholder, type = 'text' }: {
  id: string;
  label: string;
  value?: string;
  placeholder?: string;
  type?: 'text' | 'tel';
}) {
  return (
    <label className="settings-field" htmlFor={id}>
      <span>{label}</span>
      <input id={id} type={type} defaultValue={value} placeholder={placeholder} maxLength={160} autoComplete="off" aria-describedby="settings-preview-note" />
    </label>
  );
}

export function FutureAction({ children, description }: { children: ReactNode; description: string }) {
  return (
    <div className="settings-future-action">
      <Button variant="secondary" disabled>{children}</Button>
      <p className="settings-description">{description}</p>
    </div>
  );
}
