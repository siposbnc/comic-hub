//! LAN server discovery: browse for the `_comichub._tcp` mDNS advertisements that
//! standalone servers publish (Phase 3 — Milestone D; counterpart of the server's
//! `internal/discovery` package). The webview calls `discover_servers` from the
//! connect screen and renders the result as the "Servers on your network" list.

use std::collections::HashMap;
use std::net::{IpAddr, SocketAddr, TcpStream};
use std::time::{Duration, Instant};

use mdns_sd::{ServiceDaemon, ServiceEvent};
use serde::Serialize;

/// Must match the server's `discovery.ServiceType` (+ the mDNS domain suffix).
const SERVICE_TYPE: &str = "_comichub._tcp.local.";

/// One advertised server, shaped for the connect screen.
#[derive(Serialize, Clone)]
pub struct DiscoveredServer {
    /// Human-readable server name (the DNS-SD instance name, `--server-name`).
    pub name: String,
    /// Base URL ready to feed the existing connect flow, e.g. `http://192.168.1.10:8080`.
    pub url: String,
    pub host: String,
    pub port: u16,
    /// Server version from the TXT record, when present.
    pub version: Option<String>,
    /// Whether the server requires login (TXT `auth`), so the UI can label the row.
    pub auth_required: bool,
}

/// Browses the LAN for ComicHub servers for `timeout_ms` (default 2500, capped at
/// 10 000) and returns every instance that resolved in that window, sorted by name.
#[tauri::command]
pub async fn discover_servers(timeout_ms: Option<u64>) -> Result<Vec<DiscoveredServer>, String> {
    let window = Duration::from_millis(timeout_ms.unwrap_or(2500).min(10_000));
    tauri::async_runtime::spawn_blocking(move || browse(window))
        .await
        .map_err(|e| format!("discovery task: {e}"))?
}

fn browse(window: Duration) -> Result<Vec<DiscoveredServer>, String> {
    let daemon = ServiceDaemon::new().map_err(|e| format!("mdns daemon: {e}"))?;
    let events = daemon
        .browse(SERVICE_TYPE)
        .map_err(|e| format!("mdns browse: {e}"))?;

    // Keyed by fullname: a re-resolve within the window just refreshes the entry.
    let mut found: HashMap<String, DiscoveredServer> = HashMap::new();
    let deadline = Instant::now() + window;
    while let Some(remaining) = deadline.checked_duration_since(Instant::now()) {
        let event = match events.recv_timeout(remaining) {
            Ok(ev) => ev,
            Err(_) => break, // window elapsed or daemon gone
        };
        if let ServiceEvent::ServiceResolved(svc) = event {
            let Some(addr) = pick_address(svc.addresses.iter().map(|a| a.to_ip_addr()), svc.port)
            else {
                continue;
            };
            let host = match addr {
                IpAddr::V4(v4) => v4.to_string(),
                IpAddr::V6(v6) => format!("[{v6}]"), // URL-safe literal
            };
            let name = svc
                .fullname
                .strip_suffix(&format!(".{SERVICE_TYPE}"))
                .unwrap_or(&svc.fullname)
                .to_string();
            found.insert(
                svc.fullname.clone(),
                DiscoveredServer {
                    name,
                    url: format!("http://{host}:{}", svc.port),
                    host,
                    port: svc.port,
                    version: svc
                        .txt_properties
                        .get_property_val_str("version")
                        .map(str::to_string),
                    auth_required: svc.txt_properties.get_property_val_str("auth") == Some("true"),
                },
            );
        }
    }
    let _ = daemon.shutdown();

    let mut servers: Vec<_> = found.into_values().collect();
    servers.sort_by(|a, b| a.name.cmp(&b.name));
    Ok(servers)
}

/// Picks the address to connect to. The advertisement carries *every* address of the
/// server's interfaces — including host-only virtual ones (Docker/WSL vSwitches) that
/// other machines can't reach — so probe candidates with a short TCP connect (the
/// server listens on 0.0.0.0, so any reachable address accepts) and take the first
/// that answers, IPv4 first. Falls back to the first candidate when none respond, so
/// the row still renders and a click surfaces the ordinary connect error.
fn pick_address(addrs: impl Iterator<Item = IpAddr>, port: u16) -> Option<IpAddr> {
    let mut candidates: Vec<IpAddr> = addrs.collect();
    candidates.sort_by_key(|a| (a.is_ipv6(), *a)); // deterministic: v4s first
    candidates
        .iter()
        .copied()
        .find(|&addr| {
            TcpStream::connect_timeout(&SocketAddr::new(addr, port), Duration::from_millis(300))
                .is_ok()
        })
        .or_else(|| candidates.first().copied())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::net::Ipv4Addr;

    #[test]
    fn pick_address_prefers_reachable() {
        let listener = std::net::TcpListener::bind("127.0.0.1:0").unwrap();
        let port = listener.local_addr().unwrap().port();
        let unreachable = IpAddr::V4(Ipv4Addr::new(192, 0, 2, 1)); // TEST-NET-1, never routable
        let local = IpAddr::V4(Ipv4Addr::LOCALHOST);
        assert_eq!(pick_address([unreachable, local].into_iter(), port), Some(local));
        // Nothing answers: fall back to the first candidate rather than dropping the row.
        assert_eq!(pick_address([unreachable].into_iter(), port), Some(unreachable));
        assert_eq!(pick_address([].into_iter(), port), None);
    }

    /// Manual e2e against a live server (multicast doesn't work in CI):
    /// `comichub-server --mode server --bind 0.0.0.0:8099`, then
    /// `cargo test discovery -- --ignored --nocapture`.
    #[test]
    #[ignore = "needs a --mode server instance advertising on this LAN"]
    fn browse_finds_live_server() {
        let servers = browse(Duration::from_millis(4000)).expect("browse");
        println!("discovered: {}", servers.len());
        for s in &servers {
            println!("  {} -> {} (version {:?}, auth {})", s.name, s.url, s.version, s.auth_required);
        }
        assert!(!servers.is_empty(), "no ComicHub servers discovered");
    }
}
