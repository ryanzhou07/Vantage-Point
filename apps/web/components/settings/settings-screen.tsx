'use client';

import { useState } from 'react';
import { NavigationIcon, type NavigationIconName } from '../ui/NavigationIcon';
import { useDashboardProfile } from '../dashboard/profile-context';
import { FutureAction, PreviewField, SettingsCard } from './settings-parts';
import { ConnectionsPanel } from '../../features/banking/ConnectionsPanel';
import './settings.css';

const sections = [
  { id: 'general', label: 'General', description: 'Your profile and everyday preferences.' },
  { id: 'appearance', label: 'Appearance', description: 'Make your workspace feel like yours.' },
  { id: 'account', label: 'Account', description: 'Your identity and sign-in security.' },
  { id: 'connections', label: 'Plugins / Connections', description: 'Bank connections and payment contact details.' },
] as const;

type Section = typeof sections[number]['id'];
type Appearance = 'Light' | 'Dark' | 'System';

function GeneralSettings() {
  const profile = useDashboardProfile();

  return (
    <>
      <SettingsCard title="Profile" description="The information people will see when you share a receipt.">
        <div className="settings-profile-summary">
          <span className="settings-avatar" aria-hidden="true">{profile?.display_name?.slice(0, 1).toUpperCase() || 'VP'}</span>
          <div className="min-w-0">
            <p className="font-semibold break-words">{profile?.display_name || 'Your profile'}</p>
            <p className="settings-description break-words">{profile?.email || 'No email available'}</p>
          </div>
          <span className="settings-tag">Read-only profile</span>
        </div>
        <div className="settings-fields">
          <PreviewField id="display-name" label="Display name preview" value={profile?.display_name} placeholder="Your preferred name" />
          <PreviewField id="contact-phone" label="Contact phone preview" placeholder="Add a contact number" type="tel" />
        </div>
        <FutureAction description="Profile editing will be available in a future update. Draft inputs are not saved.">Save profile</FutureAction>
      </SettingsCard>
      <SettingsCard title="Regional preferences" description="Language and time-zone controls will live here.">
        <div className="settings-fields">
          <label className="settings-field">Language<select disabled defaultValue=""><option value="">Not configured — coming later</option></select></label>
          <label className="settings-field">Time zone<select disabled defaultValue=""><option value="">Not configured — coming later</option></select></label>
        </div>
      </SettingsCard>
    </>
  );
}

function AppearanceSettings() {
  const [appearance, setAppearance] = useState<Appearance>('System');

  return (
    <SettingsCard title="Color theme" description="Explore the design options. This preview does not change your application theme.">
      <fieldset>
        <legend className="settings-field-label">Theme preview</legend>
        <div className="settings-theme-grid">
          {(['Light', 'Dark', 'System'] as const).map((theme) => (
            <label key={theme} className={`settings-theme-option ${appearance === theme ? 'is-selected' : ''}`}>
              <span className={`settings-theme-art settings-theme-art--${theme.toLowerCase()}`} aria-hidden="true"><span /><span /><span /></span>
              <span className="settings-theme-label"><span>{theme}</span><input type="radio" name="theme-preview" value={theme} checked={appearance === theme} onChange={() => setAppearance(theme)} /></span>
            </label>
          ))}
        </div>
      </fieldset>
      <p role="status" className="settings-description">{appearance} selected for this preview only. Nothing is saved.</p>
      <div className="settings-divider" />
      <label className="settings-field">Display density<select disabled defaultValue=""><option value="">Comfortable — preview only</option></select></label>
      <p className="settings-description">Room for readable text, clear controls and a little breathing space.</p>
    </SettingsCard>
  );
}

function AccountSettings() {
  const profile = useDashboardProfile();

  return (
    <>
      <SettingsCard title="Sign-in identity" description="Google handles your sign-in. Vantage-Point does not store your Google password.">
        <dl className="settings-detail-list">
          <div><dt>Sign-in method</dt><dd>Google</dd></div>
          <div><dt>Email</dt><dd>{profile?.email || 'No email available'}</dd></div>
        </dl>
        <FutureAction description="Email management is not available in this preview.">Manage email</FutureAction>
      </SettingsCard>
      <SettingsCard title="Account security" description="A dedicated place for account controls as the product grows.">
        <FutureAction description="Additional security controls are planned. Manage your Google security through your Google account for now.">Manage security</FutureAction>
        <div className="settings-divider" />
        <FutureAction description="Account deletion is not available in this skeleton. No account data can be removed here.">Delete account</FutureAction>
      </SettingsCard>
    </>
  );
}

function ConnectionsSettings() {
  return (
    <>
      <SettingsCard title="Banking connections" description="Connect and disconnect your financial institutions.">
        <ConnectionsPanel />
      </SettingsCard>
      <SettingsCard title="Payment contact details" description="A future place to save payment details for reimbursements. These are contact details, not live account integrations.">
        <div className="settings-fields">
          <PreviewField id="venmo-contact" label="Venmo username preview" placeholder="@your-username" />
          <PreviewField id="zelle-contact" label="Zelle email or phone preview" placeholder="Email address or phone number" />
        </div>
        <div className="settings-payment-actions">
          <FutureAction description="Saving payment details is not available yet.">Save payment details</FutureAction>
          <FutureAction description="There are no saved details to remove in this preview.">Remove payment details</FutureAction>
        </div>
      </SettingsCard>
      <SettingsCard title="Plugins" description="Additional integrations can be managed here in the future.">
        <p className="settings-description">No plugins are available or connected.</p>
      </SettingsCard>
    </>
  );
}

export function SettingsScreen() {
  const [section, setSection] = useState<Section>('general');
  const current = sections.find((item) => item.id === section)!;

  return (
    <div className="settings-screen">
      <header className="settings-header"><p className="settings-eyebrow">YOUR WORKSPACE</p><h1>Settings</h1><p>Manage your preferences, account and connections.</p></header>
      <p id="settings-preview-note" className="settings-preview-note"><strong>Design preview.</strong> Explore the layout and try draft fields. Changes stay on this page only and are not saved or sent.</p>
      <div className="settings-layout">
        <nav aria-label="Settings sections" className="settings-section-nav">
          {sections.map((item) => <button key={item.id} type="button" aria-current={section === item.id ? 'page' : undefined} aria-controls="settings-section" onClick={() => setSection(item.id)}><NavigationIcon name={item.id as NavigationIconName} /><span>{item.label}</span></button>)}
        </nav>
        <section id="settings-section" aria-labelledby="settings-section-title" className="settings-section" key={section}>
          <header className="settings-section-header"><h2 id="settings-section-title">{current.label}</h2><p>{current.description}</p></header>
          {section === 'general' && <GeneralSettings />}
          {section === 'appearance' && <AppearanceSettings />}
          {section === 'account' && <AccountSettings />}
          {section === 'connections' && <ConnectionsSettings />}
        </section>
      </div>
    </div>
  );
}
