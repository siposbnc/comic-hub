import { create } from 'zustand';
import type { ComicHubClient, User } from '@comichub/api-client';

// Persisted auth state. NOTE: tokens live in localStorage for now; on desktop they should
// move to the OS keychain (a Tauri keychain plugin) — tracked as a follow-up. The chosen
// remote server URL is remembered so the app reconnects to it on next launch.
const SERVER_KEY = 'comichub.client.server';
const ACCESS_KEY = 'comichub.client.access';
const REFRESH_KEY = 'comichub.client.refresh';

export const tokenStore = {
  serverUrl: (): string | null => localStorage.getItem(SERVER_KEY),
  setServerUrl: (u: string) => localStorage.setItem(SERVER_KEY, u),
  clearServerUrl: () => localStorage.removeItem(SERVER_KEY),
  access: (): string => localStorage.getItem(ACCESS_KEY) ?? '',
  refresh: (): string => localStorage.getItem(REFRESH_KEY) ?? '',
  setTokens: (access: string, refresh: string) => {
    localStorage.setItem(ACCESS_KEY, access);
    localStorage.setItem(REFRESH_KEY, refresh);
  },
  clearTokens: () => {
    localStorage.removeItem(ACCESS_KEY);
    localStorage.removeItem(REFRESH_KEY);
  },
};

/** App-wide auth state: the signed-in user and whether the session dropped (refresh failed).
 *  In embedded / auth-disabled mode `user` is the implicit owner and `disconnected` stays
 *  false. */
interface AuthState {
  user: User | null;
  disconnected: boolean;
  setUser: (u: User | null) => void;
  setDisconnected: (d: boolean) => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  disconnected: false,
  setUser: (user) => set({ user, disconnected: false }),
  setDisconnected: (disconnected) => set({ disconnected }),
}));

/** True when the acting user can manage other accounts (owner/admin). */
export function isAdmin(user: User | null): boolean {
  return user?.role === 'owner' || user?.role === 'admin';
}

/** Wire a client's refresh-on-401 hook to the token store + auth state: refresh the access
 *  token from the stored refresh token, persist the rotated pair, and mark the session
 *  disconnected if refresh fails. */
export function wireRefresh(client: ComicHubClient): void {
  client.setUnauthorizedHandler(async () => {
    const refresh = tokenStore.refresh();
    if (!refresh) return null;
    try {
      const t = await client.refreshTokens(refresh);
      tokenStore.setTokens(t.access, t.refresh);
      useAuthStore.getState().setUser(t.user);
      return t.access;
    } catch {
      tokenStore.clearTokens();
      useAuthStore.getState().setDisconnected(true);
      return null;
    }
  });
}
