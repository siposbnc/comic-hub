//! Format helpers for the reader. The canonical extension list lives in the generated
//! `formats_gen.rs` (from tools/codegen/formats.json) — keep logic here, data there.

use crate::formats_gen::SUPPORTED_EXTENSIONS;

/// Returns true if `ext` (with or without a leading dot, any case) is a supported format.
pub fn is_supported(ext: &str) -> bool {
    let e = ext.trim_start_matches('.').to_ascii_lowercase();
    SUPPORTED_EXTENSIONS.iter().any(|s| *s == e)
}
