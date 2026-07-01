# Installs ComicHub as a Windows Service (docs/10-deployment.md). Run elevated:
#
#   pwsh -ExecutionPolicy Bypass -File deploy\install-windows-service.ps1 `
#     -Binary 'C:\ComicHub\comichub-server.exe' -DataDir 'C:\ComicHub\data'
#
# The service runs in server mode with auth on. The admin bootstrap credentials are
# written to the service-scoped Environment registry value (not machine-wide, and
# never on the command line); the server reads them once on first boot to create
# the account.

[CmdletBinding()]
param(
  [Parameter(Mandatory)] [string] $Binary,
  [Parameter(Mandatory)] [string] $DataDir,
  [string] $ServiceName = 'ComicHub',
  [string] $Bind = '0.0.0.0:8080',
  [string] $AdminUsername = 'admin',
  # Prompted when omitted, so it never lands in your shell history.
  [string] $AdminPassword
)

$ErrorActionPreference = 'Stop'

if (-not (Test-Path $Binary)) { throw "server binary not found: $Binary" }
if (-not $AdminPassword) {
  $sec = Read-Host -AsSecureString "Admin password for first boot"
  $AdminPassword = [System.Net.NetworkCredential]::new('', $sec).Password
}
New-Item -ItemType Directory -Force $DataDir | Out-Null

$cmd = "`"$Binary`" --mode server --bind $Bind --data-dir `"$DataDir`" --auth"
if (Get-Service $ServiceName -ErrorAction SilentlyContinue) {
  throw "service '$ServiceName' already exists (sc.exe delete $ServiceName to remove)"
}
New-Service -Name $ServiceName -BinaryPathName $cmd `
  -DisplayName 'ComicHub media server' -StartupType Automatic `
  -Description 'ComicHub comic library server (comichub-server --mode server)' | Out-Null

# Service-scoped environment (REG_MULTI_SZ "Environment" under the service key):
# the admin bootstrap is read once on first boot to create the account.
$key = "HKLM:\SYSTEM\CurrentControlSet\Services\$ServiceName"
Set-ItemProperty -Path $key -Name Environment -Type MultiString -Value @(
  "COMICHUB_ADMIN_USERNAME=$AdminUsername",
  "COMICHUB_ADMIN_PASSWORD=$AdminPassword"
)

# Restart on crash: 3 restarts, 5s apart, counter resets daily.
sc.exe failure $ServiceName reset= 86400 actions= restart/5000/restart/5000/restart/5000 | Out-Null

Start-Service $ServiceName
Write-Host "ComicHub service '$ServiceName' running on $Bind (data: $DataDir)."
Write-Host "Sign in as '$AdminUsername', then change the password in Settings."
