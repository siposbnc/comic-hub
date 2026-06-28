import { invoke } from '@tauri-apps/api/core';
import type { ComicHubClient } from '@comichub/api-client';
import type { ReaderSettings } from './types.js';

/** Where per-book reader overrides are stored. `server` syncs across devices (connected
 *  mode only); `local` keeps them on this machine. */
export type SyncMode = 'local' | 'server';

/** App-level reader configuration (not per book). Persisted in localStorage. */
export interface ReaderConfig {
  /** Remember layout/fit/direction/background per book and restore on open. */
  rememberPerBook: boolean;
  syncMode: SyncMode;
}

const CONFIG_KEY = 'comichub.reader.config';
const DEFAULT_CONFIG: ReaderConfig = { rememberPerBook: true, syncMode: 'local' };

export function loadConfig(): ReaderConfig {
  try {
    const raw = localStorage.getItem(CONFIG_KEY);
    if (raw) return { ...DEFAULT_CONFIG, ...(JSON.parse(raw) as Partial<ReaderConfig>) };
  } catch {
    // ignore malformed/unavailable storage
  }
  return { ...DEFAULT_CONFIG };
}

export function saveConfig(config: ReaderConfig): void {
  try {
    localStorage.setItem(CONFIG_KEY, JSON.stringify(config));
  } catch {
    // ignore
  }
}

/** A per-book settings store. Implementations swallow errors (best-effort persistence). */
export interface PrefsBackend {
  load(bookId: string): Promise<Partial<ReaderSettings> | null>;
  save(bookId: string, settings: ReaderSettings): Promise<void>;
}

/** Local store backed by the Tauri core (reader_prefs.json). No-ops on the web. */
export const localPrefs: PrefsBackend = {
  async load(bookId) {
    try {
      const v = await invoke<Partial<ReaderSettings> | null>('local_restore_prefs', { bookId });
      return v ?? null;
    } catch {
      return null;
    }
  },
  async save(bookId, settings) {
    try {
      await invoke('local_save_prefs', { bookId, settings });
    } catch {
      // ignore (e.g. running on the web)
    }
  },
};

/** Server store backed by the catalog (per-user, per-book). Connected mode only. */
export function serverPrefs(client: ComicHubClient): PrefsBackend {
  return {
    async load(bookId) {
      try {
        const s = await client.getReaderPrefs(bookId);
        return s && Object.keys(s).length > 0 ? (s as Partial<ReaderSettings>) : null;
      } catch {
        return null;
      }
    },
    async save(bookId, settings) {
      try {
        await client.putReaderPrefs(bookId, settings as unknown as Record<string, unknown>);
      } catch {
        // ignore
      }
    },
  };
}
