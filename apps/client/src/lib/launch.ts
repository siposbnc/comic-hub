import { useCallback } from 'react';
import { invoke } from '@tauri-apps/api/core';
import { useConnection } from './client.js';
import { useUiStore } from '../store/ui.js';
import { isTauri } from '../connection.js';

interface ReaderLaunch {
  baseUrl: string;
  token: string;
  bookId: string;
  page?: number;
}

/** Builds the reader deep link the desktop reader understands (mirrors reader launch.ts). */
export function readerDeepLink({ baseUrl, token, bookId, page }: ReaderLaunch): string {
  const p = new URLSearchParams({ server: baseUrl, bookId });
  if (token) p.set('token', token);
  if (page != null && page > 0) p.set('page', String(page));
  return `comichub-reader://open?${p.toString()}`;
}

/**
 * Launches the desktop reader for a book. In Tauri this hands the deep link to the OS
 * (the reader registers the `comichub-reader://` scheme). On the web there is no local
 * reader to spawn, so this resolves false and the caller shows a graceful fallback.
 */
async function launchReader(launch: ReaderLaunch): Promise<boolean> {
  if (!isTauri()) return false;
  await invoke('launch_reader', {
    url: readerDeepLink(launch),
    server: launch.baseUrl,
    token: launch.token,
    bookId: launch.bookId,
    page: launch.page ?? 0,
  });
  return true;
}

/**
 * Returns a one-click "open in reader" action. In the desktop app it hands a deep link
 * to the bundled reader; on the web there is no local reader, so it copies the deep
 * link and explains how to open it — never a dead end.
 */
export function useReadLaunch() {
  const connection = useConnection();
  const addToast = useUiStore((s) => s.addToast);

  return useCallback(
    async (bookId: string, page = 0) => {
      const launch = {
        baseUrl: connection.baseUrl,
        token: connection.token,
        bookId,
        page,
      };
      try {
        const launched = await launchReader(launch);
        if (launched) return;
      } catch (err) {
        addToast({
          tone: 'danger',
          title: 'Could not open the reader',
          message: err instanceof Error ? err.message : 'The reader did not start.',
        });
        return;
      }

      // Web fallback: no local reader process. Offer the deep link.
      const link = readerDeepLink(launch);
      let copied = false;
      try {
        if (navigator.clipboard) {
          await navigator.clipboard.writeText(link);
          copied = true;
        }
      } catch {
        copied = false;
      }
      addToast({
        tone: 'info',
        title: 'Open in the desktop reader',
        message: copied
          ? 'Reader link copied. Paste it into the ComicHub reader to open this issue.'
          : `Open this link in the ComicHub reader: ${link}`,
      });
    },
    [connection, addToast],
  );
}
