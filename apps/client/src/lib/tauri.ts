import { invoke } from '@tauri-apps/api/core';
import { isTauri } from '../connection.js';

/**
 * Opens the OS folder picker (Tauri only) and returns the chosen absolute path, or
 * null if the user cancelled. On the web there is no native picker, so callers fall
 * back to a typed path field.
 */
export async function pickFolder(): Promise<string | null> {
  if (!isTauri()) return null;
  const path = await invoke<string | null>('pick_folder');
  return path ?? null;
}
