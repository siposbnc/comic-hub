import { createContext, useContext, type ReactNode } from 'react';
import { ComicHubClient, type Connection } from '@comichub/api-client';

/** What the app needs to talk to the server: the typed client plus the raw connection
 *  (the WebSocket and reader deep links need the base URL + token directly). */
export interface ClientContextValue {
  client: ComicHubClient;
  connection: Connection;
}

const ClientContext = createContext<ClientContextValue | null>(null);

export function ClientProvider({
  client,
  connection,
  children,
}: {
  client: ComicHubClient;
  connection: Connection;
  children: ReactNode;
}) {
  return <ClientContext.Provider value={{ client, connection }}>{children}</ClientContext.Provider>;
}

/** The typed server client. Throws if used outside the provider so misuse fails loud. */
export function useClient(): ComicHubClient {
  const ctx = useContext(ClientContext);
  if (!ctx) throw new Error('useClient must be used within <ClientProvider>');
  return ctx.client;
}

/** The raw connection descriptor (base URL + token) for non-REST surfaces (WS, deep links). */
export function useConnection(): Connection {
  const ctx = useContext(ClientContext);
  if (!ctx) throw new Error('useConnection must be used within <ClientProvider>');
  return ctx.connection;
}
