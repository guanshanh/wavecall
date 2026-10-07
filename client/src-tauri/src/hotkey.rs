use tauri::App;

/// Register global hotkeys for Push-to-Talk and mute toggle.
pub fn register_hotkeys(_app: &App) -> Result<(), Box<dyn std::error::Error>> {
    // TODO: implement Push-to-Talk hotkey
    //
    // Example using tauri-plugin-global-shortcut:
    //
    // use tauri_plugin_global_shortcut::{Code, Modifiers, Shortcut, ShortcutState};
    //
    // let shortcut = Shortcut::new(Some(Modifiers::CONTROL), Code::KeyM);
    // app.global_shortcut().on_shortcut(shortcut, move |_app, _shortcut, event| {
    //     if event.state == ShortcutState::Pressed {
    //         // emit mute toggle event to frontend
    //     }
    // })?;

    Ok(())
}
