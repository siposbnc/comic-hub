import React from 'react';
import { Icon } from '../core/Icon';

/**
 * CoverCard — the atom of the whole app. A square-cornered cover with the print
 * signatures: registration ticks on hover/focus, a mono spine tab clipped lower-left
 * (magenta unread / hollow read / cyan reading), a read-progress underline, and an
 * optional multiselect checkbox. No drop shadow — gallery, not skeuomorph.
 */
const WIDTHS = { s: 'var(--cover-w-s)', m: 'var(--cover-w-m)', l: 'var(--cover-w-l)' };

export function CoverCard({
  cover, title, subtitle, number,
  status = 'unread',     // 'unread' | 'reading' | 'read'
  progress = 0,          // 0..1, shown when status === 'reading'
  size = 'm',
  selectable = false,
  selected = false,
  onSelect,
  onClick,
  style,
}) {
  const [hover, setHover] = React.useState(false);
  const showProgress = status === 'reading' && progress > 0;

  return (
    <div
      style={{ width: WIDTHS[size] || WIDTHS.m, display: 'flex', flexDirection: 'column', gap: 7, ...style }}
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
    >
      <div
        className="ch-reg"
        tabIndex={0}
        onClick={onClick}
        style={{
          position: 'relative', cursor: 'pointer',
          aspectRatio: 'var(--cover-aspect)',
          background: cover ? `var(--surface-cover) center/cover no-repeat` : 'var(--surface-cover)',
          backgroundImage: cover ? `url(${cover})` : undefined,
          backgroundSize: 'cover', backgroundPosition: 'center',
          outline: selected ? '2px solid var(--accent)' : 'none',
          outlineOffset: -2,
          transform: hover ? 'translateY(-2px)' : 'translateY(0)',
          transition: 'transform var(--dur-fast) var(--ease-standard)',
        }}
      >
        {/* Fallback masthead when no cover image */}
        {!cover && (
          <div style={{ position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', padding: 12, background: 'linear-gradient(155deg, var(--ink-700), var(--ink-900))' }}>
            <span style={{ fontFamily: 'var(--font-display)', fontWeight: 800, fontSize: '1rem', lineHeight: 1.05, color: 'var(--paper-100)' }}>{title}</span>
          </div>
        )}

        {/* Multiselect checkbox — appears on hover or when any selected */}
        {selectable && (hover || selected) && (
          <button
            type="button"
            aria-label={selected ? 'Deselect' : 'Select'}
            onClick={(e) => { e.stopPropagation(); onSelect && onSelect(!selected); }}
            style={{
              position: 'absolute', top: 7, left: 7,
              width: 22, height: 22, display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
              borderRadius: 'var(--radius-sm)',
              background: selected ? 'var(--accent)' : 'color-mix(in oklab, var(--ink-900) 60%, transparent)',
              border: `1px solid ${selected ? 'var(--accent)' : 'var(--paper-100)'}`,
              color: selected ? 'var(--text-on-accent)' : 'var(--paper-100)',
              cursor: 'pointer', backdropFilter: 'blur(2px)',
            }}
          >
            {selected && <Icon name="check" size={13} />}
          </button>
        )}

        {/* Spine tab — the longbox divider / issue number */}
        {number != null && (
          <span className="ch-spine-tab" data-state={status === 'read' ? 'read' : status === 'reading' ? 'reading' : undefined}>
            {number}
          </span>
        )}

        {/* Read-progress underline */}
        {showProgress && (
          <div className="ch-progress" style={{ position: 'absolute', left: 0, right: 0, bottom: 0 }}>
            <span style={{ width: `${Math.round(progress * 100)}%` }} />
          </div>
        )}
      </div>

      {(title || subtitle) && (
        <div style={{ minWidth: 0 }}>
          {title && <div style={{ fontFamily: 'var(--font-body)', fontWeight: 600, fontSize: 'var(--text-small)', color: 'var(--text-primary)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{title}</div>}
          {subtitle && <div style={{ fontFamily: 'var(--font-mono)', fontSize: '0.68rem', color: 'var(--text-tertiary)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', marginTop: 1 }}>{subtitle}</div>}
        </div>
      )}
    </div>
  );
}
