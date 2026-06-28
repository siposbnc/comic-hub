//! ComicHub reader Tauri core. In Phase 0 it captures the file path the reader was
//! launched with (file-association double-click) so the webview can show what it will
//! open. Direct archive reading + the connected-mode page stream arrive in Phase 1.

mod formats;
mod formats_gen;
mod local;

use std::path::Path;
use std::process::Command;

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

/// Opens the OS file picker for a comic archive and returns the chosen absolute path, or
/// `None` if the user cancelled. Shells out to the platform's native dialog so the crate's
/// dependency surface stays unchanged (mirrors the client's `pick_folder`).
#[tauri::command]
fn pick_comic_file() -> Result<Option<String>, String> {
    #[cfg(target_os = "windows")]
    {
        // A WinForms OpenFileDialog driven by PowerShell in an STA thread. Prints the
        // selected path on stdout, nothing on cancel.
        let script = "Add-Type -AssemblyName System.Windows.Forms; \
            $f = New-Object System.Windows.Forms.OpenFileDialog; \
            $f.Title = 'Open a comic'; \
            $f.Filter = 'Comic archives (*.cbz;*.cbr;*.cb7;*.cbt;*.zip;*.tar)|*.cbz;*.cbr;*.cb7;*.cbt;*.zip;*.tar|All files (*.*)|*.*'; \
            if ($f.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) { \
            [Console]::Out.Write($f.FileName) }";
        let out = Command::new("powershell")
            .args(["-NoProfile", "-STA", "-NonInteractive", "-Command", script])
            .output()
            .map_err(|e| format!("open file dialog: {e}"))?;
        let path = String::from_utf8_lossy(&out.stdout).trim().to_string();
        return Ok(if path.is_empty() { None } else { Some(path) });
    }

    #[cfg(target_os = "macos")]
    {
        let script = "POSIX path of (choose file with prompt \"Open a comic\" \
            of type {\"cbz\",\"cbr\",\"cb7\",\"cbt\",\"zip\",\"tar\"})";
        let out = Command::new("osascript")
            .args(["-e", script])
            .output()
            .map_err(|e| format!("open file dialog: {e}"))?;
        let path = String::from_utf8_lossy(&out.stdout).trim().to_string();
        return Ok(if path.is_empty() { None } else { Some(path) });
    }

    #[cfg(all(unix, not(target_os = "macos")))]
    {
        let out = Command::new("zenity")
            .args([
                "--file-selection",
                "--title=Open a comic",
                "--file-filter=Comics | *.cbz *.cbr *.cb7 *.cbt *.zip *.tar",
            ])
            .output()
            .map_err(|e| format!("open file dialog: {e}"))?;
        let path = String::from_utf8_lossy(&out.stdout).trim().to_string();
        return Ok(if path.is_empty() { None } else { Some(path) });
    }

    #[allow(unreachable_code)]
    Ok(None)
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
            pick_comic_file,
            local::local_open,
            local::local_manifest,
            local::local_page,
            local::local_thumb,
            local::local_prefetch,
            local::local_save_progress,
            local::local_restore_progress,
            local::local_save_prefs,
            local::local_restore_prefs
        ])
        .run(tauri::generate_context!())
        .expect("error while running ComicHub reader");
}
