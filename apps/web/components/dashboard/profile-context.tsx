'use client';

import { createContext, useContext } from 'react';
import type { Profile } from '../../lib/auth/flows';

export const DashboardProfileContext = createContext<Profile | null>(null);

export function useDashboardProfile() {
  return useContext(DashboardProfileContext);
}
