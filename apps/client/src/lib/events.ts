import { useEffect } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import type { Connection, PresenceEntry } from '@comichub/api-client';
import { useConnection } from './client.js';
import { qk } from './queries.js';
import { useUiStore, type TrackedJob } from '../store/ui.js';

type WsFrame = {
  type: string;
  topic?: string;
  data?: unknown;
};

interface JobData {
  id: string;
  type: string;
  state: TrackedJob['state'];
  progress: number;
  total: number;
  done: number;
  error?: string;
}

const TOPICS = ['jobs', 'progress', 'library', 'presence'];
const RECONNECT_MIN = 1_000;
const RECONNECT_MAX = 15_000;

function wsUrl(connection: Connection): string {
  const base = connection.baseUrl.replace(/^http/, 'ws');
  const token = connection.token ? `?token=${encodeURIComponent(connection.token)}` : '';
  return `${base}/api/v1/ws${token}`;
}

/**
 * Opens the multiplexed WS once and keeps the UI live: job progress drives the top-bar
 * indicator, and catalog/progress events invalidate the matching React Query caches.
 * Reconnects with exponential backoff and re-subscribes (docs/03-api.md §10).
 */
export function useServerEvents(): void {
  const connection = useConnection();
  const qc = useQueryClient();

  useEffect(() => {
    let socket: WebSocket | null = null;
    let closed = false;
    let backoff = RECONNECT_MIN;
    let retryTimer: ReturnType<typeof setTimeout> | undefined;
    let hadConnection = false;

    const { upsertJob, clearFinishedJobs, addToast } = useUiStore.getState();

    const onJob = (job: JobData) => {
      upsertJob(job);
      if (job.state === 'done') {
        qc.invalidateQueries({ queryKey: qk.libraries });
        qc.invalidateQueries({ queryKey: ['series'] });
        qc.invalidateQueries({ queryKey: ['seriesDetail'] });
        qc.invalidateQueries({ queryKey: ['discover'] });
        // Drop the finished job from the indicator shortly after, once its 100% shows.
        setTimeout(() => clearFinishedJobs(), 1_500);
      } else if (job.state === 'failed') {
        addToast({
          tone: 'danger',
          title: 'Scan failed',
          message: job.error || 'The library scan did not finish.',
        });
        setTimeout(() => clearFinishedJobs(), 4_000);
      } else if (job.state === 'canceled') {
        setTimeout(() => clearFinishedJobs(), 500);
      }
    };

    const onProgress = () => {
      qc.invalidateQueries({ queryKey: qk.continueReading });
      qc.invalidateQueries({ queryKey: ['discover'] });
      qc.invalidateQueries({ queryKey: ['book'] });
      qc.invalidateQueries({ queryKey: ['seriesDetail'] });
      // Reading lists / collections / smart lists / the tracker surface per-issue read state
      // too, so a progress write elsewhere (e.g. the standalone reader leaving an issue
      // half-read) must refresh them as well.
      qc.invalidateQueries({ queryKey: ['readingList'] });
      qc.invalidateQueries({ queryKey: ['collection'] });
      qc.invalidateQueries({ queryKey: ['smartList'] });
      qc.invalidateQueries({ queryKey: qk.tracker });
    };

    const onLibrary = () => {
      qc.invalidateQueries({ queryKey: ['series'] });
      qc.invalidateQueries({ queryKey: ['seriesDetail'] });
      qc.invalidateQueries({ queryKey: ['discover'] });
      qc.invalidateQueries({ queryKey: qk.libraries });
    };

    // Presence ticks arrive per page turn, so patch the cache in place (no refetch):
    // the Now Reading strip's progress bars creep live.
    const onPresence = (frame: WsFrame) => {
      if (frame.type === 'presence.updated' && frame.data) {
        const e = frame.data as PresenceEntry;
        qc.setQueryData<PresenceEntry[]>(qk.presence, (cur) => [
          e,
          ...(cur ?? []).filter((x) => x.userId !== e.userId),
        ]);
      } else if (frame.type === 'presence.cleared' && frame.data) {
        const { userId } = frame.data as { userId: string };
        qc.setQueryData<PresenceEntry[]>(qk.presence, (cur) =>
          (cur ?? []).filter((x) => x.userId !== userId),
        );
      }
    };

    const connect = () => {
      if (closed) return;
      try {
        socket = new WebSocket(wsUrl(connection));
      } catch {
        scheduleRetry();
        return;
      }

      socket.onopen = () => {
        backoff = RECONNECT_MIN;
        socket?.send(JSON.stringify({ type: 'subscribe', topics: TOPICS }));
        // Presence changed while we were disconnected — re-seed from the snapshot.
        qc.invalidateQueries({ queryKey: qk.presence });
        // Events published while the socket was down (sleep/wake, server restart) are
        // gone for good, so a *re*-connect resyncs every push-refreshed cache.
        if (hadConnection) {
          onProgress();
          onLibrary();
        }
        hadConnection = true;
      };

      socket.onmessage = (ev) => {
        let frame: WsFrame;
        try {
          frame = JSON.parse(ev.data as string) as WsFrame;
        } catch {
          return;
        }
        if (frame.topic === 'jobs' && frame.data) onJob(frame.data as JobData);
        else if (frame.topic === 'progress') onProgress();
        else if (frame.topic === 'library') onLibrary();
        else if (frame.topic === 'presence') onPresence(frame);
      };

      socket.onclose = () => {
        if (!closed) scheduleRetry();
      };
      socket.onerror = () => {
        socket?.close();
      };
    };

    const scheduleRetry = () => {
      if (closed) return;
      retryTimer = setTimeout(connect, backoff);
      backoff = Math.min(backoff * 2, RECONNECT_MAX);
    };

    connect();

    return () => {
      closed = true;
      if (retryTimer) clearTimeout(retryTimer);
      if (socket) {
        socket.onclose = null;
        socket.close();
      }
    };
  }, [connection, qc]);
}
