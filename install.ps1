# termidi installer for Windows:
#   irm https://raw.githubusercontent.com/EmreErdogan/termidi/main/install.ps1 | iex
$ErrorActionPreference = 'Stop'

$App = 'termidi'
$Repo = 'EmreErdogan/termidi'
$InstallDir = if ($env:INSTALL_DIR) { $env:INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "Programs\$App" }

$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
  'AMD64' { 'amd64' }
  'ARM64' { 'arm64' }
  default { throw "unsupported architecture: $env:PROCESSOR_ARCHITECTURE" }
}

$url = "https://github.com/$Repo/releases/latest/download/$App-windows-$arch.exe"
$exe = Join-Path $InstallDir "$App.exe"
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
Write-Host "downloading $url"
Invoke-WebRequest -Uri $url -OutFile "$exe.tmp" -UseBasicParsing
Move-Item -Force "$exe.tmp" $exe

# Make sure InstallDir is on the user PATH for future shells.
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if (($userPath -split ';') -notcontains $InstallDir) {
  [Environment]::SetEnvironmentVariable('Path', ($userPath.TrimEnd(';') + ";$InstallDir").TrimStart(';'), 'User')
  Write-Host "added $InstallDir to your PATH (open a new terminal to use it)"
}
$env:Path = "$env:Path;$InstallDir"

Write-Host "installed $exe ($(& $exe version))"
