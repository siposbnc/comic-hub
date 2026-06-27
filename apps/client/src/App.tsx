import { useEffect, useMemo, useState } from 'react';
import { QueryClientProvider } from '@tanstack/react-query';
import { RouterProvider } from '@tanstack/react-router';
import { ComicHubClient, type Connection } from '@comichub/api-client';
import { resolveConnection, isTauri } from './connection.js';
import { ClientProvider } from './lib/client.js';
import { createQueryClient } from './lib/queries.js';
import { router } from './router.js';
import { useUiStore, applyTheme, applyAccent } from './store/ui.js';

type Boot =
  | { kind: 'starting' }
  | { kind: 'ready'; client: ComicHubClient; connection: Connection }
  | { kind: 'error'; message: string };

/**
 * App root: resolve a server connection (embedded sidecar in Tauri, dev server on the
 * web), then mount the query + client providers around the router. Theme/accent are
 * applied to the document before the first paint of real screens.
 */
export function App() {
  const [boot, setBoot] = useState<Boot>({ kind: 'starting' });
  const queryClient = useMemo(() => createQueryClient(), []);
  const theme = useUiStore((s) => s.theme);
  const accent = useUiStore((s) => s.accent);

  useEffect(() => {
    applyTheme(theme);
    applyAccent(accent);
  }, [theme, accent]);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const connection = await resolveConnection();
        const client = new ComicHubClient(connection);
        await client.health();
        if (!cancelled) setBoot({ kind: 'ready', client, connection });
      } catch (err) {
        if (!cancelled) {
          setBoot({ kind: 'error', message: err instanceof Error ? err.message : String(err) });
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  if (boot.kind === 'starting') return <BootScreen state="starting" />;
  if (boot.kind === 'error') return <BootScreen state="error" message={boot.message} />;

  return (
    <QueryClientProvider client={queryClient}>
      <ClientProvider client={boot.client} connection={boot.connection}>
        <RouterProvider router={router} />
      </ClientProvider>
    </QueryClientProvider>
  );
}

/** The pre-router boot screen: connecting spinner or a clear, recoverable error. */
function BootScreen({ state, message }: { state: 'starting' | 'error'; message?: string }) {
  return (
    <div style={{ minHeight: '100vh', display: 'grid', placeItems: 'center', padding: 32 }}>
      <div
        style={{
          width: 'min(480px, 100%)',
          background: 'var(--surface-raised)',
          border: '1px solid var(--border-hairline)',
          borderRadius: 'var(--radius-lg)',
          padding: 28,
          textAlign: 'center',
        }}
      >
        <h1
          style={{
            margin: '0 0 6px',
            fontFamily: 'var(--font-display)',
            fontWeight: 800,
            fontSize: 'var(--text-title)',
            color: 'var(--text-primary)',
          }}
        >
          ComicHub
        </h1>
        {state === 'starting' ? (
          <p style={{ color: 'var(--text-secondary)', margin: 0 }}>Connecting to your library…</p>
        ) : (
          <div>
            <p style={{ color: 'var(--text-secondary)', margin: '0 0 12px' }}>
              Could not reach the server.
            </p>
            <p
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 'var(--text-small)',
                color: 'var(--danger)',
                margin: '0 0 16px',
                wordBreak: 'break-word',
              }}
            >
              {message}
            </p>
            <p style={{ fontSize: 'var(--text-small)', color: 'var(--text-tertiary)', margin: 0 }}>
              {isTauri()
                ? 'The embedded server did not start. Try relaunching the app.'
                : 'Start the dev server (see CLAUDE notes) or set VITE_SERVER_URL, then reload.'}
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
