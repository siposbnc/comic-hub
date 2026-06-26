mod server;

use server::ServerProcess;

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .manage(ServerProcess::default())
        .invoke_handler(tauri::generate_handler![
            server::start_server,
            server::stop_server
        ])
        .run(tauri::generate_context!())
        .expect("error while running ComicHub client");
}
