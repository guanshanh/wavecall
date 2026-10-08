use tauri::App;

/// Global PTT shortcuts are registered from the frontend via
/// `tauri-plugin-global-shortcut` (Pressed/Released). No business logic here.
pub fn register_hotkeys(_app: &App) -> Result<(), Box<dyn std::error::Error>> {
    Ok(())
}
