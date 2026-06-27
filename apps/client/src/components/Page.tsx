import type { ReactNode } from 'react';

/** Standard screen scaffold: gutter padding, a title row, and an actions slot. */
export function Page({
  title,
  eyebrow,
  actions,
  children,
}: {
  title: ReactNode;
  eyebrow?: ReactNode;
  actions?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div style={{ padding: 'var(--pad-screen, 32px)', maxWidth: 1320, margin: '0 auto' }}>
      <div
        style={{
          display: 'flex',
          alignItems: 'flex-end',
          justifyContent: 'space-between',
          gap: 16,
          marginBottom: 28,
        }}
      >
        <div style={{ minWidth: 0 }}>
          {eyebrow && (
            <div
              className="ch-mono"
              style={{
                fontSize: 'var(--text-label)',
                textTransform: 'uppercase',
                letterSpacing: 'var(--tracking-label)',
                color: 'var(--text-tertiary)',
                marginBottom: 6,
              }}
            >
              {eyebrow}
            </div>
          )}
          <h1
            style={{
              margin: 0,
              fontFamily: 'var(--font-display)',
              fontSize: 'var(--text-display-l)',
              lineHeight: 'var(--leading-display-l)',
              fontWeight: 800,
              letterSpacing: 'var(--tracking-tight)',
              color: 'var(--text-primary)',
            }}
          >
            {title}
          </h1>
        </div>
        {actions && <div style={{ flex: 'none', display: 'flex', gap: 10 }}>{actions}</div>}
      </div>
      {children}
    </div>
  );
}

/** Centered spinner used while a screen's primary query is loading. */
export function LoadingState({ label = 'Loading…' }: { label?: string }) {
  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        gap: 10,
        minHeight: 240,
        color: 'var(--text-tertiary)',
        fontFamily: 'var(--font-mono)',
        fontSize: 'var(--text-small)',
      }}
    >
      <span style={{ display: 'inline-flex', animation: 'ch-spin 1.1s linear infinite' }}>
        <svg
          width="16"
          height="16"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.5"
          strokeLinecap="round"
        >
          <path d="M3 12a9 9 0 0 1 15-6.7L21 8" />
          <path d="M21 3v5h-5" />
        </svg>
      </span>
      {label}
    </div>
  );
}

/** Inline error panel with a retry affordance. */
export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <div
      role="alert"
      style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        gap: 12,
        padding: '48px 32px',
        textAlign: 'center',
        color: 'var(--text-secondary)',
      }}
    >
      <div style={{ color: 'var(--danger)' }}>Something went wrong</div>
      <div style={{ fontSize: 'var(--text-small)', color: 'var(--text-tertiary)', maxWidth: 460 }}>
        {message}
      </div>
      {onRetry && (
        <button
          type="button"
          onClick={onRetry}
          style={{
            marginTop: 4,
            height: 34,
            padding: '0 16px',
            background: 'var(--surface-card)',
            color: 'var(--text-primary)',
            border: '1px solid var(--border-strong)',
            borderRadius: 'var(--radius-md)',
            cursor: 'pointer',
            fontFamily: 'var(--font-body)',
            fontSize: 'var(--text-small)',
          }}
        >
          Try again
        </button>
      )}
    </div>
  );
}
