use tauri::{
    menu::{Menu, MenuItem},
    tray::TrayIconBuilder,
    App,
};

/// Create the system tray icon and menu.
pub fn create_tray(app: &App) -> Result<(), Box<dyn std::error::Error>> {
    let toggle_mute = MenuItem::with_id(app, "toggle_mute", "静音/取消静音", true, None::<&str>)?;
    let quit = MenuItem::with_id(app, "quit", "退出", true, None::<&str>)?;
    let menu = Menu::with_items(app, &[&toggle_mute, &quit])?;

    TrayIconBuilder::new()
        .menu(&menu)
        .tooltip("Wavecall")
        .on_menu_event(|app, event| match event.id.as_ref() {
            "toggle_mute" => {
                // TODO: emit mute toggle event to frontend
                println!("Toggle mute from tray");
            }
            "quit" => {
                app.exit(0);
            }
            _ => {}
        })
        .build(app)?;

    Ok(())
}
