import { invoke } from '@tauri-apps/api/core';
import type { Connection } from '@comichub/api-client';

/** Shape returned by the Rust `start_server` command. */
interface RustConnection {
  base_url: string;
  token: string;
  port: number;
  pid: number;
}

/** True when running inside the Tauri webview (vs a plain browser dev server). */
export function isTauri(): boolean {
  return typeof window !== 'undefined' && '__TAURI_INTERNALS__' in window;
}

/** Asks the Rust core to spawn (or attach to) the embedded sidecar and returns the
 *  connection descriptor from its handshake. */
export async function startEmbeddedServer(): Promise<Connection> {
  const c = await invoke<RustConnection>('start_server');
  return { baseUrl: c.base_url, token: c.token, port: c.port, pid: c.pid };
}

export async function stopEmbeddedServer(): Promise<void> {
  await invoke('stop_server');
}

/** One LAN server found via mDNS (shape returned by the Rust `discover_servers` command). */
export interface DiscoveredServer {
  /** Human-readable server name (the server's `--server-name`, default its hostname). */
  name: string;
  /** Ready-to-connect base URL, e.g. `http://192.168.1.10:8080`. */
  url: string;
  host: string;
  port: number;
  version: string | null;
  /** Whether the server requires login, so the connect screen can label the row. */
  auth_required: boolean;
}

/**
 * Browses the LAN for ComicHub servers advertising over mDNS (Milestone D). Resolves
 * after the browse window (default 2.5s) with everything found — possibly empty. Only
 * meaningful inside Tauri; a plain browser can't multicast, so it resolves to [].
 */
export async function discoverServers(timeoutMs?: number): Promise<DiscoveredServer[]> {
  if (!isTauri()) return [];
  return invoke<DiscoveredServer[]>('discover_servers', { timeoutMs });
}

/** Web-dev fallback: talk to a server started by hand (see CLAUDE run notes). Auth is
 *  off in `--mode=server`, so the token is empty. */
function webConnection(): Connection {
  const baseUrl =
    (import.meta.env.VITE_SERVER_URL as string | undefined) || 'http://127.0.0.1:8099';
  return { baseUrl: baseUrl.replace(/\/$/, ''), token: '' };
}

/**
 * Resolves the connection for the current host: the embedded sidecar inside Tauri, or
 * the standalone dev server in a plain browser. The single entry point the app boots
 * from.
 */
export async function resolveConnection(): Promise<Connection> {
  if (isTauri()) {
    return startEmbeddedServer();
  }
  return webConnection();
}
