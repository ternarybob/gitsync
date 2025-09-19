# -----------------------------------------------------------------------
# Build Script for GitSync
# -----------------------------------------------------------------------

param (
    [string]$Environment = "dev",
    [string]$Version = "",
    [switch]$Clean,
    [switch]$Test,
    [switch]$Verbose,
    [switch]$Release,
    [string]$OS = "",
    [string]$Arch = ""
)

<#
.SYNOPSIS
    Build script for GitSync

.DESCRIPTION
    This script builds the GitSync binary for local development and deployment.
    Outputs the executable to the project's bin directory.

.PARAMETER Environment
    Target environment for build (dev, staging, prod)

.PARAMETER Version
    Version to embed in the binary (defaults to git commit hash)

.PARAMETER Clean
    Clean build artifacts before building

.PARAMETER Test
    Run tests before building

.PARAMETER Verbose
    Enable verbose output

.PARAMETER Release
    Build optimized release binary

.PARAMETER OS
    Target operating system (windows, linux, darwin)

.PARAMETER Arch
    Target architecture (amd64, arm64)

.EXAMPLE
    .\build.ps1
    Build gitsync for development

.EXAMPLE
    .\build.ps1 -Environment prod -Release -Test
    Build optimized production binary with tests

.EXAMPLE
    .\build.ps1 -OS linux -Arch amd64 -Release
    Cross-compile for Linux amd64
#>

Push-Location (Split-Path (Split-Path $MyInvocation.MyCommand.Path))

try {
    Write-Host "GitSync Build Script" -ForegroundColor Cyan
    Write-Host "Environment: $Environment" -ForegroundColor Yellow
    Write-Host "Current Location: $(Get-Location)"

    # Check for running GitSync processes and kill them
    Write-Host "`nChecking for running GitSync processes..." -ForegroundColor Yellow
    $runningProcesses = Get-Process -Name "gitsync*" -ErrorAction SilentlyContinue
    if ($runningProcesses) {
        Write-Host "Found $($runningProcesses.Count) running GitSync process(es). Terminating..." -ForegroundColor Red
        foreach ($process in $runningProcesses) {
            Write-Host "  Killing process: $($process.Name) (PID: $($process.Id))" -ForegroundColor Red
            try {
                $process.Kill()
                $process.WaitForExit(5000)  # Wait up to 5 seconds for graceful exit
            }
            catch {
                Write-Warning "Failed to kill process $($process.Id): $_"
            }
        }
        Start-Sleep -Seconds 1  # Brief pause to ensure processes are fully terminated
        Write-Host "All GitSync processes terminated" -ForegroundColor Green
    }
    else {
        Write-Host "No running GitSync processes found" -ForegroundColor Green
    }

    # Validate environment
    $validEnvironments = @("dev", "staging", "prod")
    if ($Environment -notin $validEnvironments) {
        Write-Error "Invalid environment: $Environment. Valid options: $($validEnvironments -join ', ')"
        exit 1
    }

    # Get version information and auto-increment
    $versionFilePath = ".version"
    $currentVersion = "0.0.1"  # Default starting version
    $buildTime = Get-Date -Format "yyyy-MM-dd HH:mm:ss"

    if (-not $Version) {
        # Read and auto-increment version from .version file
        if (Test-Path $versionFilePath) {
            $versionLines = Get-Content $versionFilePath
            foreach ($line in $versionLines) {
                if ($line -match '^version:\s*(.+)$') {
                    $currentVersion = $matches[1].Trim()
                    break
                }
            }

            # Parse and increment version
            if ($currentVersion -match '^(\d+)\.(\d+)\.(\d+)$') {
                $major = [int]$matches[1]
                $minor = [int]$matches[2]
                $patch = [int]$matches[3] + 1
                $Version = "$major.$minor.$patch"
            }
            else {
                $Version = $currentVersion
            }
        }
        else {
            # Create initial version file if it doesn't exist
            $Version = $currentVersion
        }

        # Update .version file with new version and build timestamp
        $versionContent = @"
version: $Version
build: $buildTime
"@
        Set-Content -Path $versionFilePath -Value $versionContent
        Write-Host "Auto-incremented version to: $Version" -ForegroundColor Cyan
    }
    else {
        # Manual version specified, update .version file
        $versionContent = @"
version: $Version
build: $buildTime
"@
        Set-Content -Path $versionFilePath -Value $versionContent
        Write-Host "Using specified version: $Version" -ForegroundColor Green
    }

    Write-Host "Build Time: $buildTime"

    # Clean if requested
    if ($Clean) {
        Write-Host "`nCleaning build artifacts..." -ForegroundColor Yellow
        if (Test-Path "bin") {
            Remove-Item -Path "bin" -Recurse -Force
        }
        go clean -cache
        Write-Host "Clean complete" -ForegroundColor Green
    }

    # Run tests if requested
    if ($Test) {
        Write-Host "`nRunning tests..." -ForegroundColor Yellow
        $testArgs = @("test", "./...")
        if ($Verbose) {
            $testArgs += "-v"
        }
        $testResult = & go @testArgs
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Tests failed"
            exit 1
        }
        Write-Host "Tests passed" -ForegroundColor Green
    }

    # Create bin directory if it doesn't exist
    $binDir = Join-Path -Path (Get-Location) -ChildPath "bin"
    if (-not (Test-Path $binDir)) {
        New-Item -ItemType Directory -Path $binDir | Out-Null
    }

    # Determine output binary name based on target OS
    $isWindowsOS = [System.Environment]::OSVersion.Platform -eq "Win32NT"
    if ($OS -eq "windows" -or (-not $OS -and $env:GOOS -eq "windows") -or (-not $OS -and -not $env:GOOS -and $isWindowsOS)) {
        $outputName = "gitsync.exe"
    }
    elseif ($OS -eq "darwin" -or (-not $OS -and $env:GOOS -eq "darwin")) {
        $outputName = "gitsync-darwin"
    }
    else {
        $outputName = "gitsync-linux"
    }
    $outputPath = Join-Path -Path $binDir -ChildPath $outputName

    # Set up build environment
    $env:CGO_ENABLED = "0"
    if ($OS) {
        $env:GOOS = $OS
    }
    if ($Arch) {
        $env:GOARCH = $Arch
    }

    # Build arguments
    $buildArgs = @(
        "build",
        "-o", $outputPath
    )

    # Add ldflags - same format as bash script
    $versionFlag = "-X github.com/ternarybob/gitsync/internal/common.Version=$Version"
    $buildFlag = "-X 'github.com/ternarybob/gitsync/internal/common.Build=$buildTime'"
    $ldflags = "$versionFlag $buildFlag"

    if ($Release) {
        Write-Host "`nBuilding release binary..." -ForegroundColor Yellow
        $ldflags += " -s -w"
        $buildArgs += "-trimpath"
    }
    else {
        Write-Host "`nBuilding development binary..." -ForegroundColor Yellow
    }

    $buildArgs += "-ldflags", $ldflags

    if ($Verbose) {
        $buildArgs += "-v"
    }

    # Add source path
    $buildArgs += "./cmd/gitsync"

    # Show build command if verbose
    if ($Verbose) {
        Write-Host "Build command: go $($buildArgs -join ' ')" -ForegroundColor DarkGray
    }

    # Execute build
    $buildResult = & go @buildArgs 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Build failed: $buildResult"
        exit 1
    }

    # Display build results
    Write-Host "`nBuild successful!" -ForegroundColor Green
    Write-Host "Output: $outputPath" -ForegroundColor Yellow

    # Show binary info
    $fileInfo = Get-Item $outputPath
    Write-Host "Size: $([math]::Round($fileInfo.Length / 1MB, 2)) MB"

    if ($OS -and $Arch) {
        Write-Host "Target: $OS/$Arch"
    }

    # Version information is embedded via build flags, no separate file needed

    Write-Host "`nBuild complete!" -ForegroundColor Green
}
catch {
    Write-Error "Build failed: $_"
    exit 1
}
finally {
    Pop-Location
}