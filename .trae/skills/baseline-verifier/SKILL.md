# 基线验证器 (Baseline Verifier)

## 描述

验证基线结果，确保代码改动未引入回归问题。在每次代码改动后、提交前或用户要求验证时调用。

## 何时调用

**必须调用此技能的场景**：
1. 修改了策略逻辑后
2. 修改了信号价格计算后
3. 修改了移仓换月逻辑后
4. 修改了统计计算方法后
5. 用户明确要求验证基线
6. 在提交代码前进行最终验证

## 验证逻辑

### 核心流程

```
1. 遍历 baselines 目录下的基线数据文件
   ↓
2. 构建相同请求参数获取返回结果
   ↓
3. 对比 signals 属性的内容和 statistics 属性的内容
   ↓
4. 分析差异并给出结论
```

### 详细步骤

#### 步骤 1：遍历基线文件
- 扫描 `baselines/` 目录
- 读取所有 `.json` 基线文件
- 提取请求参数（symbol, start_date, end_date, strategy, params）

#### 步骤 2：执行回测
- 使用基线文件的请求参数
- 调用 `/api/backtest` 接口执行回测
- 获取最新的回测结果文件

#### 步骤 3：对比验证

**Signals 验证**：
- 对比基线和结果的 `signals` 数组
- 检查信号数量是否一致
- 逐个对比信号的关键字段：`signal_date`、`price`、`direction`、`symbol`
- 如果不一致，说明策略逻辑被修改

**Statistics 验证**：
- 对比基线和结果的 `statistics` 对象
- 检查关键统计指标：`total_return`、`max_drawdown`、`sharpe_ratio`、`win_rate`
- 如果不一致，说明统计计算方法被修改

#### 步骤 4：给出结论

**Signals 差异**：
- ✅ **策略相关出现了改动引发**
- 可能原因：
  - 修改了策略逻辑
  - 修改了信号价格计算
  - 修改了移仓换月逻辑
  - 修改了止损/反向信号价格计算

**Statistics 差异**：
- ✅ **统计相关出现了改动引发**
- 可能原因：
  - 修改了统计计算方法
  - 修改了收益率计算逻辑
  - 修改了回撤计算逻辑
  - 修改了资金曲线计算

## 验证流程

### 步骤 1：启动 Web 服务

首先确保 Web 服务正在运行：

```bash
go run cmd/web/main.go
```

### 步骤 2：执行验证脚本

验证脚本已集成到此技能中，可以直接运行：

**Windows PowerShell**:
```powershell
# 从项目根目录运行
.\.trae\skills\baseline-verifier\verify_baselines.ps1

# 或使用项目级脚本
.\scripts\verify_baselines.ps1
```

**Linux/macOS**:
```bash
# 从项目根目录运行
chmod +x .trae/skills/baseline-verifier/verify_baselines.sh
./.trae/skills/baseline-verifier/verify_baselines.sh

# 或使用项目级脚本
chmod +x scripts/verify_baselines.sh
./scripts/verify_baselines.sh
```

### 步骤 3：分析结果

验证脚本会自动执行以下操作：

1. **遍历基线文件**：扫描 `baselines/` 目录下的所有基线文件
2. **执行回测**：使用基线参数重新运行回测
3. **对比验证**：对比 `signals` 和 `statistics` 属性
4. **给出结论**：明确指出差异原因

## 结果解读

### 验证通过

```
[PASS] Signals verification passed
[PASS] Statistics verification passed

[SUCCESS] All verifications passed! No regression detected.
```

**结论**：
- ✅ 代码改动未引入回归问题
- ✅ 策略逻辑保持一致
- ✅ 统计计算保持一致

### 验证失败

```
[FAIL] Signals verification failed
  Baseline signals: 50, Result signals: 48
  → 策略相关出现了改动引发

[FAIL] Statistics verification failed
  Baseline total_return: 0.25, Result total_return: 0.23
  → 统计相关出现了改动引发

[WARNING] Regression detected! Please check the failed verifications above.
```

**结论**：
- ❌ 检测到回归问题
- 🔍 需要分析差异原因
- 📝 决定是修复问题还是更新基线

## 差异分析

### Signals 差异分析

**信号数量不一致**：
```
Baseline signals: 50
Result signals: 48
```

**可能原因**：
1. 修改了策略逻辑，导致信号生成规则变化
2. 修改了信号价格计算，导致触发条件变化
3. 修改了移仓换月逻辑，影响信号连续性
4. 修改了止损/反向信号价格，影响平仓时机

**信号内容不一致**：
```
Baseline: signal_date=2024-01-05, price=3983, direction=1
Result:   signal_date=2024-01-05, price=3985, direction=1
```

**可能原因**：
1. 信号价格计算逻辑被修改
2. K线数据源发生变化
3. 价格精度处理方式改变

### Statistics 差异分析

**总收益率不一致**：
```
Baseline total_return: 0.25
Result total_return: 0.23
```

**可能原因**：
1. 收益率计算公式被修改
2. 资金曲线计算逻辑变化
3. 保证金计算方式改变

**最大回撤不一致**：
```
Baseline max_drawdown: 0.15
Result max_drawdown: 0.18
```

**可能原因**：
1. 回撤计算方法被修改
2. 资金曲线计算逻辑变化
3. 峰值计算方式改变

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

## 重要提示

1. **每次代码改动后必须验证**：这是防止回归问题的关键步骤
2. **不要跳过验证**：即使是很小的改动也可能影响结果
3. **保持基线最新**：有意的修改需要及时更新基线
4. **记录修改原因**：更新基线时记录修改的原因和内容

## 示例输出

```
Starting verification of 2 baseline files...

========== Verifying: RB_yinyang_20240101_20241231_3_baseline.json ==========
Parameters: Symbol=RB, Strategy=yinyang, Start=20240101, End=20241231
Running backtest...
Result file: RB_yinyang_20240101_20241231_3_1777021234.json

Verifying signals...
[PASS] Signals verification passed (73 signals)

Verifying statistics...
[PASS] Statistics verification passed

========== Verifying: BU_yinyang_20251101_20260414_3_baseline.json ==========
Parameters: Symbol=BU, Strategy=yinyang, Start=20251101, End=20260414
Running backtest...
Result file: BU_yinyang_20251101_20260414_3_1777021235.json

Verifying signals...
[FAIL] Signal count mismatch
  Baseline: 50 signals
  Result:   48 signals
  >>> Strategy-related changes detected <<<

Verifying statistics...
[FAIL] Statistics verification failed
  Different fields: total_return, max_drawdown
  >>> Statistics-related changes detected <<<

========== Summary ==========
Total: 2 baseline files
Passed: 1
Failed: 1

[WARNING] Regression detected! Please check the failed verifications above.
If this is intentional, please update the baseline files.
```

## 为什么基线检测至关重要

基线检测是**最后一道防线**，能够：

1. **代码正确性**：任何修改策略逻辑、信号计算或统计方法都会被检测到
2. **结果一致性**：确保结果在代码改动后保持一致
3. **早期发现**：在问题到达生产环境之前捕获
4. **文档记录**：基线文件作为预期行为的文档

## 常见错误

1. **跳过验证**："只是一个小改动" - 小改动可能产生大影响
2. **不更新基线**：有意的修改需要更新基线文件
3. **忽略失败**：验证失败意味着有问题，立即调查
4. **不运行验证**：即使认为什么都没改变，也要运行验证

## 与开发流程集成

1. **提交前**：运行基线验证
2. **合并后**：在主分支运行基线验证
3. **发布前**：运行完整测试套件的基线验证
4. **部署后**：监控生产结果与基线的对比

## 改进历史

### v2.0 (2026-04-25)

**改进内容**：
1. **修复JSON解析问题**
   - 添加 `-Encoding UTF8` 参数，解决大文件和特殊字符解析问题
   - 提高JSON解析的稳定性和兼容性

2. **优化对比逻辑**
   - 只对比 `signals` 和 `statistics` 关键字段
   - 移除JSON序列化对比，改为逐字段对比
   - 避免因JSON格式差异（如 `params: null`）导致的假阳性

3. **简化输出信息**
   - 输出汇总信息而非详细差异
   - 只显示关键统计指标差异
   - 提高可读性和效率

**对比方式**：
- **Signals**: 检查信号数量，逐个对比 `signal_date`、`price`、`direction`、`symbol`
- **Statistics**: 对比 `total_return`、`max_drawdown`、`sharpe_ratio`、`win_rate`

**测试结果**：
```
Starting verification of 2 baseline files...

========== Verifying: BU_yinyang_20251101_20260414_3_1777006858.json ==========
Parameters: Symbol=BU, Strategy=yinyang, Start=20251101, End=20260414
Running backtest...
Result file: BU_yinyang_20251101_20260414_3_1777090412.json

Verifying signals...
[PASS] Signals verification passed (55 signals)

Verifying statistics...
[PASS] Statistics verification passed

========== Verifying: RB_yinyang_20240101_20241231_3_1776998613.json ==========
Parameters: Symbol=RB, Strategy=yinyang, Start=20240101, End=20241231
Running backtest...
Result file: RB_yinyang_20240101_20241231_3_1777090413.json

Verifying signals...
[PASS] Signals verification passed (73 signals)

Verifying statistics...
[PASS] Statistics verification passed

========== Summary ==========
Total: 2 baseline files
Passed: 2
Failed: 0

[SUCCESS] All verifications passed! No regression detected.
```
