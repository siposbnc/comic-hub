//! Thin OS-integration commands the client webview invokes. These stay dependency-free
//! on purpose — they shell out to the platform's own facilities so the crate's
//! dependency surface (and lockfile) is unchanged.

use std::process::Command;

/// Opens the OS folder picker and returns the chosen absolute path, or `None` if the
/// user cancelled. The web build has no native picker and falls back to a typed path.
#[tauri::command]
pub fn pick_folder() -> Result<Option<String>, String> {
    #[cfg(target_os = "windows")]
    {
        // A WinForms FolderBrowserDialog driven by PowerShell in an STA thread. Prints the
        // selected path on stdout, nothing on cancel.
        let script = "Add-Type -AssemblyName System.Windows.Forms; \
            $f = New-Object System.Windows.Forms.FolderBrowserDialog; \
            $f.Description = 'Choose a comics folder'; \
            if ($f.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) { \
            [Console]::Out.Write($f.SelectedPath) }";
        let out = Command::new("powershell")
            .args(["-NoProfile", "-STA", "-NonInteractive", "-Command", script])
            .output()
            .map_err(|e| format!("open folder dialog: {e}"))?;
        let path = String::from_utf8_lossy(&out.stdout).trim().to_string();
        return Ok(if path.is_empty() { None } else { Some(path) });
    }

    #[cfg(target_os = "macos")]
    {
        let script = "POSIX path of (choose folder with prompt \"Choose a comics folder\")";
        let out = Command::new("osascript")
            .args(["-e", script])
            .output()
            .map_err(|e| format!("open folder dialog: {e}"))?;
        let path = String::from_utf8_lossy(&out.stdout).trim().to_string();
        return Ok(if path.is_empty() { None } else { Some(path) });
    }

    #[cfg(all(unix, not(target_os = "macos")))]
    {
        // Best-effort via zenity if present; otherwise the UI uses the typed-path fallback.
        let out = Command::new("zenity")
            .args(["--file-selection", "--directory", "--title=Choose a comics folder"])
            .output()
            .map_err(|e| format!("open folder dialog: {e}"))?;
        let path = String::from_utf8_lossy(&out.stdout).trim().to_string();
        return Ok(if path.is_empty() { None } else { Some(path) });
    }

    #[allow(unreachable_code)]
    Ok(None)
}

/// Hands the reader deep link to the OS so the registered `comichub-reader://` handler
/// (the desktop reader) opens the book. `server`/`token`/`book_id`/`page` are accepted for
/// forward-compatibility (e.g. spawning the reader binary directly) but the deep link is
/// the canonical path today.
#[tauri::command]
pub fn launch_reader(
    url: String,
    _server: Option<String>,
    _token: Option<String>,
    _book_id: Option<String>,
    _page: Option<u32>,
) -> Result<(), String> {
    open_url(&url)
}

/// Opens a URL/URI with the platform's default handler.
fn open_url(url: &str) -> Result<(), String> {
    #[cfg(target_os = "windows")]
    let result = Command::new("cmd").args(["/C", "start", "", url]).spawn();

    #[cfg(target_os = "macos")]
    let result = Command::new("open").arg(url).spawn();

    #[cfg(all(unix, not(target_os = "macos")))]
    let result = Command::new("xdg-open").arg(url).spawn();

    result.map(|_| ()).map_err(|e| format!("open reader link: {e}"))
}
