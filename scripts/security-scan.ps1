# Local Security Scanning Script for BigTown Project

Write-Host "=== 1. Checking Go Backend Vulnerabilities (govulncheck) ===" -ForegroundColor Cyan
Push-Location backend
try {
    go run golang.org/x/vuln/cmd/govulncheck@latest ./...
} finally {
    Pop-Location
}

Write-Host "`n=== 2. Checking Frontend Dependencies (npm audit) ===" -ForegroundColor Cyan
Push-Location frontend
try {
    npm audit --audit-level=high
} finally {
    Pop-Location
}

Write-Host "`n=== 3. Checking E2E Testing Dependencies (npm audit) ===" -ForegroundColor Cyan
Push-Location testing/e2e
try {
    npm audit --audit-level=high
} finally {
    Pop-Location
}

Write-Host "`n=== Security Audit Complete! ===" -ForegroundColor Green
