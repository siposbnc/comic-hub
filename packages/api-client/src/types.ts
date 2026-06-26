// Wire types shared by the client and reader. Mirrors docs/03-api.md. These are kept
// hand-written in Phase 0; later they can be generated from the server's OpenAPI spec.

export type ServerMode = 'embedded' | 'server';

export interface HealthStatus {
  status: 'ok';
  version: string;
}

export interface ServerCapabilities {
  avif: boolean;
  pdf: boolean;
  epub: boolean;
  multiuser: boolean;
}

export interface ServerInfo {
  name: string;
  version: string;
  commit: string;
  mode: ServerMode;
  capabilities: ServerCapabilities;
}

export interface ServerStats {
  libraries: number;
  series: number;
  books: number;
}

export type UserRole = 'owner' | 'admin' | 'member' | 'restricted';

export interface User {
  id: string;
  username: string;
  displayName: string;
  role: UserRole;
}

export interface AuthHandshakeResult {
  mode: ServerMode;
  user: User;
}

export interface ApiErrorBody {
  error: {
    code: string;
    message: string;
  };
}

/** Connection descriptor the client obtains from the embedded sidecar handshake. */
export interface Connection {
  baseUrl: string;
  token: string;
  port?: number;
  pid?: number;
  version?: string;
}
