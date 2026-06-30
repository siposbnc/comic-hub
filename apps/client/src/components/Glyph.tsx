/** A few Lucide-style glyphs the design-system `Icon` registry doesn't carry yet
 *  (server, log-out, shield) — same 24×24 / 1.5px-stroke language. Fold these into the DS
 *  Icon on the next design-system sync; until then they live here, app-local. */
const PATHS: Record<string, string[]> = {
  server: [
    'M5 3h14a2 2 0 0 1 2 2v3a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2Z',
    'M5 14h14a2 2 0 0 1 2 2v3a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-3a2 2 0 0 1 2-2Z',
    'M7 6.5h.01',
    'M7 17.5h.01',
  ],
  'log-out': ['M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4', 'M16 17l5-5-5-5', 'M21 12H9'],
  shield: ['M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10Z'],
};

export function Glyph({
  name,
  size = 18,
  color = 'currentColor',
}: {
  name: 'server' | 'log-out' | 'shield';
  size?: number;
  color?: string;
}) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke={color}
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      style={{ display: 'block', flex: 'none' }}
    >
      {(PATHS[name] ?? []).map((d, i) => (
        <path key={i} d={d} />
      ))}
    </svg>
  );
}
