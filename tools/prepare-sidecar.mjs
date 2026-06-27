// Prepares the comichub-server binary as a Tauri "externalBin" sidecar for the client.
// Tauri requires the binary to exist at build time named with the Rust target triple
// (e.g. binaries/comichub-server-x86_64-pc-windows-msvc.exe); at bundle time it is
// copied next to the app executable (suffix stripped), where server.rs finds it.
//
// Run from anywhere; paths are resolved relative to the repo root. Invoked by the
// client's tauri.conf.json `beforeBuildCommand`.
import { execFileSync } from 'node:child_process';
import { copyFileSync, mkdirSync } from 'node:fs';
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
const serverBin = join(repoRoot, 'server', 'bin', `comichub-server${ext}`);
const outDir = join(repoRoot, 'apps', 'client', 'src-tauri', 'binaries');
const outBin = join(outDir, `comichub-server-${triple}${ext}`);

console.log(`[prepare-sidecar] building server (${triple})…`);
execFileSync('go', ['build', '-C', join(repoRoot, 'server'), '-o', join('bin', `comichub-server${ext}`), './cmd/comichub-server'], {
  stdio: 'inherit',
});

mkdirSync(outDir, { recursive: true });
copyFileSync(serverBin, outBin);
console.log(`[prepare-sidecar] sidecar ready: ${outBin}`);
