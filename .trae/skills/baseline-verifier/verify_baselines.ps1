$baselinesDir = "baselines"
$retDir = "ret"

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
        $response = Invoke-RestMethod -Uri 'http://localhost:8080/api/backtest' -Method POST -ContentType 'application/json' -Body $body -TimeoutSec 300
        
        Start-Sleep -Seconds 1
        
        $pattern = "${symbol}_${strategy}_${startDate}_${endDate}_*_*.json"
        $latestFile = Get-ChildItem -Path "$retDir\$pattern" -ErrorAction SilentlyContinue | Sort-Object LastWriteTime -Descending | Select-Object -First 1
        
        if (-not $latestFile) {
            Write-Host "[FAIL] Result file not found" -ForegroundColor Red
            $failCount++
            continue
        }
        
        Write-Host "Result file: $($latestFile.Name)" -ForegroundColor Gray
        
        $resultJson = Get-Content $latestFile.FullName -Raw -Encoding UTF8
        $result = $resultJson | ConvertFrom-Json
        
        Write-Host ""
        Write-Host "Verifying signals..." -ForegroundColor Yellow
        
        $baselineSignalCount = $baseline.signals.Count
        $resultSignalCount = $result.signals.Count
        
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
                $rSig = $result.signals[$i]
                
                if ($bSig.signal_date -ne $rSig.signal_date -or
                    $bSig.price -ne $rSig.price -or
                    $bSig.direction -ne $rSig.direction -or
                    $bSig.symbol -ne $rSig.symbol) {
                    $allMatch = $false
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
        
        $statsPass = $false
        $statsMatch = $true
        $diffStats = @()
        
        if ($baseline.statistics.total_return -ne $result.statistics.total_return) {
            $statsMatch = $false
            $diffStats += "total_return"
        }
        if ($baseline.statistics.max_drawdown -ne $result.statistics.max_drawdown) {
            $statsMatch = $false
            $diffStats += "max_drawdown"
        }
        if ($baseline.statistics.sharpe_ratio -ne $result.statistics.sharpe_ratio) {
            $statsMatch = $false
            $diffStats += "sharpe_ratio"
        }
        if ($baseline.statistics.win_rate -ne $result.statistics.win_rate) {
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
