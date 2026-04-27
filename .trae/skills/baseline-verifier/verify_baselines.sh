#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BASELINES_DIR="$PROJECT_ROOT/baselines"
RET_DIR="$PROJECT_ROOT/ret"

if [ ! -d "$BASELINES_DIR" ]; then
    echo -e "\033[31mError: baselines directory not found\033[0m"
    exit 1
fi

BASELINE_FILES=$(find "$BASELINES_DIR" -name "*.json" -type f)
BASELINE_COUNT=$(echo "$BASELINE_FILES" | grep -c .)

if [ "$BASELINE_COUNT" -eq 0 ]; then
    echo -e "\033[33mWarning: No baseline files found\033[0m"
    exit 0
fi

echo -e "\033[36mStarting verification of $BASELINE_COUNT baseline files...\033[0m"
echo ""

PASS_COUNT=0
FAIL_COUNT=0

for FILE in $BASELINE_FILES; do
    FILENAME=$(basename "$FILE")
    echo -e "\033[36m========== Verifying: $FILENAME ==========\033[0m"
    
    SYMBOL=$(jq -r '.request.symbol' "$FILE")
    START_DATE=$(jq -r '.request.start_date' "$FILE")
    END_DATE=$(jq -r '.request.end_date' "$FILE")
    STRATEGY=$(jq -r '.request.strategy' "$FILE")
    PARAMS=$(jq -r '.request.params' "$FILE")
    
    echo "Parameters: Symbol=$SYMBOL, Strategy=$STRATEGY, Start=$START_DATE, End=$END_DATE"
    
    BODY=$(jq -n \
        --arg symbol "$SYMBOL" \
        --arg start_date "$START_DATE" \
        --arg end_date "$END_DATE" \
        --arg strategy "$STRATEGY" \
        --argjson params "$PARAMS" \
        '{symbol: $symbol, start_date: $start_date, end_date: $end_date, strategy: $strategy, params: $params}')
    
    echo -e "\033[33mRunning backtest...\033[0m"
    RESPONSE=$(curl -s -X POST http://localhost:8080/api/backtest \
        -H "Content-Type: application/json" \
        -d "$BODY")
    
    sleep 1
    
    PATTERN="${SYMBOL}_${STRATEGY}_${START_DATE}_${END_DATE}_*_*.json"
    LATEST_FILE=$(ls -t "$RET_DIR"/$PATTERN 2>/dev/null | head -n 1)
    
    if [ -z "$LATEST_FILE" ]; then
        echo -e "\033[31m[FAIL] Result file not found\033[0m"
        FAIL_COUNT=$((FAIL_COUNT + 1))
        continue
    fi
    
    LATEST_FILENAME=$(basename "$LATEST_FILE")
    echo -e "\033[90mResult file: $LATEST_FILENAME\033[0m"
    
    echo ""
    echo -e "\033[33mVerifying signals...\033[0m"
    
    BASELINE_SIGNAL_COUNT=$(jq '.signals | length' "$FILE")
    RESULT_SIGNAL_COUNT=$(jq '.signals | length' "$LATEST_FILE")
    
    SIGNALS_PASS=true
    if [ "$BASELINE_SIGNAL_COUNT" -ne "$RESULT_SIGNAL_COUNT" ]; then
        echo -e "\033[31m[FAIL] Signal count mismatch\033[0m"
        echo -e "\033[90m  Baseline: $BASELINE_SIGNAL_COUNT signals\033[0m"
        echo -e "\033[90m  Result:   $RESULT_SIGNAL_COUNT signals\033[0m"
        echo -e "\033[31m  >>> Strategy-related changes detected <<<\033[0m"
        SIGNALS_PASS=false
    else
        ALL_MATCH=true
        for ((i=0; i<BASELINE_SIGNAL_COUNT; i++)); do
            B_DATE=$(jq -r ".signals[$i].signal_date" "$FILE")
            R_DATE=$(jq -r ".signals[$i].signal_date" "$LATEST_FILE")
            B_PRICE=$(jq -r ".signals[$i].price" "$FILE")
            R_PRICE=$(jq -r ".signals[$i].price" "$LATEST_FILE")
            B_DIR=$(jq -r ".signals[$i].direction" "$FILE")
            R_DIR=$(jq -r ".signals[$i].direction" "$LATEST_FILE")
            B_SYM=$(jq -r ".signals[$i].symbol" "$FILE")
            R_SYM=$(jq -r ".signals[$i].symbol" "$LATEST_FILE")
            
            if [ "$B_DATE" != "$R_DATE" ] || [ "$B_PRICE" != "$R_PRICE" ] || \
               [ "$B_DIR" != "$R_DIR" ] || [ "$B_SYM" != "$R_SYM" ]; then
                ALL_MATCH=false
                break
            fi
        done
        
        if [ "$ALL_MATCH" = true ]; then
            echo -e "\033[32m[PASS] Signals verification passed ($BASELINE_SIGNAL_COUNT signals)\033[0m"
        else
            echo -e "\033[31m[FAIL] Signal content mismatch\033[0m"
            echo -e "\033[31m  >>> Strategy-related changes detected <<<\033[0m"
            SIGNALS_PASS=false
        fi
    fi
    
    echo ""
    echo -e "\033[33mVerifying statistics...\033[0m"
    
    STATS_PASS=true
    DIFF_STATS=""
    
    B_TOTAL_RET=$(jq -r '.statistics.total_return' "$FILE")
    R_TOTAL_RET=$(jq -r '.statistics.total_return' "$LATEST_FILE")
    if [ "$B_TOTAL_RET" != "$R_TOTAL_RET" ]; then
        STATS_PASS=false
        DIFF_STATS="$DIFF_STATS total_return"
    fi
    
    B_MAX_DD=$(jq -r '.statistics.max_drawdown' "$FILE")
    R_MAX_DD=$(jq -r '.statistics.max_drawdown' "$LATEST_FILE")
    if [ "$B_MAX_DD" != "$R_MAX_DD" ]; then
        STATS_PASS=false
        DIFF_STATS="$DIFF_STATS max_drawdown"
    fi
    
    B_SHARPE=$(jq -r '.statistics.sharpe_ratio' "$FILE")
    R_SHARPE=$(jq -r '.statistics.sharpe_ratio' "$LATEST_FILE")
    if [ "$B_SHARPE" != "$R_SHARPE" ]; then
        STATS_PASS=false
        DIFF_STATS="$DIFF_STATS sharpe_ratio"
    fi
    
    B_WIN_RATE=$(jq -r '.statistics.win_rate' "$FILE")
    R_WIN_RATE=$(jq -r '.statistics.win_rate' "$LATEST_FILE")
    if [ "$B_WIN_RATE" != "$R_WIN_RATE" ]; then
        STATS_PASS=false
        DIFF_STATS="$DIFF_STATS win_rate"
    fi
    
    if [ "$STATS_PASS" = true ]; then
        echo -e "\033[32m[PASS] Statistics verification passed\033[0m"
    else
        echo -e "\033[31m[FAIL] Statistics verification failed\033[0m"
        echo -e "\033[90m  Different fields:$DIFF_STATS\033[0m"
        echo -e "\033[31m  >>> Statistics-related changes detected <<<\033[0m"
    fi
    
    if [ "$SIGNALS_PASS" = true ] && [ "$STATS_PASS" = true ]; then
        PASS_COUNT=$((PASS_COUNT + 1))
    else
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
    
    if [ -f "$LATEST_FILE" ]; then
        rm -f "$LATEST_FILE"
        echo -e "\033[90mCleaned up result file: $(basename "$LATEST_FILE")\033[0m"
    fi
    
    echo ""
done

echo -e "\033[36m========== Summary ==========\033[0m"
echo "Total: $BASELINE_COUNT baseline files"
echo -e "\033[32mPassed: $PASS_COUNT\033[0m"

if [ $FAIL_COUNT -gt 0 ]; then
    echo -e "\033[31mFailed: $FAIL_COUNT\033[0m"
    echo ""
    echo -e "\033[31m[WARNING] Regression detected! Please check the failed verifications above.\033[0m"
    echo -e "\033[33mIf this is intentional, please update the baseline files.\033[0m"
    exit 1
else
    echo -e "\033[32mFailed: $FAIL_COUNT\033[0m"
    echo ""
    echo -e "\033[32m[SUCCESS] All verifications passed! No regression detected.\033[0m"
fi
