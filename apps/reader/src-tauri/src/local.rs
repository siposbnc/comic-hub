//! Standalone-mode page provider. Opens a comic archive directly from disk (no server),
//! lists + natural-sorts image entries, serves page bytes and thumbnails, and persists
//! reading progress to a local JSON store keyed by content hash. See docs/06-reader.md §4.
//!
//! Supported here: CBZ (zip) and CBT (tar). CBR/CB7/PDF need extra native libraries and
//! are handled by the server in connected mode; standalone returns a clear error for them.

use std::cmp::Ordering;
use std::fs::File;
use std::io::Read;
use std::path::{Path, PathBuf};
use std::sync::{Mutex, OnceLock};

use serde::Serialize;
use sha2::{Digest, Sha256};
use tauri::ipc::Response;
use tauri::Manager;

const IMAGE_EXTS: &[&str] = &["jpg", "jpeg", "png", "gif", "webp", "bmp", "avif"];

#[derive(Clone, Copy, PartialEq, Debug)]
enum Format {
    Zip,
    Tar,
}

#[derive(Serialize, Clone)]
#[serde(rename_all = "camelCase")]
pub struct PageMeta {
    idx: usize,
    w: u32,
    h: u32,
    #[serde(skip_serializing_if = "Option::is_none")]
    r#type: Option<String>,
    double: bool,
}

#[derive(Serialize, Clone)]
#[serde(rename_all = "camelCase")]
pub struct Manifest {
    book_id: String,
    page_count: usize,
    reading_dir: String,
    pages: Vec<PageMeta>,
}

struct OpenBook {
    path: PathBuf,
    format: Format,
    entries: Vec<String>,
    manifest: Manifest,
}

fn open_state() -> &'static Mutex<Option<OpenBook>> {
    static OPEN: OnceLock<Mutex<Option<OpenBook>>> = OnceLock::new();
    OPEN.get_or_init(|| Mutex::new(None))
}

/// Natural, case-insensitive comparison so "page2" precedes "page10".
fn natural_cmp(a: &str, b: &str) -> Ordering {
    let a = a.to_lowercase();
    let b = b.to_lowercase();
    let mut ai = a.chars().peekable();
    let mut bi = b.chars().peekable();
    loop {
        match (ai.peek().copied(), bi.peek().copied()) {
            (None, None) => return Ordering::Equal,
            (None, Some(_)) => return Ordering::Less,
            (Some(_), None) => return Ordering::Greater,
            (Some(ca), Some(cb)) => {
                if ca.is_ascii_digit() && cb.is_ascii_digit() {
                    let mut na = String::new();
                    while let Some(c) = ai.peek().copied() {
                        if c.is_ascii_digit() {
                            na.push(c);
                            ai.next();
                        } else {
                            break;
                        }
                    }
                    let mut nb = String::new();
                    while let Some(c) = bi.peek().copied() {
                        if c.is_ascii_digit() {
                            nb.push(c);
                            bi.next();
                        } else {
                            break;
                        }
                    }
                    let ta = na.trim_start_matches('0');
                    let tb = nb.trim_start_matches('0');
                    let ord = ta.len().cmp(&tb.len()).then_with(|| ta.cmp(tb));
                    if ord != Ordering::Equal {
                        return ord;
                    }
                } else {
                    match ca.cmp(&cb) {
                        Ordering::Equal => {
                            ai.next();
                            bi.next();
                        }
                        o => return o,
                    }
                }
            }
        }
    }
}

fn is_image_entry(name: &str) -> bool {
    if name.ends_with('/') {
        return false;
    }
    let base = name.rsplit('/').next().unwrap_or(name);
    // Skip macOS metadata and hidden files.
    if base.starts_with('.') || name.contains("__MACOSX") {
        return false;
    }
    Path::new(base)
        .extension()
        .and_then(|e| e.to_str())
        .map(|e| IMAGE_EXTS.contains(&e.to_ascii_lowercase().as_str()))
        .unwrap_or(false)
}

fn detect_format(path: &Path) -> Result<Format, String> {
    match path
        .extension()
        .and_then(|e| e.to_str())
        .map(|e| e.to_ascii_lowercase())
        .as_deref()
    {
        Some("cbz") | Some("zip") => Ok(Format::Zip),
        Some("cbt") | Some("tar") => Ok(Format::Tar),
        Some(other) => Err(format!(
            "Standalone reading of .{other} files isn't supported yet — open it through a ComicHub server."
        )),
        None => Err("Unknown file type.".into()),
    }
}

fn list_entries(path: &Path, format: Format) -> Result<Vec<String>, String> {
    let mut names: Vec<String> = Vec::new();
    match format {
        Format::Zip => {
            let file = File::open(path).map_err(|e| e.to_string())?;
            let mut zip = zip::ZipArchive::new(file).map_err(|e| e.to_string())?;
            for i in 0..zip.len() {
                let entry = zip.by_index(i).map_err(|e| e.to_string())?;
                if entry.is_file() && is_image_entry(entry.name()) {
                    names.push(entry.name().to_string());
                }
            }
        }
        Format::Tar => {
            let file = File::open(path).map_err(|e| e.to_string())?;
            let mut archive = tar::Archive::new(file);
            for entry in archive.entries().map_err(|e| e.to_string())? {
                let entry = entry.map_err(|e| e.to_string())?;
                let name = entry
                    .path()
                    .map_err(|e| e.to_string())?
                    .to_string_lossy()
                    .replace('\\', "/");
                if is_image_entry(&name) {
                    names.push(name);
                }
            }
        }
    }
    names.sort_by(|a, b| natural_cmp(a, b));
    if names.is_empty() {
        return Err("No readable images found in this archive.".into());
    }
    Ok(names)
}

fn read_entry(path: &Path, format: Format, name: &str) -> Result<Vec<u8>, String> {
    match format {
        Format::Zip => {
            let file = File::open(path).map_err(|e| e.to_string())?;
            let mut zip = zip::ZipArchive::new(file).map_err(|e| e.to_string())?;
            let mut entry = zip.by_name(name).map_err(|e| e.to_string())?;
            let mut buf = Vec::with_capacity(entry.size() as usize);
            entry.read_to_end(&mut buf).map_err(|e| e.to_string())?;
            Ok(buf)
        }
        Format::Tar => {
            let file = File::open(path).map_err(|e| e.to_string())?;
            let mut archive = tar::Archive::new(file);
            for entry in archive.entries().map_err(|e| e.to_string())? {
                let mut entry = entry.map_err(|e| e.to_string())?;
                let ename = entry
                    .path()
                    .map_err(|e| e.to_string())?
                    .to_string_lossy()
                    .replace('\\', "/");
                if ename == name {
                    let mut buf = Vec::new();
                    entry.read_to_end(&mut buf).map_err(|e| e.to_string())?;
                    return Ok(buf);
                }
            }
            Err(format!("Entry not found: {name}"))
        }
    }
}

/// SHA-256 of the file, streamed — stable content hash for progress reconciliation.
fn content_hash(path: &Path) -> Result<String, String> {
    let mut file = File::open(path).map_err(|e| e.to_string())?;
    let mut hasher = Sha256::new();
    let mut buf = [0u8; 64 * 1024];
    loop {
        let n = file.read(&mut buf).map_err(|e| e.to_string())?;
        if n == 0 {
            break;
        }
        hasher.update(&buf[..n]);
    }
    Ok(format!("{:x}", hasher.finalize()))
}

fn dimensions(bytes: &[u8]) -> (u32, u32) {
    image::ImageReader::new(std::io::Cursor::new(bytes))
        .with_guessed_format()
        .ok()
        .and_then(|r| r.into_dimensions().ok())
        .unwrap_or((0, 0))
}

fn reading_dir(path: &Path, format: Format, entries: &[String]) -> String {
    // Default LTR; honor ComicInfo.xml's right-to-left Manga flag when present.
    let info = entries.iter().find(|n| {
        n.rsplit('/')
            .next()
            .map(|b| b.eq_ignore_ascii_case("ComicInfo.xml"))
            .unwrap_or(false)
    });
    // ComicInfo.xml is not in the image list; probe directly.
    let _ = info;
    if let Ok(bytes) = read_entry(path, format, "ComicInfo.xml") {
        if let Ok(text) = String::from_utf8(bytes) {
            if text.contains("YesAndRightToLeft") {
                return "rtl".into();
            }
        }
    }
    "ltr".into()
}

fn build_manifest(path: &Path, format: Format, entries: &[String]) -> Result<Manifest, String> {
    let book_id = content_hash(path)?;
    let mut pages = Vec::with_capacity(entries.len());
    for (idx, name) in entries.iter().enumerate() {
        let bytes = read_entry(path, format, name)?;
        let (w, h) = dimensions(&bytes);
        let double = w > 0 && h > 0 && (w as f32) / (h as f32) > 1.2;
        pages.push(PageMeta {
            idx,
            w,
            h,
            r#type: if idx == 0 {
                Some("FrontCover".into())
            } else {
                None
            },
            double,
        });
    }
    Ok(Manifest {
        book_id,
        page_count: entries.len(),
        reading_dir: reading_dir(path, format, entries),
        pages,
    })
}

fn progress_store_path(app: &tauri::AppHandle) -> Result<PathBuf, String> {
    let dir = app.path().app_data_dir().map_err(|e| e.to_string())?;
    std::fs::create_dir_all(&dir).map_err(|e| e.to_string())?;
    Ok(dir.join("reader_progress.json"))
}

fn read_progress_store(app: &tauri::AppHandle) -> serde_json::Value {
    progress_store_path(app)
        .ok()
        .and_then(|p| std::fs::read(p).ok())
        .and_then(|b| serde_json::from_slice(&b).ok())
        .unwrap_or_else(|| serde_json::json!({}))
}

#[tauri::command]
pub fn local_open(path: String) -> Result<Manifest, String> {
    let p = PathBuf::from(&path);
    if !p.exists() {
        return Err("File not found.".into());
    }
    let format = detect_format(&p)?;
    let entries = list_entries(&p, format)?;
    let manifest = build_manifest(&p, format, &entries)?;
    let mut guard = open_state().lock().map_err(|e| e.to_string())?;
    *guard = Some(OpenBook {
        path: p,
        format,
        entries,
        manifest: manifest.clone(),
    });
    Ok(manifest)
}

#[tauri::command]
pub fn local_manifest() -> Result<Manifest, String> {
    let guard = open_state().lock().map_err(|e| e.to_string())?;
    guard
        .as_ref()
        .map(|b| b.manifest.clone())
        .ok_or_else(|| "No book is open.".into())
}

#[tauri::command]
pub fn local_page(idx: usize) -> Result<Response, String> {
    let (path, format, name) = {
        let guard = open_state().lock().map_err(|e| e.to_string())?;
        let book = guard.as_ref().ok_or("No book is open.")?;
        let name = book
            .entries
            .get(idx)
            .ok_or_else(|| format!("Page {idx} out of range"))?
            .clone();
        (book.path.clone(), book.format, name)
    };
    let bytes = read_entry(&path, format, &name)?;
    Ok(Response::new(bytes))
}

#[tauri::command]
pub fn local_thumb(idx: usize) -> Result<Response, String> {
    let (path, format, name) = {
        let guard = open_state().lock().map_err(|e| e.to_string())?;
        let book = guard.as_ref().ok_or("No book is open.")?;
        let name = book
            .entries
            .get(idx)
            .ok_or_else(|| format!("Page {idx} out of range"))?
            .clone();
        (book.path.clone(), book.format, name)
    };
    let bytes = read_entry(&path, format, &name)?;
    let img = image::load_from_memory(&bytes).map_err(|e| e.to_string())?;
    let thumb = img.thumbnail(200, 300);
    let mut out = std::io::Cursor::new(Vec::new());
    thumb
        .write_to(&mut out, image::ImageFormat::Jpeg)
        .map_err(|e| e.to_string())?;
    Ok(Response::new(out.into_inner()))
}

#[tauri::command]
pub fn local_prefetch(_from: usize, _count: usize) -> Result<(), String> {
    // Reads are fast (per-entry); the JS PageCache drives the actual window. No-op for now.
    Ok(())
}

#[tauri::command]
pub fn local_save_progress(
    app: tauri::AppHandle,
    progress: serde_json::Value,
) -> Result<(), String> {
    let book_id = progress
        .get("bookId")
        .and_then(|v| v.as_str())
        .ok_or("progress.bookId missing")?
        .to_string();
    let mut store = read_progress_store(&app);
    if let Some(map) = store.as_object_mut() {
        map.insert(book_id, progress);
    }
    let path = progress_store_path(&app)?;
    std::fs::write(path, serde_json::to_vec_pretty(&store).map_err(|e| e.to_string())?)
        .map_err(|e| e.to_string())?;
    Ok(())
}

#[tauri::command]
pub fn local_restore_progress(
    app: tauri::AppHandle,
    book_id: String,
) -> Result<Option<serde_json::Value>, String> {
    let store = read_progress_store(&app);
    Ok(store.get(&book_id).cloned())
}

// ── Per-book reader overrides (layout/fit/direction/…) ─────────────────────────────────
// Stored locally keyed by book id, the standalone counterpart to the server's reader-prefs
// endpoint. The settings blob is opaque (the webview defines its shape).

fn prefs_store_path(app: &tauri::AppHandle) -> Result<PathBuf, String> {
    let dir = app.path().app_data_dir().map_err(|e| e.to_string())?;
    std::fs::create_dir_all(&dir).map_err(|e| e.to_string())?;
    Ok(dir.join("reader_prefs.json"))
}

fn read_prefs_store(app: &tauri::AppHandle) -> serde_json::Value {
    prefs_store_path(app)
        .ok()
        .and_then(|p| std::fs::read(p).ok())
        .and_then(|b| serde_json::from_slice(&b).ok())
        .unwrap_or_else(|| serde_json::json!({}))
}

#[tauri::command]
pub fn local_save_prefs(
    app: tauri::AppHandle,
    book_id: String,
    settings: serde_json::Value,
) -> Result<(), String> {
    let mut store = read_prefs_store(&app);
    if let Some(map) = store.as_object_mut() {
        map.insert(book_id, settings);
    }
    let path = prefs_store_path(&app)?;
    std::fs::write(path, serde_json::to_vec_pretty(&store).map_err(|e| e.to_string())?)
        .map_err(|e| e.to_string())?;
    Ok(())
}

#[tauri::command]
pub fn local_restore_prefs(
    app: tauri::AppHandle,
    book_id: String,
) -> Result<Option<serde_json::Value>, String> {
    let store = read_prefs_store(&app);
    Ok(store.get(&book_id).cloned())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Write;

    fn png_bytes(w: u32, h: u32) -> Vec<u8> {
        let img = image::RgbImage::from_pixel(w, h, image::Rgb([10, 20, 30]));
        let mut out = std::io::Cursor::new(Vec::new());
        image::DynamicImage::ImageRgb8(img)
            .write_to(&mut out, image::ImageFormat::Png)
            .unwrap();
        out.into_inner()
    }

    #[test]
    fn natural_sort_orders_numerically() {
        let mut v = vec![
            "page10.jpg".to_string(),
            "page2.jpg".into(),
            "page1.jpg".into(),
        ];
        v.sort_by(|a, b| natural_cmp(a, b));
        assert_eq!(v, vec!["page1.jpg", "page2.jpg", "page10.jpg"]);
    }

    #[test]
    fn open_cbz_builds_sorted_manifest() {
        let dir = std::env::temp_dir().join(format!("chtest_{}", std::process::id()));
        std::fs::create_dir_all(&dir).unwrap();
        let path = dir.join("test.cbz");
        {
            let file = File::create(&path).unwrap();
            let mut zip = zip::ZipWriter::new(file);
            let opts = zip::write::SimpleFileOptions::default();
            for name in ["page10.png", "page2.png", "page1.png", ".hidden.png"] {
                zip.start_file(name, opts).unwrap();
                let wide = name == "page10.png";
                zip.write_all(&png_bytes(if wide { 40 } else { 20 }, 30)).unwrap();
            }
            zip.finish().unwrap();
        }

        let format = detect_format(&path).unwrap();
        let entries = list_entries(&path, format).unwrap();
        assert_eq!(entries, vec!["page1.png", "page2.png", "page10.png"]);

        let manifest = build_manifest(&path, format, &entries).unwrap();
        assert_eq!(manifest.page_count, 3);
        assert_eq!(manifest.reading_dir, "ltr");
        assert_eq!((manifest.pages[0].w, manifest.pages[0].h), (20, 30));
        assert!(!manifest.pages[0].double);
        assert!(manifest.pages[2].double, "40x30 page should be flagged wide");
        assert_eq!(content_hash(&path).unwrap().len(), 64);

        let bytes = read_entry(&path, format, "page1.png").unwrap();
        assert!(!bytes.is_empty());

        std::fs::remove_dir_all(&dir).ok();
    }

    #[test]
    fn open_cbt_builds_sorted_manifest() {
        let dir = std::env::temp_dir().join(format!("chtest_cbt_{}", std::process::id()));
        std::fs::create_dir_all(&dir).unwrap();
        let path = dir.join("test.cbt");
        {
            let file = File::create(&path).unwrap();
            let mut tb = tar::Builder::new(file);
            for name in ["page10.png", "page2.png", "page1.png", ".hidden.png"] {
                let wide = name == "page10.png";
                let data = png_bytes(if wide { 40 } else { 20 }, 30);
                let mut header = tar::Header::new_gnu();
                header.set_size(data.len() as u64);
                header.set_mode(0o644);
                header.set_cksum();
                tb.append_data(&mut header, name, data.as_slice()).unwrap();
            }
            tb.finish().unwrap();
        }

        let format = detect_format(&path).unwrap();
        assert_eq!(format, Format::Tar);
        let entries = list_entries(&path, format).unwrap();
        assert_eq!(entries, vec!["page1.png", "page2.png", "page10.png"]);

        let manifest = build_manifest(&path, format, &entries).unwrap();
        assert_eq!(manifest.page_count, 3);
        assert_eq!(manifest.reading_dir, "ltr");
        assert_eq!((manifest.pages[0].w, manifest.pages[0].h), (20, 30));
        assert!(!manifest.pages[0].double);
        assert!(manifest.pages[2].double, "40x30 page should be flagged wide");
        assert_eq!(content_hash(&path).unwrap().len(), 64);

        let bytes = read_entry(&path, format, "page1.png").unwrap();
        assert!(!bytes.is_empty());

        std::fs::remove_dir_all(&dir).ok();
    }

    #[test]
    fn unsupported_format_is_reported() {
        // CBR and CB7 need native libraries; standalone reading errors with a helpful
        // message pointing the user at a server (file associations still advertise them).
        assert!(detect_format(Path::new("x.cbr")).is_err());
        assert!(detect_format(Path::new("x.cb7")).is_err());
    }
}
