mod commands;
mod server;

use tauri::{Manager, RunEvent};

use server::ServerProcess;

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .manage(ServerProcess::default())
        .invoke_handler(tauri::generate_handler![
            server::start_server,
            server::stop_server,
            commands::pick_folder,
            commands::launch_reader
        ])
        .build(tauri::generate_context!())
        .expect("error while building ComicHub client")
        // Kill the spawned server sidecar when the app exits so it never outlives the
        // client (orphaned servers lock their binary and the database on the next run).
        .run(|app, event| {
            if let RunEvent::Exit = event {
                if let Some(state) = app.try_state::<ServerProcess>() {
                    if let Ok(mut guard) = state.0.lock() {
                        if let Some(mut child) = guard.take() {
                            let _ = child.kill();
                        }
                    }
                }
            }
        });
}
