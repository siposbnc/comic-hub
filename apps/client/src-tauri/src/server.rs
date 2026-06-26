//! Embedded-mode server supervision: spawn the bundled `comichub-server` sidecar,
//! wait for its handshake file, and hand the connection back to the webview. See
//! docs/01-architecture.md §3.

use std::fs;
use std::path::PathBuf;
use std::process::{Child, Command};
use std::sync::Mutex;
use std::time::{Duration, Instant};

use serde::{Deserialize, Serialize};
use tauri::{AppHandle, Manager, State};

/// Holds the spawned sidecar so we can stop it on exit.
#[derive(Default)]
pub struct ServerProcess(pub Mutex<Option<Child>>);

/// Connection descriptor returned to the frontend.
#[derive(Serialize, Clone)]
pub struct Connection {
    pub base_url: String,
    pub token: String,
    pub port: u16,
    pub pid: u32,
}

/// The handshake file the server writes (see server connection.Handshake).
#[derive(Deserialize)]
struct Handshake {
    port: u16,
    token: String,
    pid: u32,
    #[serde(rename = "baseUrl")]
    base_url: String,
}

#[tauri::command]
pub fn start_server(
    app: AppHandle,
    state: State<'_, ServerProcess>,
) -> Result<Connection, String> {
    let data_dir = app
        .path()
        .app_data_dir()
        .map_err(|e| format!("resolve app data dir: {e}"))?;
    fs::create_dir_all(&data_dir).map_err(|e| format!("create data dir: {e}"))?;

    let handshake = data_dir.join("connection.json");
    let _ = fs::remove_file(&handshake); // stale handshake from a previous run

    let bin = resolve_server_bin()?;
    let child = Command::new(&bin)
        .arg("--mode=embedded")
        .arg(format!("--data-dir={}", data_dir.display()))
        .arg(format!("--handshake-file={}", handshake.display()))
        .spawn()
        .map_err(|e| format!("spawn server ({}): {e}", bin.display()))?;

    *state.0.lock().unwrap() = Some(child);

    let deadline = Instant::now() + Duration::from_secs(10);
    loop {
        if handshake.exists() {
            let data =
                fs::read_to_string(&handshake).map_err(|e| format!("read handshake: {e}"))?;
            let hs: Handshake =
                serde_json::from_str(&data).map_err(|e| format!("parse handshake: {e}"))?;
            return Ok(Connection {
                base_url: hs.base_url,
                token: hs.token,
                port: hs.port,
                pid: hs.pid,
            });
        }
        if Instant::now() > deadline {
            return Err("server did not publish its handshake within 10s".into());
        }
        std::thread::sleep(Duration::from_millis(100));
    }
}

#[tauri::command]
pub fn stop_server(state: State<'_, ServerProcess>) -> Result<(), String> {
    if let Some(mut child) = state.0.lock().unwrap().take() {
        let _ = child.kill();
    }
    Ok(())
}

/// Locate the server binary. Resolution order:
/// 1. `COMICHUB_SERVER_BIN` env override (CI / power users).
/// 2. Dev path: `<repo>/server/bin/comichub-server[.exe]` relative to the Tauri crate.
/// 3. (Phase 1) bundled sidecar resource next to the app binary.
fn resolve_server_bin() -> Result<PathBuf, String> {
    if let Ok(p) = std::env::var("COMICHUB_SERVER_BIN") {
        return Ok(PathBuf::from(p));
    }

    let exe = if cfg!(windows) {
        "comichub-server.exe"
    } else {
        "comichub-server"
    };

    // `tauri dev` runs with CWD = apps/client/src-tauri
    let dev = PathBuf::from("../../../server/bin").join(exe);
    if dev.exists() {
        return Ok(dev);
    }

    // Next to the current executable (bundled layout).
    if let Ok(mut here) = std::env::current_exe() {
        here.pop();
        let bundled = here.join(exe);
        if bundled.exists() {
            return Ok(bundled);
        }
    }

    Err(format!(
        "server binary not found. Build it (`go build` in /server) or set COMICHUB_SERVER_BIN. Looked for {exe} under ../../../server/bin and next to the app."
    ))
}
