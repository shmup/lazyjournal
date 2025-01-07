$OS = $([System.Environment]::OSVersion).VersionString
$ARCH = $env:PROCESSOR_ARCHITECTURE.ToLower()

Write-Host "Current system: " -NoNewline
Write-Host $OS -ForegroundColor Green -NoNewline
Write-Host " (architecture: " -NoNewline
Write-Host $ARCH -ForegroundColor Green -NoNewline
Write-Host ")"

$dstPath = "$env:LOCALAPPDATA\lazyjournal"
if (!(Test-Path $dstPath)) {
    New-Item -Path $dstPath -ItemType Directory | Out-Null
    Write-Host "Directory created: " -NoNewline
    Write-Host $dstPath -ForegroundColor Blue
}

$beforeEnvPath = [Environment]::GetEnvironmentVariable("Path", "User")
if (!($($beforeEnvPath).Split(";") -contains $dstPath)) {
    $afterEnvPath = $beforeEnvPath + ";$dstPath"
    [Environment]::SetEnvironmentVariable("Path", $afterEnvPath, "User")
    Write-Host "The path has been added to the Path environment variable for the current user."
}

$GITHUB_LATEST_VERSION = (Invoke-RestMethod "https://api.github.com/repos/Lifailon/lazyjournal/releases/latest").tag_name
if ($null -ne $GITHUB_LATEST_VERSION) {
    $urlDownload = "https://github.com/Lifailon/lazyjournal/releases/download/$GITHUB_LATEST_VERSION/lazyjournal-$GITHUB_LATEST_VERSION-windows-$ARCH.exe"
    Invoke-RestMethod -Uri $urlDownload -OutFile "$dstPath\lazyjournal.exe"
    Write-Host "✔  Installation completed " -NoNewline
    Write-Host "successfully" -ForegroundColor Green -NoNewline
    Write-Host " in " -NoNewline
    Write-Host "$dstPath\lazyjournal.exe" -ForegroundColor Blue -NoNewline
    Write-Host " (version: $GITHUB_LATEST_VERSION)"
    Write-Host "To launch the interface from anywhere" -NoNewline
    Write-Host " re-login " -ForegroundColor Green -NoNewline
    Write-Host "to the current session"
} else {
    Write-Host "Error. " -ForegroundColor Red -NoNewline
    Write-Host "Unable to get the latest version from GitHub repository, check your internet connection."
}
