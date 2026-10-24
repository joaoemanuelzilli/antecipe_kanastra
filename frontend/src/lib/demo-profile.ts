import { useMemo, useSyncExternalStore } from 'react';

export interface DemoProfile {
  name: string;
  email: string;
  company: string;
  role: string;
  confirmLogout: boolean;
  reducedMotion: boolean;
}
export const DEFAULT_DEMO_PROFILE: DemoProfile = { name: 'Marina Costa', email: 'marina@empresa.example', company: 'Empresa Exemplo', role: 'Financeiro', confirmLogout: true, reducedMotion: false };
const KEY = 'antecipe.demo.profile.v1';
const EVENT = 'antecipe:demo-profile';
function snapshot(): string | null { try { return sessionStorage.getItem(KEY); } catch { return null; } }
export function parseDemoProfile(raw: string | null): DemoProfile {
  try {
    const value: unknown = JSON.parse(raw ?? 'null');
    if (!value || typeof value !== 'object' || Array.isArray(value)) return DEFAULT_DEMO_PROFILE;
    const record = value as Record<string, unknown>;
    return { ...DEFAULT_DEMO_PROFILE, ...Object.fromEntries(['name', 'email', 'company', 'role'].filter(key => typeof record[key] === 'string' && (record[key] as string).length <= 120).map(key => [key, record[key]])), confirmLogout: typeof record.confirmLogout === 'boolean' ? record.confirmLogout : true, reducedMotion: record.reducedMotion === true };
  } catch { return DEFAULT_DEMO_PROFILE; }
}
export function saveDemoProfile(profile: DemoProfile): boolean {
  try {
    // Explicit allowlist: never persist passwords, login values or other form fields.
    const { name, email, company, role, confirmLogout, reducedMotion } = profile;
    sessionStorage.setItem(KEY, JSON.stringify({ name, email, company, role, confirmLogout, reducedMotion }));
    window.dispatchEvent(new Event(EVENT));
    return true;
  } catch { return false; }
}
function subscribe(listener: () => void): () => void { window.addEventListener(EVENT, listener); window.addEventListener('storage', listener); return () => { window.removeEventListener(EVENT, listener); window.removeEventListener('storage', listener); }; }
export function useDemoProfile(): DemoProfile {
  const raw = useSyncExternalStore(subscribe, snapshot, () => null);
  return useMemo(() => parseDemoProfile(raw), [raw]);
}
