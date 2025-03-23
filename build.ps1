# Get version from GitVersion
$version = (gitversion /showvariable FullSemVer)
Write-Host "Building version: $version"

# Build flags - fixing quote issues
$ldflags = "-X main.Version=$version"

# Build for Windows x64
Write-Host "Building for Windows x64..."
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -ldflags "$ldflags" -o "neptunus-windows-amd64.exe" ./cmd/neptunus

# Build for Linux x64
Write-Host "Building for Linux x64..."
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -ldflags "$ldflags" -o "neptunus-linux-amd64" ./cmd/neptunus

Write-Host "Build complete!"