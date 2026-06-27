//! ComicHub reader Tauri core. In Phase 0 it captures the file path the reader was
//! launched with (file-association double-click) so the webview can show what it will
//! open. Direct archive reading + the connected-mode page stream arrive in Phase 1.

mod formats;
mod formats_gen;
mod local;

use std::path::Path;

/// Returns the comic file path passed on the command line, if any. The first argument
/// whose extension is a supported comic format wins.
#[tauri::command]
fn get_open_path() -> Option<String> {
    std::env::args().skip(1).find(|arg| {
        Path::new(arg)
            .extension()
            .and_then(|e| e.to_str())
            .map(formats::is_supported)
            .unwrap_or(false)
    })
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .invoke_handler(tauri::generate_handler![
            get_open_path,
            local::local_open,
            local::local_manifest,
            local::local_page,
            local::local_thumb,
            local::local_prefetch,
            local::local_save_progress,
            local::local_restore_progress
        ])
        .run(tauri::generate_context!())
        .expect("error while running ComicHub reader");
}
