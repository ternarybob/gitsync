# -----------------------------------------------------------------------
# Test Script for GitSync
# -----------------------------------------------------------------------

param (
    [string]$Package = "./...",
    [switch]$Verbose,
    [switch]$Coverage,
    [switch]$Race,
    [switch]$Short,
    [switch]$Bench,
    [string]$Run = ""
)

<#
.SYNOPSIS
    Test script for GitSync

.DESCRIPTION
    This script runs tests for the GitSync project with various options.

.PARAMETER Package
    Specific package to test (default: all packages)

.PARAMETER Verbose
    Enable verbose test output

.PARAMETER Coverage
    Generate coverage report

.PARAMETER Race
    Enable race detector

.PARAMETER Short
    Run short tests only

.PARAMETER Bench
    Run benchmarks

.PARAMETER Run
    Run only tests matching pattern

.EXAMPLE
    .\test.ps1
    Run all tests

.EXAMPLE
    .\test.ps1 -Coverage -Verbose
    Run tests with coverage and verbose output

.EXAMPLE
    .\test.ps1 -Run "TestSync" -Package "./internal/sync"
    Run specific tests in a package
#>

Push-Location (Split-Path (Split-Path $MyInvocation.MyCommand.Path))

try {
    Write-Host "GitSync Test Runner" -ForegroundColor Cyan
    Write-Host "===================" -ForegroundColor Cyan

    # Build test command
    $testArgs = @("test")

    if ($Verbose) {
        $testArgs += "-v"
        Write-Host "Verbose: Enabled" -ForegroundColor Yellow
    }

    if ($Coverage) {
        $testArgs += "-cover"
        $testArgs += "-coverprofile=coverage.out"
        Write-Host "Coverage: Enabled" -ForegroundColor Yellow
    }

    if ($Race) {
        $testArgs += "-race"
        Write-Host "Race Detector: Enabled" -ForegroundColor Yellow
    }

    if ($Short) {
        $testArgs += "-short"
        Write-Host "Short Tests: Enabled" -ForegroundColor Yellow
    }

    if ($Bench) {
        $testArgs += "-bench=."
        Write-Host "Benchmarks: Enabled" -ForegroundColor Yellow
    }

    if ($Run) {
        $testArgs += "-run"
        $testArgs += $Run
        Write-Host "Pattern: $Run" -ForegroundColor Yellow
    }

    # Add package
    $testArgs += $Package
    Write-Host "Package: $Package" -ForegroundColor Yellow

    Write-Host "`nRunning tests..." -ForegroundColor Green
    Write-Host "Command: go $($testArgs -join ' ')" -ForegroundColor DarkGray
    Write-Host ""

    # Run tests
    & go @testArgs
    $exitCode = $LASTEXITCODE

    if ($exitCode -eq 0) {
        Write-Host "`n✓ All tests passed!" -ForegroundColor Green

        # Show coverage report if generated
        if ($Coverage) {
            Write-Host "`nGenerating coverage report..." -ForegroundColor Yellow
            go tool cover -func=coverage.out

            Write-Host "`nTo view HTML coverage report, run:" -ForegroundColor Cyan
            Write-Host "  go tool cover -html=coverage.out" -ForegroundColor White
        }
    }
    else {
        Write-Host "`n✗ Tests failed with exit code: $exitCode" -ForegroundColor Red
        exit $exitCode
    }
}
catch {
    Write-Error "Test execution failed: $_"
    exit 1
}
finally {
    Pop-Location
}