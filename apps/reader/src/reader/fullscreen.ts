/** Thin wrappers over the Fullscreen API; no-op safe when unsupported (e.g. some webviews). */

export function isFullscreen(): boolean {
  return typeof document !== 'undefined' && document.fullscreenElement != null;
}

export async function toggleFullscreen(): Promise<void> {
  if (isFullscreen()) {
    await exitFullscreen();
  } else {
    await requestFullscreen();
  }
}

export async function requestFullscreen(): Promise<void> {
  try {
    await document.documentElement.requestFullscreen?.();
  } catch {
    // Fullscreen can be blocked by the host; fail quietly.
  }
}

export async function exitFullscreen(): Promise<void> {
  try {
    if (document.fullscreenElement) await document.exitFullscreen?.();
  } catch {
    // ignore
  }
}
