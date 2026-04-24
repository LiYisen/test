---
name: "baseline-verifier"
description: "验证基线结果，确保代码改动未引入回归问题。在每次代码改动后、提交前或用户要求验证时调用。"
---

# 基线验证器 (Baseline Verifier)

此技能用于验证期货回测系统的基线结果，确保代码改动没有引入回归问题。

## 何时调用

**必须调用此技能的场景**：
1. 修改了策略逻辑后
2. 修改了信号价格计算后
3. 修改了移仓换月逻辑后
4. 修改了统计计算方法后
5. 用户明确要求验证基线
6. 在提交代码前进行最终验证

## 验证流程

### 1. 启动Web服务

首先确保Web服务正在运行：

```bash
go run cmd/web/main.go
```

### 2. 执行验证脚本

**Windows PowerShell**:
```powershell
.\scripts\verify_baselines.ps1
```

**Linux/macOS**:
```bash
chmod +x scripts/verify_baselines.sh
./scripts/verify_baselines.sh
```

### 3. 验证内容

验证脚本会自动检查以下内容：

**Signals验证**：
- 对比基线文件和最新结果中的交易信号
- 验证策略执行是否正确
- 检测策略逻辑是否被修改

**Statistics验证**：
- 对比基线文件和最新结果中的统计数据
- 验证统计计算方法是否正确
- 检测收益率和回撤计算是否被修改

### 4. 结果分析

**验证通过**：
```
✓ Signals验证通过
✓ Statistics验证通过
```

**验证失败**：
```
✗ Signals验证失败：策略执行出现问题或策略被修改
✗ Statistics验证失败：统计出现问题或统计方法被修改
```

## 验证失败处理

### Signals验证失败

检查以下内容：
1. 是否修改了策略逻辑
2. 是否修改了信号价格计算
3. 是否修改了移仓换月逻辑
4. 是否修改了止损/反向信号价格计算

### Statistics验证失败

检查以下内容：
1. 是否修改了统计计算方法
2. 是否修改了收益率计算逻辑
3. 是否修改了回撤计算逻辑
4. 是否修改了资金曲线计算

## 更新基线

如果是有意的修改导致结果变化，需要更新基线文件：

**Windows PowerShell**:
```powershell
$symbol = "RB"
$strategy = "yinyang"
$startDate = "20240101"
$endDate = "20241231"

$pattern = "${symbol}_${strategy}_${startDate}_${endDate}_*_*.json"
$latestFile = Get-ChildItem -Path "ret\$pattern" | Sort-Object LastWriteTime -Descending | Select-Object -First 1
Copy-Item $latestFile.FullName -Destination "baselines\${symbol}_${strategy}_${startDate}_${endDate}_baseline.json"
```

**Linux/macOS**:
```bash
symbol="RB"
strategy="yinyang"
start_date="20240101"
end_date="20241231"

pattern="${symbol}_${strategy}_${start_date}_${end_date}_*_*.json"
latest_file=$(ls -t ret/$pattern | head -n 1)
cp "$latest_file" "baselines/${symbol}_${strategy}_${start_date}_${end_date}_baseline.json"
```

## 基线文件位置

基线文件存储在 `baselines/` 目录，文件名格式：
```
{品种}_{策略}_{开始日期}_{结束日期}_baseline.json
```

## 注意事项

1. **每次代码改动后必须验证**：这是防止回归问题的关键步骤
2. **不要跳过验证**：即使是很小的改动也可能影响结果
3. **保持基线最新**：有意的修改需要及时更新基线
4. **记录修改原因**：更新基线时记录修改的原因和内容

## 示例输出

```
开始验证 3 个基线文件...

========== 验证基线: RB_yinyang_20240101_20241231_3_baseline.json ==========
参数: 品种=RB, 策略=yinyang, 开始=20240101, 结束=20241231
✓ Signals验证通过
✓ Statistics验证通过

========== 验证基线: BU_yinyang_20251101_20260414_3_baseline.json ==========
参数: 品种=BU, 策略=yinyang, 开始=20251101, 结束=20260414
✓ Signals验证通过
✓ Statistics验证通过

========== 验证总结 ==========
总计: 3 个基线文件
通过: 3 个
失败: 0 个
```
