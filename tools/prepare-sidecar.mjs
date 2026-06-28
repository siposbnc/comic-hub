// Builds the comichub-server binary and places it where the client expects it:
//   • server/bin/comichub-server[.exe]  — used by `tauri dev` (see client server.rs)
//   • apps/client/src-tauri/binaries/comichub-server-<triple>[.exe] — the Tauri
//     "externalBin" sidecar bundled into packaged builds.
//
// Invoked by the client's tauri.conf.json before{Dev,Build}Command. It builds to a unique
// temp file first (Windows locks a running .exe, so we never build straight onto a path a
// server might be holding open), then copies into both targets. If a target is in use by a
// still-running server, we keep the existing binary and warn rather than failing the build.
import { execFileSync } from 'node:child_process';
import { copyFileSync, mkdirSync, existsSync, rmSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const repoRoot = join(dirname(fileURLToPath(import.meta.url)), '..');
const isWin = process.platform === 'win32';
const ext = isWin ? '.exe' : '';

function hostTriple() {
  const out = execFileSync('rustc', ['-vV'], { encoding: 'utf8' });
  const m = out.match(/^host:\s*(.+)$/m);
  if (!m) throw new Error('could not determine Rust host triple from `rustc -vV`');
  return m[1].trim();
}

const triple = hostTriple();
const serverBinDir = join(repoRoot, 'server', 'bin');
const devBin = join(serverBinDir, `comichub-server${ext}`); // tauri dev
const sidecarDir = join(repoRoot, 'apps', 'client', 'src-tauri', 'binaries');
const sidecarBin = join(sidecarDir, `comichub-server-${triple}${ext}`); // packaged
const tmpBin = join(serverBinDir, `.comichub-server-${process.pid}${ext}`);

mkdirSync(serverBinDir, { recursive: true });
mkdirSync(sidecarDir, { recursive: true });

console.log(`[prepare-sidecar] building server (${triple})…`);
execFileSync('go', ['build', '-C', join(repoRoot, 'server'), '-o', tmpBin, './cmd/comichub-server'], {
  stdio: 'inherit',
});

/** Copy the freshly built binary onto a target, tolerating a locked (in-use) destination. */
function place(dest) {
  try {
    copyFileSync(tmpBin, dest);
    console.log(`[prepare-sidecar] updated ${dest}`);
  } catch (err) {
    if (existsSync(dest)) {
      console.warn(
        `[prepare-sidecar] ${dest} is in use — keeping the existing binary. ` +
          `Close any running ComicHub to pick up server changes.`,
      );
    } else {
      rmSync(tmpBin, { force: true });
      throw err;
    }
  }
}

try {
  place(devBin);
  place(sidecarBin);
} finally {
  rmSync(tmpBin, { force: true });
}
