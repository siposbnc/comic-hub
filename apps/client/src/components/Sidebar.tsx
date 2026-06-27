import { useNavigate, useRouterState } from '@tanstack/react-router';
import { Icon, SidebarItem, SidebarSection } from '@comichub/ui';
import { useLibraries } from '../lib/queries.js';

/** Left rail: Home, the live library list, and Settings. 232px, full height. */
export function Sidebar() {
  const navigate = useNavigate();
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const { data: libraries, isLoading } = useLibraries();

  return (
    <nav
      aria-label="Primary"
      style={{
        width: 'var(--sidebar-width, 232px)',
        flex: 'none',
        height: '100vh',
        borderRight: '1px solid var(--border-hairline)',
        background: 'var(--surface-raised)',
        display: 'flex',
        flexDirection: 'column',
        padding: '14px 12px',
        gap: 2,
        overflowY: 'auto',
      }}
    >
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 9,
          padding: '6px 10px 14px',
        }}
      >
        <Icon name="layers" size={20} color="var(--accent)" />
        <span
          style={{
            fontFamily: 'var(--font-display)',
            fontWeight: 800,
            fontSize: '1.05rem',
            letterSpacing: '0.01em',
            color: 'var(--text-primary)',
          }}
        >
          ComicHub
        </span>
      </div>

      <SidebarItem
        icon="home"
        label="Home"
        active={pathname === '/'}
        onClick={() => navigate({ to: '/' })}
      />

      <SidebarSection>Libraries</SidebarSection>
      {isLoading && (
        <div
          style={{
            padding: '6px 10px',
            color: 'var(--text-tertiary)',
            fontSize: 'var(--text-small)',
          }}
        >
          Loading…
        </div>
      )}
      {libraries && libraries.length === 0 && (
        <div
          style={{
            padding: '6px 10px',
            color: 'var(--text-tertiary)',
            fontSize: 'var(--text-small)',
          }}
        >
          No libraries yet
        </div>
      )}
      {libraries?.map((lib) => (
        <SidebarItem
          key={lib.id}
          icon="library"
          label={lib.name}
          active={pathname === `/library/${lib.id}`}
          onClick={() => navigate({ to: '/library/$id', params: { id: lib.id } })}
        />
      ))}

      <div style={{ flex: 1 }} />

      <SidebarItem
        icon="settings"
        label="Settings"
        active={pathname === '/settings'}
        onClick={() => navigate({ to: '/settings' })}
      />
    </nav>
  );
}
