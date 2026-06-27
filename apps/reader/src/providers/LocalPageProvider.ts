import { invoke } from '@tauri-apps/api/core';
import type { Manifest, PageOpts, PageProvider, Progress } from '@comichub/reader-core';

/**
 * LocalPageProvider — standalone mode. Talks to the Rust core (src-tauri) which opens
 * the archive on disk, lists + natural-sorts image entries, returns page bytes, and
 * persists progress to a local store keyed by content hash. No server involved.
 *
 * The matching Tauri commands live in apps/reader/src-tauri/src/local.rs.
 */
export class LocalPageProvider implements PageProvider {
  private cachedManifest: Manifest | null = null;

  constructor(private readonly bookId: string) {}

  /** Opens the archive once and returns its manifest; the content hash becomes bookId. */
  static async open(path: string): Promise<{ provider: LocalPageProvider; manifest: Manifest }> {
    const manifest = await invoke<Manifest>('local_open', { path });
    const provider = new LocalPageProvider(manifest.bookId);
    provider.cachedManifest = manifest;
    return { provider, manifest };
  }

  async manifest(): Promise<Manifest> {
    if (this.cachedManifest) return this.cachedManifest;
    this.cachedManifest = await invoke<Manifest>('local_manifest', { bookId: this.bookId });
    return this.cachedManifest;
  }

  async page(idx: number, _opts?: PageOpts): Promise<Blob> {
    const bytes = await invoke<ArrayBuffer>('local_page', { idx });
    return new Blob([bytes]);
  }

  async thumb(idx: number): Promise<Blob> {
    const bytes = await invoke<ArrayBuffer>('local_thumb', { idx });
    return new Blob([bytes]);
  }

  prefetch(from: number, count: number): void {
    // Hint the Rust core to pre-extract upcoming pages; safe to ignore failures.
    void invoke('local_prefetch', { from, count }).catch(() => undefined);
  }

  saveProgress(progress: Progress): void {
    void invoke('local_save_progress', { progress }).catch(() => undefined);
  }

  async restoreProgress(): Promise<Progress | null> {
    try {
      return await invoke<Progress | null>('local_restore_progress', { bookId: this.bookId });
    } catch {
      return null;
    }
  }
}
