/** Window controls for the reader. Prefers the native Tauri window — so window size and
 *  fullscreen are remembered across launches by tauri-plugin-window-state — and falls back
 *  to the HTML5 Fullscreen API / a best-effort window.close() when running in a plain
 *  browser (dev, demo). No-op safe when a call is unsupported or the host refuses it. */
import { getCurrentWindow } from '@tauri-apps/api/window';

const inTauri = typeof window !== 'undefined' && '__TAURI_INTERNALS__' in window;

export async function isFullscreen(): Promise<boolean> {
  if (inTauri) {
    try {
      return await getCurrentWindow().isFullscreen();
    } catch {
      return false;
    }
  }
  return typeof document !== 'undefined' && document.fullscreenElement != null;
}

export async function setFullscreen(on: boolean): Promise<void> {
  if (inTauri) {
    try {
      await getCurrentWindow().setFullscreen(on);
    } catch {
      // Host may refuse; fail quietly.
    }
    return;
  }
  try {
    if (on) await document.documentElement.requestFullscreen?.();
    else if (document.fullscreenElement) await document.exitFullscreen?.();
  } catch {
    // Fullscreen can be blocked by the host; fail quietly.
  }
}

/** Close the reader window (quits the standalone app). */
export async function closeWindow(): Promise<void> {
  if (inTauri) {
    try {
      await getCurrentWindow().close();
      return;
    } catch {
      // ignore
    }
  }
  try {
    window.close();
  } catch {
    // ignore
  }
}
