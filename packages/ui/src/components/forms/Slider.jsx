import React from 'react';

/** Slider — cyan fill track. Cover-size density (S/M/L), brightness, prefetch depth. */
export function Slider({ value = 50, min = 0, max = 100, step = 1, onChange, disabled = false, style, ...rest }) {
  const pct = ((value - min) / (max - min)) * 100;
  return (
    <input
      type="range"
      value={value} min={min} max={max} step={step} disabled={disabled}
      onChange={(e) => onChange && onChange(Number(e.target.value))}
      style={{
        WebkitAppearance: 'none', appearance: 'none',
        width: '100%', height: 4, borderRadius: 'var(--radius-pill)',
        background: `linear-gradient(to right, var(--accent) ${pct}%, var(--ink-600) ${pct}%)`,
        outline: 'none', cursor: disabled ? 'not-allowed' : 'pointer', opacity: disabled ? 0.5 : 1,
        ...style,
      }}
      {...rest}
    />
  );
}
