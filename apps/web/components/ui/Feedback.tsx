import type { ReactNode } from 'react';

export type FeedbackProps = {
  tone?: 'status' | 'error';
  children: ReactNode;
};

export function Feedback({ tone = 'status', children }: FeedbackProps) {
  return (
    <p className={`vp-feedback vp-feedback--${tone}`} role={tone === 'error' ? 'alert' : 'status'}>
      {children}
    </p>
  );
}
