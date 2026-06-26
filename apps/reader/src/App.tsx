import { useEffect, useState } from 'react';
import { invoke } from '@tauri-apps/api/core';

function isTauri(): boolean {
  return typeof window !== 'undefined' && '__TAURI_INTERNALS__' in window;
}

/**
 * Phase 0 reader: detect how it was launched. A path argument (file-association
 * double-click) means standalone mode; no path means it was opened from the client
 * (connected mode arrives in Phase 1). Rendering of pages lands in Phase 1.
 */
export function App() {
  const [openPath, setOpenPath] = useState<string | null>(null);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    (async () => {
      if (isTauri()) {
        try {
          const path = await invoke<string | null>('get_open_path');
          setOpenPath(path);
        } catch {
          setOpenPath(null);
        }
      }
      setReady(true);
    })();
  }, []);

  const mode = openPath ? 'Standalone' : 'Connected (pending)';

  return (
    <div className="reader">
      <div>
        <p className="mode">{ready ? mode : '…'}</p>
        <h1>ComicHub Reader</h1>
        {openPath ? (
          <>
            <p>Ready to open:</p>
            <p className="path">{openPath}</p>
          </>
        ) : (
          <p className="path">No file provided. Double-click a .cbz, or launch from the client.</p>
        )}
      </div>
    </div>
  );
}
