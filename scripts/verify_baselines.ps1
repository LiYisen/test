$baselinesDir = "baselines"

if (-not (Test-Path $baselinesDir)) {
    Write-Host "错误: baselines目录不存在" -ForegroundColor Red
    exit 1
}

$baselineFiles = Get-ChildItem -Path $baselinesDir -Filter "*.json"

if ($baselineFiles.Count -eq 0) {
    Write-Host "警告: baselines目录中没有基线文件" -ForegroundColor Yellow
    exit 0
}

Write-Host "开始验证 $($baselineFiles.Count) 个基线文件..." -ForegroundColor Cyan
Write-Host ""

$passCount = 0
$failCount = 0

foreach ($file in $baselineFiles) {
    Write-Host "========== 验证基线: $($file.Name) ==========" -ForegroundColor Cyan
    
    try {
        $baseline = Get-Content $file.FullName -Raw | ConvertFrom-Json
        
        $symbol = $baseline.request.symbol
        $startDate = $baseline.request.start_date
        $endDate = $baseline.request.end_date
        $strategy = $baseline.request.strategy
        $params = $baseline.request.params
        
        Write-Host "参数: 品种=$symbol, 策略=$strategy, 开始=$startDate, 结束=$endDate"
        
        $body = @{
            symbol = $symbol
            start_date = $startDate
            end_date = $endDate
            strategy = $strategy
            params = $params
        } | ConvertTo-Json -Depth 3
        
        $response = Invoke-RestMethod -Uri 'http://localhost:8080/api/backtest' -Method POST -ContentType 'application/json' -Body $body -TimeoutSec 300
        
        Start-Sleep -Seconds 1
        
        $pattern = "${symbol}_${strategy}_${startDate}_${endDate}_*_*.json"
        $latestFile = Get-ChildItem -Path "ret\$pattern" | Sort-Object LastWriteTime -Descending | Select-Object -First 1
        
        if (-not $latestFile) {
            Write-Host "✗ 未找到结果文件" -ForegroundColor Red
            $failCount++
            continue
        }
        
        $result = Get-Content $latestFile.FullName -Raw | ConvertFrom-Json
        
        $baselineSignals = $baseline.signals | ConvertTo-Json -Depth 10 -Compress
        $resultSignals = $result.signals | ConvertTo-Json -Depth 10 -Compress
        
        $signalsPass = $false
        if ($baselineSignals -eq $resultSignals) {
            Write-Host "✓ Signals验证通过" -ForegroundColor Green
            $signalsPass = $true
        } else {
            Write-Host "✗ Signals验证失败：策略执行出现问题或策略被修改" -ForegroundColor Red
            Write-Host "  基线文件: $($file.FullName)"
            Write-Host "  结果文件: $($latestFile.FullName)"
        }
        
        $baselineStats = $baseline.statistics | ConvertTo-Json -Depth 10 -Compress
        $resultStats = $result.statistics | ConvertTo-Json -Depth 10 -Compress
        
        $statsPass = $false
        if ($baselineStats -eq $resultStats) {
            Write-Host "✓ Statistics验证通过" -ForegroundColor Green
            $statsPass = $true
        } else {
            Write-Host "✗ Statistics验证失败：统计出现问题或统计方法被修改" -ForegroundColor Red
            Write-Host "  基线文件: $($file.FullName)"
            Write-Host "  结果文件: $($latestFile.FullName)"
        }
        
        if ($signalsPass -and $statsPass) {
            $passCount++
        } else {
            $failCount++
        }
        
    } catch {
        Write-Host "✗ 验证失败: $_" -ForegroundColor Red
        $failCount++
    }
    
    Write-Host ""
}

Write-Host "========== 验证总结 ==========" -ForegroundColor Cyan
Write-Host "总计: $($baselineFiles.Count) 个基线文件"
Write-Host "通过: $passCount 个" -ForegroundColor Green
Write-Host "失败: $failCount 个" -ForegroundColor $(if ($failCount -gt 0) { "Red" } else { "Green" })

if ($failCount -gt 0) {
    exit 1
}
