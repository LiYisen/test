$baselinesDir = "baselines"
$apiBase = "http://localhost:8080/api"

function Normalize-Direction {
    param([string]$dir)
    switch ($dir) {
        "0" { return "Buy" }
        "1" { return "Sell" }
        "2" { return "Close" }
        "3" { return "CloseShort" }
        "4" { return "CloseLong" }
        default { return $dir }
    }
}

function Stats-Equal {
    param([string]$b, [string]$r)
    if ($b -eq $r) { return $true }
    try {
        $bVal = [double]$b
        $rVal = [double]$r
        $diff = [Math]::Abs($bVal - $rVal)
        return $diff -lt 0.0001
    } catch {
        return $false
    }
}

if (-not (Test-Path $baselinesDir)) {
    Write-Host "Error: baselines directory not found" -ForegroundColor Red
    exit 1
}

$baselineFiles = Get-ChildItem -Path $baselinesDir -Filter "*.json"

if ($baselineFiles.Count -eq 0) {
    Write-Host "Warning: No baseline files found" -ForegroundColor Yellow
    exit 0
}

Write-Host "Starting verification of $($baselineFiles.Count) baseline files..." -ForegroundColor Cyan
Write-Host "Data source: SQLite database (via API)" -ForegroundColor DarkGray
Write-Host ""

$passCount = 0
$failCount = 0

foreach ($file in $baselineFiles) {
    Write-Host "========== Verifying: $($file.Name) ==========" -ForegroundColor Cyan

    try {
        $jsonContent = Get-Content $file.FullName -Raw -Encoding UTF8
        $baseline = $jsonContent | ConvertFrom-Json

        $symbol = $baseline.request.symbol
        $startDate = $baseline.request.start_date
        $endDate = $baseline.request.end_date
        $strategy = $baseline.request.strategy
        $params = $baseline.request.params

        Write-Host "Parameters: Symbol=$symbol, Strategy=$strategy, Start=$startDate, End=$endDate"

        $body = @{
            symbol = $symbol
            start_date = $startDate
            end_date = $endDate
            strategy = $strategy
            params = $params
        } | ConvertTo-Json -Depth 3

        Write-Host "Running backtest..." -ForegroundColor Yellow
        $response = Invoke-RestMethod -Uri "$apiBase/backtest" -Method POST -ContentType 'application/json' -Body $body -TimeoutSec 300

        $resultId = $response.id
        if (-not $resultId) {
            Write-Host "[FAIL] No result ID in response" -ForegroundColor Red
            $failCount++
            continue
        }

        Write-Host "Result ID: $resultId" -ForegroundColor Gray

        Start-Sleep -Seconds 1

        Write-Host ""
        Write-Host "Verifying signals..." -ForegroundColor Yellow

        $resultSignalsData = Invoke-RestMethod -Uri "$apiBase/results/$resultId/data?type=signals" -Method GET
        $resultSignals = $resultSignalsData.signals

        $baselineSignalCount = $baseline.signals.Count
        $resultSignalCount = $resultSignals.Count

        $signalsPass = $false
        if ($baselineSignalCount -ne $resultSignalCount) {
            Write-Host "[FAIL] Signal count mismatch" -ForegroundColor Red
            Write-Host "  Baseline: $baselineSignalCount signals" -ForegroundColor Gray
            Write-Host "  Result:   $resultSignalCount signals" -ForegroundColor Gray
            Write-Host "  >>> Strategy-related changes detected <<<" -ForegroundColor Red
        } else {
            $allMatch = $true
            for ($i = 0; $i -lt $baselineSignalCount; $i++) {
                $bSig = $baseline.signals[$i]
                $rSig = $resultSignals[$i]

                $bDate = $bSig.signal_date
                $rDate = $rSig.date
                $bPrice = $bSig.price
                $rPrice = $rSig.price
                $bDir = Normalize-Direction $bSig.direction
                $rDir = $rSig.direction
                $bSym = $bSig.symbol
                $rSym = $rSig.symbol

                if ($bDate -ne $rDate -or $bPrice -ne $rPrice -or $bDir -ne $rDir -or $bSym -ne $rSym) {
                    $allMatch = $false
                    Write-Host "  Mismatch at signal $i:" -ForegroundColor DarkGray
                    Write-Host "    Baseline: date=$bDate price=$bPrice dir=$bDir sym=$bSym" -ForegroundColor DarkGray
                    Write-Host "    Result:   date=$rDate price=$rPrice dir=$rDir sym=$rSym" -ForegroundColor DarkGray
                    break
                }
            }

            if ($allMatch) {
                Write-Host "[PASS] Signals verification passed ($baselineSignalCount signals)" -ForegroundColor Green
                $signalsPass = $true
            } else {
                Write-Host "[FAIL] Signal content mismatch" -ForegroundColor Red
                Write-Host "  >>> Strategy-related changes detected <<<" -ForegroundColor Red
            }
        }

        Write-Host ""
        Write-Host "Verifying statistics..." -ForegroundColor Yellow

        $resultStatsData = Invoke-RestMethod -Uri "$apiBase/results/$resultId/data?type=stats" -Method GET
        $resultStats = $resultStatsData.statistics

        $statsPass = $false
        $statsMatch = $true
        $diffStats = @()

        if (-not (Stats-Equal $baseline.statistics.TotalReturn $resultStats.total_return)) {
            $statsMatch = $false
            $diffStats += "total_return"
        }
        if (-not (Stats-Equal $baseline.statistics.MaxDrawdown $resultStats.max_drawdown)) {
            $statsMatch = $false
            $diffStats += "max_drawdown"
        }
        if (-not (Stats-Equal $baseline.statistics.SharpeRatio $resultStats.sharpe_ratio)) {
            $statsMatch = $false
            $diffStats += "sharpe_ratio"
        }
        if (-not (Stats-Equal $baseline.statistics.WinRate $resultStats.win_rate)) {
            $statsMatch = $false
            $diffStats += "win_rate"
        }

        if ($statsMatch) {
            Write-Host "[PASS] Statistics verification passed" -ForegroundColor Green
            $statsPass = $true
        } else {
            Write-Host "[FAIL] Statistics verification failed" -ForegroundColor Red
            Write-Host "  Different fields: $($diffStats -join ', ')" -ForegroundColor Gray
            Write-Host "  >>> Statistics-related changes detected <<<" -ForegroundColor Red
        }

        if ($signalsPass -and $statsPass) {
            $passCount++
        } else {
            $failCount++
        }

        Write-Host "Cleaning up result: $resultId" -ForegroundColor Yellow
        try {
            Invoke-RestMethod -Uri "$apiBase/results/$resultId" -Method DELETE | Out-Null
        } catch {}

    } catch {
        Write-Host "[FAIL] Verification error: $_" -ForegroundColor Red
        Write-Host "Details: $($_.Exception.Message)" -ForegroundColor Gray
        $failCount++
    }

    Write-Host ""
}

Write-Host "========== Summary ==========" -ForegroundColor Cyan
Write-Host "Total: $($baselineFiles.Count) baseline files"
Write-Host "Passed: $passCount" -ForegroundColor Green
Write-Host "Failed: $failCount" -ForegroundColor $(if ($failCount -gt 0) { "Red" } else { "Green" })

if ($failCount -gt 0) {
    Write-Host ""
    Write-Host "[WARNING] Regression detected! Please check the failed verifications above." -ForegroundColor Red
    Write-Host "If this is intentional, please update the baseline files." -ForegroundColor Yellow
    exit 1
} else {
    Write-Host ""
    Write-Host "[SUCCESS] All verifications passed! No regression detected." -ForegroundColor Green
}
