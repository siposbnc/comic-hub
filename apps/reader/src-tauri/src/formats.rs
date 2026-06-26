//! The canonical set of comic formats the reader can open — the single source of truth
//! on the Rust side. Keep in sync with the spec (docs/00-overview.md §6), the frontend's
//! `@comichub/reader-core` SUPPORTED_FORMATS, and `domain.SupportedFormats` on the server.
//! Don't hardcode format lists elsewhere in this crate.

/// Lowercase extensions (without the dot) the reader can open. PDF is openable here even
/// though it isn't auto-registered as a file association (see tauri.conf.json).
pub const SUPPORTED_EXTENSIONS: &[&str] = &["cbz", "cbr", "cb7", "cbt", "pdf"];

/// Returns true if `ext` (with or without a leading dot, any case) is a supported format.
pub fn is_supported(ext: &str) -> bool {
    let e = ext.trim_start_matches('.').to_ascii_lowercase();
    SUPPORTED_EXTENSIONS.iter().any(|s| *s == e)
}
