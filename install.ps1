$ErrorActionPreference = "Stop"

$Repo = if ($env:MAILCLI_REPO) { $env:MAILCLI_REPO } else { "nonozone/MailCli" }
$Version = if ($env:MAILCLI_VERSION) { $env:MAILCLI_VERSION } else { "latest" }
$InstallDir = if ($env:MAILCLI_INSTALL_DIR) { $env:MAILCLI_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "Programs\MailCLI" }
$AutoConfigure = $env:MAILCLI_AGENT_AUTO_CONFIGURE -in @("1", "true", "TRUE", "yes", "YES")
$Agents = if ($env:MAILCLI_AGENTS) { $env:MAILCLI_AGENTS.Split(",") | Where-Object { $_ } } else { @() }
$BaseUrlOverride = if ($env:MAILCLI_BASE_URL) { $env:MAILCLI_BASE_URL } else { "" }

function Get-MailCLIArchitecture {
  switch ([System.Runtime.InteropServices.RuntimeInformation]::ProcessArchitecture.ToString().ToLowerInvariant()) {
    "x64" { "amd64"; break }
    "arm64" { "arm64"; break }
    default { throw "unsupported architecture: $([System.Runtime.InteropServices.RuntimeInformation]::ProcessArchitecture)" }
  }
}

$Arch = Get-MailCLIArchitecture
$Asset = "mailcli_windows_$Arch.zip"
if ($BaseUrlOverride) {
  $BaseUrl = $BaseUrlOverride.TrimEnd("/")
} elseif ($Version -eq "latest") {
  $BaseUrl = "https://github.com/$Repo/releases/latest/download"
} else {
  $BaseUrl = "https://github.com/$Repo/releases/download/$Version"
}

$TempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("mailcli-install-" + [System.Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $TempDir | Out-Null

try {
  $Archive = Join-Path $TempDir $Asset
  $ChecksumFile = Join-Path $TempDir "checksums.txt"

  Write-Host "Installing MailCLI from $Repo ($Version) for windows/$Arch"
  Invoke-WebRequest -Uri "$BaseUrl/$Asset" -OutFile $Archive

  try {
    Invoke-WebRequest -Uri "$BaseUrl/checksums.txt" -OutFile $ChecksumFile
    $Expected = Get-Content $ChecksumFile |
      ForEach-Object { $_.Trim() } |
      Where-Object { $_ -match "\s$([regex]::Escape($Asset))$" } |
      ForEach-Object { ($_ -split "\s+")[0].ToLowerInvariant() } |
      Select-Object -First 1
    if (-not $Expected) {
      throw "checksum entry not found for $Asset"
    }
    $Actual = (Get-FileHash -Algorithm SHA256 -Path $Archive).Hash.ToLowerInvariant()
    if ($Actual -ne $Expected) {
      throw "checksum mismatch for $Asset"
    }
  } catch {
    throw "checksum verification failed: $($_.Exception.Message)"
  }

  Expand-Archive -Path $Archive -DestinationPath $TempDir -Force
  New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
  $Target = Join-Path $InstallDir "mailcli.exe"
  Copy-Item -Path (Join-Path $TempDir "mailcli.exe") -Destination $Target -Force
  Write-Host "Installed: $Target"

  $UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
  if (-not (($UserPath -split ";") -contains $InstallDir)) {
    Write-Warning "$InstallDir is not in your user PATH. Add it to PATH or call $Target directly."
  }

  if ($AutoConfigure) {
    $Args = @("agent", "configure", "--mailcli-bin", $Target)
    foreach ($Agent in $Agents) {
      $Args += @("--agent", $Agent)
    }
    & $Target @Args
  } else {
    & $Target agent doctor --mailcli-bin $Target
    Write-Host "To register MailCLI with detected agents, rerun with MAILCLI_AGENT_AUTO_CONFIGURE=1."
  }
} finally {
  Remove-Item -Path $TempDir -Recurse -Force -ErrorAction SilentlyContinue
}
