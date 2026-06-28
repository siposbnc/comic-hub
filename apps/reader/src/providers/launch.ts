import { invoke } from '@tauri-apps/api/core';
import { ComicHubClient } from '@comichub/api-client';
import type { Manifest, PageProvider } from '@comichub/reader-core';
import { ServerPageProvider } from '@comichub/reader-core';
import { LocalPageProvider } from './LocalPageProvider.js';
import { serverPrefs, type PrefsBackend } from '../reader/prefs.js';

const DEFAULT_SERVER_URL = 'http://127.0.0.1:8099';

export type LaunchResult =
  | {
      kind: 'connected';
      provider: PageProvider;
      startPage?: number;
      title?: string;
      /** Server-backed per-book settings store (only in connected mode). */
      prefsServer?: PrefsBackend;
    }
  | { kind: 'standalone'; provider: PageProvider; manifest: Manifest; title?: string }
  | { kind: 'empty' }
  | { kind: 'error'; message: string };

function isTauri(): boolean {
  return typeof window !== 'undefined' && '__TAURI_INTERNALS__' in window;
}

/**
 * Extracts a human message from a thrown value. Tauri's `invoke` rejects with the plain
 * string a command returns (not an Error), so the helpful Rust message — e.g. "Standalone
 * reading of .cbr files isn't supported yet" — is a string we must surface, not drop.
 */
function errMessage(err: unknown, fallback: string): string {
  if (typeof err === 'string' && err.trim()) return err;
  if (err instanceof Error && err.message) return err.message;
  return fallback;
}

/** Best-effort, stable-ish device label for progress reconciliation. */
function deviceLabel(): string {
  if (typeof navigator !== 'undefined' && navigator.platform) {
    return `reader-${navigator.platform}`;
  }
  return 'reader';
}

/**
 * Reads launch params from the URL. Supports both the dev harness query string and a
 * `comichub-reader://open?...` deep link (its search part is parsed the same way).
 */
function readParams(): URLSearchParams {
  if (typeof window === 'undefined') return new URLSearchParams();
  const { search, hash } = window.location;
  if (search && search.length > 1) return new URLSearchParams(search);
  // Deep links may surface their query in the hash depending on the host.
  const q = hash.indexOf('?');
  if (q >= 0) return new URLSearchParams(hash.slice(q));
  return new URLSearchParams();
}

/**
 * Reads launch params delivered via a `comichub-reader://open?...` deep link (the
 * client's one-click Read). Tauri-only; the plugin returns the URL(s) the app was
 * launched with. Returns null on web or when there's no deep link.
 */
async function readDeepLinkParams(): Promise<URLSearchParams | null> {
  try {
    const { getCurrent } = await import('@tauri-apps/plugin-deep-link');
    const urls = await getCurrent();
    if (!urls) return null;
    for (const raw of urls) {
      try {
        const u = new URL(raw);
        if (u.searchParams.get('bookId')) return u.searchParams;
      } catch {
        // not a parseable URL; skip
      }
    }
  } catch {
    // plugin unavailable (web) or no launch URLs
  }
  return null;
}

/** Parses launch params out of an explicit `comichub-reader://open?...` URL. */
function paramsFromUrl(raw: string): URLSearchParams {
  try {
    return new URL(raw).searchParams;
  } catch {
    return new URLSearchParams();
  }
}

/**
 * Resolves how the reader was launched and builds the matching PageProvider. Pass an
 * explicit deep-link URL to re-open a book while the reader is already running (the
 * single-instance forward path); otherwise params come from the window/deep link.
 */
export async function resolveLaunch(explicitUrl?: string): Promise<LaunchResult> {
  let params = explicitUrl ? paramsFromUrl(explicitUrl) : readParams();
  // A fresh launch from the client arrives as a deep link, not in window.location.
  if (!params.get('bookId') && !explicitUrl && isTauri()) {
    const deepLink = await readDeepLinkParams();
    if (deepLink) params = deepLink;
  }
  const bookId = params.get('bookId');
  const envServer = (import.meta.env.VITE_SERVER_URL as string | undefined) || undefined;

  // Connected mode: an explicit bookId (deep link or dev harness).
  if (bookId) {
    const baseUrl = params.get('server') || envServer || DEFAULT_SERVER_URL;
    const token = params.get('token') || '';
    const pageParam = params.get('page');
    const startPage = pageParam !== null ? Number.parseInt(pageParam, 10) : undefined;
    const client = new ComicHubClient({ baseUrl, token });
    const provider = new ServerPageProvider(client, bookId, deviceLabel());
    return {
      kind: 'connected',
      provider,
      startPage: Number.isFinite(startPage as number) ? startPage : undefined,
      prefsServer: serverPrefs(client),
    };
  }

  // Standalone mode: a file path from the OS (file association double-click).
  if (isTauri()) {
    try {
      const path = await invoke<string | null>('get_open_path');
      if (path) {
        const { provider, manifest } = await LocalPageProvider.open(path);
        const title = path.split(/[\\/]/).pop();
        return { kind: 'standalone', provider, manifest, title };
      }
    } catch (err) {
      return { kind: 'error', message: errMessage(err, 'Could not open the comic file.') };
    }
  }

  return { kind: 'empty' };
}

/**
 * Resolves a launch for an explicit on-disk comic path — the "Open file…" action inside the
 * reader. Always standalone mode (no server); surfaces the Rust opener's message on failure.
 */
export async function resolveLaunchFromPath(path: string): Promise<LaunchResult> {
  try {
    const { provider, manifest } = await LocalPageProvider.open(path);
    const title = path.split(/[\\/]/).pop();
    return { kind: 'standalone', provider, manifest, title };
  } catch (err) {
    return { kind: 'error', message: errMessage(err, 'Could not open the comic file.') };
  }
}
