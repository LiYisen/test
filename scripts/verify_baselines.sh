#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

if [ -d "$PROJECT_ROOT/baselines" ]; then
    BASELINES_DIR="$PROJECT_ROOT/baselines"
elif [ -d "baselines" ]; then
    BASELINES_DIR="baselines"
else
    echo -e "\033[31mError: baselines directory not found\033[0m"
    exit 1
fi

API_BASE="http://localhost:8080/api"

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

if ! command -v jq &> /dev/null; then
    echo -e "\033[31mError: jq is required but not installed\033[0m"
    exit 1
fi

echo -e "\033[36mStarting verification of $BASELINE_COUNT baseline files...\033[0m"
echo -e "\033[90mData source: SQLite database (via API)\033[0m"
echo ""

PASS_COUNT=0
FAIL_COUNT=0

normalize_direction() {
    case "$1" in
        0) echo "Buy" ;;
        1) echo "Sell" ;;
        2) echo "Close" ;;
        3) echo "CloseShort" ;;
        4) echo "CloseLong" ;;
        *) echo "$1" ;;
    esac
}

normalize_price() {
    echo "$1" | sed 's/\.0*$//' | sed 's/\(\.[0-9]*[1-9]\)0*$/\1/'
}

stats_equal() {
    local b="$1"
    local r="$2"
    local b_fmt=$(printf "%.4f" "$b" 2>/dev/null)
    if [ "$b_fmt" = "$r" ]; then
        return 0
    fi
    local diff=$(echo "$b $r" | awk '{if($1==$2) print 0; else print ($1-$2)>0?$1-$2:$2-$1}')
    local tol="0.0001"
    if [ "$(echo "$diff < $tol" | bc -l 2>/dev/null || echo "0")" = "1" ]; then
        return 0
    fi
    return 1
}

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
    RESPONSE=$(curl -s -X POST "$API_BASE/backtest" \
        -H "Content-Type: application/json" \
        -d "$BODY" \
        --max-time 300)

    if [ $? -ne 0 ]; then
        echo -e "\033[31m[FAIL] Backtest request failed\033[0m"
        FAIL_COUNT=$((FAIL_COUNT + 1))
        echo ""
        continue
    fi

    RESULT_ID=$(echo "$RESPONSE" | jq -r '.id // empty')
    if [ -z "$RESULT_ID" ]; then
        echo -e "\033[31m[FAIL] No result ID in response\033[0m"
        echo -e "\033[90mResponse: $RESPONSE\033[0m"
        FAIL_COUNT=$((FAIL_COUNT + 1))
        echo ""
        continue
    fi

    echo -e "\033[90mResult ID: $RESULT_ID\033[0m"

    sleep 1

    echo ""
    echo -e "\033[33mVerifying signals...\033[0m"

    RESULT_SIGNALS=$(curl -s "$API_BASE/results/$RESULT_ID/data?type=signals" | jq -c '.signals')
    if [ -z "$RESULT_SIGNALS" ] || [ "$RESULT_SIGNALS" = "null" ]; then
        echo -e "\033[31m[FAIL] Failed to fetch signals from API\033[0m"
        FAIL_COUNT=$((FAIL_COUNT + 1))
        echo ""
        continue
    fi

    BASELINE_SIGNAL_COUNT=$(jq '.signals | length' "$FILE")
    RESULT_SIGNAL_COUNT=$(echo "$RESULT_SIGNALS" | jq 'length')

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
            R_DATE=$(echo "$RESULT_SIGNALS" | jq -r ".[$i].date")
            B_PRICE=$(jq -r ".signals[$i].price" "$FILE")
            R_PRICE=$(echo "$RESULT_SIGNALS" | jq -r ".[$i].price")
            B_DIR=$(jq -r ".signals[$i].direction" "$FILE")
            R_DIR=$(echo "$RESULT_SIGNALS" | jq -r ".[$i].direction")
            B_SYM=$(jq -r ".signals[$i].symbol" "$FILE")
            R_SYM=$(echo "$RESULT_SIGNALS" | jq -r ".[$i].symbol")

            B_DIR_NORM=$(normalize_direction "$B_DIR")
            B_PRICE_NORM=$(normalize_price "$B_PRICE")
            R_PRICE_NORM=$(normalize_price "$R_PRICE")

            if [ "$B_DATE" != "$R_DATE" ] || [ "$B_PRICE_NORM" != "$R_PRICE_NORM" ] || \
               [ "$B_DIR_NORM" != "$R_DIR" ] || [ "$B_SYM" != "$R_SYM" ]; then
                ALL_MATCH=false
                echo -e "\033[90m  Mismatch at signal $i:\033[0m"
                echo -e "\033[90m    Baseline: date=$B_DATE price=$B_PRICE_NORM dir=$B_DIR_NORM sym=$B_SYM\033[0m"
                echo -e "\033[90m    Result:   date=$R_DATE price=$R_PRICE_NORM dir=$R_DIR sym=$R_SYM\033[0m"
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

    RESULT_STATS=$(curl -s "$API_BASE/results/$RESULT_ID/data?type=stats" | jq -c '.statistics')
    if [ -z "$RESULT_STATS" ] || [ "$RESULT_STATS" = "null" ]; then
        echo -e "\033[31m[FAIL] Failed to fetch statistics from API\033[0m"
        FAIL_COUNT=$((FAIL_COUNT + 1))
        echo ""
        continue
    fi

    STATS_PASS=true
    DIFF_STATS=""

    B_TOTAL_RET=$(jq -r '.statistics.TotalReturn' "$FILE")
    R_TOTAL_RET=$(echo "$RESULT_STATS" | jq -r '.total_return')
    if ! stats_equal "$B_TOTAL_RET" "$R_TOTAL_RET"; then
        STATS_PASS=false
        DIFF_STATS="$DIFF_STATS total_return"
    fi

    B_MAX_DD=$(jq -r '.statistics.MaxDrawdown' "$FILE")
    R_MAX_DD=$(echo "$RESULT_STATS" | jq -r '.max_drawdown')
    if ! stats_equal "$B_MAX_DD" "$R_MAX_DD"; then
        STATS_PASS=false
        DIFF_STATS="$DIFF_STATS max_drawdown"
    fi

    B_SHARPE=$(jq -r '.statistics.SharpeRatio' "$FILE")
    R_SHARPE=$(echo "$RESULT_STATS" | jq -r '.sharpe_ratio')
    if ! stats_equal "$B_SHARPE" "$R_SHARPE"; then
        STATS_PASS=false
        DIFF_STATS="$DIFF_STATS sharpe_ratio"
    fi

    B_WIN_RATE=$(jq -r '.statistics.WinRate' "$FILE")
    R_WIN_RATE=$(echo "$RESULT_STATS" | jq -r '.win_rate')
    if ! stats_equal "$B_WIN_RATE" "$R_WIN_RATE"; then
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

    echo -e "\033[33mCleaning up result: $RESULT_ID\033[0m"
    curl -s -X DELETE "$API_BASE/results/$RESULT_ID" > /dev/null

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
