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
    let mut builder = tauri::Builder::default();

    // Single-instance must be registered first so a second launch (the OS opening a
    // comichub-reader:// link while we're already running) re-focuses this instance and
    // forwards the URL to the deep-link plugin instead of spawning a new window.
    #[cfg(any(target_os = "windows", target_os = "linux"))]
    {
        use tauri::{Emitter, Manager};
        builder = builder.plugin(tauri_plugin_single_instance::init(|app, argv, _cwd| {
            if let Some(window) = app.get_webview_window("main") {
                let _ = window.set_focus();
            }
            // A second launch carrying a comichub-reader:// link (the client's one-click
            // Read while we're already open) re-focuses this instance; forward the URL to
            // the running webview so it swaps to the new book instead of doing nothing.
            if let Some(url) = argv.iter().find(|a| a.starts_with("comichub-reader://")) {
                let _ = app.emit("reader://open-url", url.clone());
            }
        }));
    }

    builder
        .plugin(tauri_plugin_deep_link::init())
        .setup(|_app| {
            // Register the scheme at runtime so it also works in dev / unpackaged runs.
            // Installed builds register it via tauri.conf.json plugins.deep-link.
            #[cfg(any(target_os = "windows", target_os = "linux"))]
            {
                use tauri_plugin_deep_link::DeepLinkExt;
                let _ = _app.deep_link().register("comichub-reader");
            }
            Ok(())
        })
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
