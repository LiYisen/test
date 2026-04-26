#!/bin/bash

baselines_dir="baselines"

if [ ! -d "$baselines_dir" ]; then
    echo "错误: baselines目录不存在"
    exit 1
fi

shopt -s nullglob
baseline_files=("$baselines_dir"/*.json)
shopt -u nullglob

if [ ${#baseline_files[@]} -eq 0 ]; then
    echo "警告: baselines目录中没有基线文件"
    exit 0
fi

echo "开始验证 ${#baseline_files[@]} 个基线文件..."
echo ""

pass_count=0
fail_count=0

for file in "${baseline_files[@]}"; do
    echo "========== 验证基线: $(basename "$file") =========="
    
    if ! command -v jq &> /dev/null; then
        echo "✗ 需要安装jq工具"
        exit 1
    fi
    
    symbol=$(jq -r '.request.symbol' "$file")
    start_date=$(jq -r '.request.start_date' "$file")
    end_date=$(jq -r '.request.end_date' "$file")
    strategy=$(jq -r '.request.strategy' "$file")
    
    echo "参数: 品种=$symbol, 策略=$strategy, 开始=$start_date, 结束=$end_date"
    
    body=$(jq -n \
        --arg symbol "$symbol" \
        --arg start_date "$start_date" \
        --arg end_date "$end_date" \
        --arg strategy "$strategy" \
        --argjson params "$(jq '.request.params' "$file")" \
        '{symbol: $symbol, start_date: $start_date, end_date: $end_date, strategy: $strategy, params: $params}')
    
    response=$(curl -s -X POST http://localhost:8080/api/backtest \
        -H "Content-Type: application/json" \
        -d "$body" \
        --max-time 300)
    
    if [ $? -ne 0 ]; then
        echo "✗ 回测执行失败"
        fail_count=$((fail_count + 1))
        echo ""
        continue
    fi
    
    sleep 1
    
    pattern="${symbol}_${strategy}_${start_date}_${end_date}_*_*.json"
    latest_file=$(ls -t ret/$pattern 2>/dev/null | grep -v '_baseline\.json$' | head -n 1)
    
    if [ -z "$latest_file" ]; then
        echo "✗ 未找到结果文件"
        fail_count=$((fail_count + 1))
        echo ""
        continue
    fi
    
    baseline_signals=$(jq -c '.signals' "$file")
    result_signals=$(jq -c '.signals' "$latest_file")
    
    signals_pass=false
    if [ "$baseline_signals" = "$result_signals" ]; then
        echo "✓ Signals验证通过"
        signals_pass=true
    else
        echo "✗ Signals验证失败：策略执行出现问题或策略被修改"
        echo "  基线文件: $file"
        echo "  结果文件: $latest_file"
    fi
    
    baseline_stats=$(jq -c '.statistics' "$file")
    result_stats=$(jq -c '.statistics' "$latest_file")
    
    stats_pass=false
    if [ "$baseline_stats" = "$result_stats" ]; then
        echo "✓ Statistics验证通过"
        stats_pass=true
    else
        echo "✗ Statistics验证失败：统计出现问题或统计方法被修改"
        echo "  基线文件: $file"
        echo "  结果文件: $latest_file"
    fi
    
    if [ "$signals_pass" = true ] && [ "$stats_pass" = true ]; then
        pass_count=$((pass_count + 1))
    else
        fail_count=$((fail_count + 1))
    fi
    
    echo ""
done

echo "========== 验证总结 =========="
echo "总计: ${#baseline_files[@]} 个基线文件"
echo "通过: $pass_count 个"
if [ $fail_count -gt 0 ]; then
    echo "失败: $fail_count 个"
    exit 1
else
    echo "失败: $fail_count 个"
fi
