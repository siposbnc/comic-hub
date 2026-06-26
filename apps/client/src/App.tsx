import { useEffect, useState } from 'react';
import { ComicHubClient, type ServerInfo, type ServerStats } from '@comichub/api-client';
import { isTauri, startEmbeddedServer } from './connection.js';

type Phase =
  | { kind: 'starting' }
  | { kind: 'ready'; info: ServerInfo; stats: ServerStats; port?: number }
  | { kind: 'error'; message: string };

/**
 * Phase 0 client: prove the embedded loop end-to-end. The Rust core spawns the
 * sidecar, we read its handshake, then hit /healthz, /server/info and /server/stats
 * and render the result. Real screens (Home, Library, …) land in Phase 1.
 */
export function App() {
  const [phase, setPhase] = useState<Phase>({ kind: 'starting' });

  useEffect(() => {
    let cancelled = false;

    (async () => {
      try {
        if (!isTauri()) {
          throw new Error(
            'Not running inside Tauri. Launch with `pnpm tauri dev` so the sidecar can start.',
          );
        }
        const connection = await startEmbeddedServer();
        const client = new ComicHubClient(connection);
        await client.health();
        const [info, stats] = await Promise.all([client.serverInfo(), client.serverStats()]);
        if (!cancelled) setPhase({ kind: 'ready', info, stats, port: connection.port });
      } catch (err) {
        if (!cancelled) {
          setPhase({ kind: 'error', message: err instanceof Error ? err.message : String(err) });
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <div className="app">
      <div className="panel">
        <h1 className="wordmark">ComicHub</h1>
        <p className="subtitle">Phase 0 — embedded server handshake</p>
        {phase.kind === 'starting' && (
          <p>
            <span className="dot dot--wait" />
            Starting the media server…
          </p>
        )}
        {phase.kind === 'ready' && (
          <dl className="kv">
            <dt>status</dt>
            <dd>
              <span className="dot dot--ok" />
              connected
            </dd>
            <dt>server</dt>
            <dd>
              {phase.info.name} {phase.info.version}
            </dd>
            <dt>mode</dt>
            <dd>{phase.info.mode}</dd>
            <dt>port</dt>
            <dd>{phase.port ?? '—'}</dd>
            <dt>libraries</dt>
            <dd>{phase.stats.libraries}</dd>
            <dt>series</dt>
            <dd>{phase.stats.series}</dd>
            <dt>books</dt>
            <dd>{phase.stats.books}</dd>
          </dl>
        )}
        {phase.kind === 'error' && (
          <p>
            <span className="dot dot--err" />
            Could not reach the server.
            <span className="error">{phase.message}</span>
          </p>
        )}
      </div>
    </div>
  );
}
