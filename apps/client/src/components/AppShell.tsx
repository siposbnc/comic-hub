import { Outlet } from '@tanstack/react-router';
import { Sidebar } from './Sidebar.js';
import { TopBar } from './TopBar.js';
import { ToastHost } from './ToastHost.js';
import { useServerEvents } from '../lib/events.js';

/** App frame: fixed sidebar + utility bar around the routed content. Also the single
 *  mount point for the live server-event socket and the toast stack. */
export function AppShell() {
  useServerEvents();

  return (
    <div style={{ display: 'flex', height: '100vh', overflow: 'hidden' }}>
      <Sidebar />
      <div style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>
        <TopBar />
        <main style={{ flex: 1, minHeight: 0, overflowY: 'auto' }}>
          <Outlet />
        </main>
      </div>
      <ToastHost />
    </div>
  );
}
