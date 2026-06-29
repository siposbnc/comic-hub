// Builds the comichub-server binary and places it where the client expects it:
//   • server/bin/comichub-server[.exe]  — used by `tauri dev` (see client server.rs)
//   • apps/client/src-tauri/binaries/comichub-server-<triple>[.exe] — the Tauri
//     "externalBin" sidecar bundled into packaged builds.
//
// Invoked by the client's tauri.conf.json before{Dev,Build}Command. On Windows it first
// stops any stray comichub-server processes (a hard-killed or crashed dev client can orphan
// its sidecar, and a running .exe is locked — so a leftover server would otherwise keep this
// script copying the OLD binary and new endpoints would 404). It builds to a unique temp file
// first, then copies into both targets. If a target is somehow still in use, we keep the
// existing binary and warn rather than failing the build.
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

// On Windows, stop any leftover servers so the build can replace the (otherwise locked)
// binaries. Best-effort: taskkill exits non-zero when nothing matches, which we ignore.
function killStrayServers() {
  if (!isWin) return; // POSIX can overwrite a running binary; no lock trap to clear
  for (const name of [`comichub-server${ext}`, `comichub-server-${triple}${ext}`]) {
    try {
      execFileSync('taskkill', ['/IM', name, '/F'], { stdio: 'ignore' });
      console.log(`[prepare-sidecar] stopped stray ${name}`);
    } catch {
      // none running — fine
    }
  }
}

killStrayServers();

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
