// Hide the console window on Windows release builds (GUI app).
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod hotkey;
mod tray;

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_global_shortcut::Builder::new().build())
        .setup(|app| {
            tray::create_tray(app)?;
            hotkey::register_hotkeys(app)?;
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running wavecall");
}

fn main() {
    run();
}
