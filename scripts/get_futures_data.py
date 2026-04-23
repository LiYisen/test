# -*- coding: utf-8 -*-
"""
期货数据获取脚本
使用akshare获取期货日K线数据、交易日历和期货指数行情
输出JSON格式数据供Go解析

用法:
    python get_futures_data.py --type kline --symbol ru2401 --start 20240101 --end 20240401
    python get_futures_data.py --type calendar --start 20240101 --end 20240401
    python get_futures_data.py --type index --symbol ru
"""

import json
import argparse
import sys
from datetime import datetime

try:
    import akshare as ak
except ImportError:
    print(json.dumps({"error": "akshare未安装，请运行: pip install akshare"}))
    sys.exit(1)


def get_futures_daily_sina(symbol: str, start_date: str, end_date: str) -> dict:
    """
    获取期货日K线数据（支持多合约批量获取）
    返回成交量volume、持仓量hold和结算价settle
    """
    try:
        # 格式化日期
        start = f"{start_date[:4]}-{start_date[4:6]}-{start_date[6:8]}"
        end = f"{end_date[:4]}-{end_date[4:6]}-{end_date[6:8]}"
        
        # 获取期货日K线数据
        try:
            df = ak.futures_zh_daily_sina(symbol=symbol)
        except Exception:
            return {"error": f"合约{symbol}获取失败(akshare错误)", "data": []}
        
        if df is None or df.empty:
            return {"error": f"未找到合约{symbol}的数据", "data": []}
        
        # 确保日期列是datetime类型
        df['date'] = pd.to_datetime(df['date'])
        
        # 筛选日期范围
        df = df[(df['date'] >= start) & (df['date'] <= end)]
        
        # 转换数据格式
        result = []
        for _, row in df.iterrows():
            result.append({
                "date": row['date'].strftime('%Y-%m-%d'),
                "open": float(row['open']),
                "high": float(row['high']),
                "low": float(row['low']),
                "close": float(row['close']),
                "volume": float(row['volume']),
                "amount": 0.0,  # akshare此接口不提供成交额
                "hold": float(row['hold']) if 'hold' in row else 0.0,
                "settle": float(row['settle']) if 'settle' in row else 0.0
            })
        
        return {"symbol": symbol, "data": result, "count": len(result)}
        
    except Exception as e:
        return {"error": str(e), "data": []}
def get_trade_calendar_sina(start_date: str, end_date: str) -> dict:
    """
    获取交易日历
    """
    try:
        # 格式化日期
        start = f"{start_date[:4]}-{start_date[4:6]}-{start_date[6:8]}"
        end = f"{end_date[:4]}-{end_date[4:6]}-{end_date[6:8]}"
        
        # 获取交易日历
        df = ak.tool_trade_date_hist_sina()
        
        if df is None or df.empty:
            return {"error": "未获取到交易日历数据", "data": []}
        
        # 确保日期列是datetime类型
        df['trade_date'] = pd.to_datetime(df['trade_date'])
        
        # 筛选日期范围
        df = df[(df['trade_date'] >= start) & (df['trade_date'] <= end)]
        
        # 转换数据格式
        result = []
        for _, row in df.iterrows():
            result.append({
                "date": row['trade_date'].strftime('%Y-%m-%d'),
                "is_trading_day": True  # 所有返回的日期都是交易日
            })
        
        return {"data": result, "count": len(result)}
        
    except Exception as e:
        return {"error": str(e), "data": []}
    except Exception as e:
        return {"error": str(e), "data": []}
def get_futures_index_sina(symbol: str, start_date: str, end_date: str) -> dict:
    """
    获取期货指数行情
    """
    try:
        # 格式化日期
        start = f"{start_date[:4]}-{start_date[4:6]}-{start_date[6:8]}"
        end = f"{end_date[:4]}-{end_date[4:6]}-{end_date[6:8]}"
        
        # 获取期货指数行情
        df = ak.futures_zh_index_sina(symbol=symbol)
        
        if df is None or df.empty:
            return {"error": f"未找到指数{symbol}的数据", "data": []}
        
        # 确保日期列是datetime类型
        df['date'] = pd.to_datetime(df['date'])
        
        # 筛选日期范围
        df = df[(df['date'] >= start) & (df['date'] <= end)]
        
        # 转换数据格式
        result = []
        for _, row in df.iterrows():
            result.append({
                "date": row['date'].strftime('%Y-%m-%d'),
                "open": float(row['open']) if 'open' in row else 0.0,
                "high": float(row['high']) if 'high' in row else 0.0,
                "low": float(row['low']) if 'low' in row else 0.0,
                "close": float(row['close']) if 'close' in row else 0.0,
                "volume": float(row['volume']) if 'volume' in row else 0.0,
                "hold": float(row['hold']) if 'hold' in row else 0.0,
                "settle": float(row['settle']) if 'settle' in row else 0.0
            })
        
        return {"symbol": symbol, "data": result, "count": len(result)}
        
    except Exception as e:
        return {"error": str(e), "data": []}
    except Exception as e:
        return {"error": str(e), "data": []}


def main():
    parser = argparse.ArgumentParser(description='期货数据获取脚本')
    parser.add_argument('--type', type=str, required=True, 
                        choices=['kline', 'calendar', 'index'],
                        help='数据类型: kline(期货日K线), calendar(交易日历), index(期货指数)')
    parser.add_argument('--symbol', type=str, default='',
                        help='期货品种/合约代码 (kline和index类型必需)')
    parser.add_argument('--start', type=str, required=True,
                        help='开始日期，格式: YYYYMMDD')
    parser.add_argument('--end', type=str, required=True,
                        help='结束日期，格式: YYYYMMDD')
    
    args = parser.parse_args()
    
    # 验证参数
    if args.type in ['kline', 'index'] and not args.symbol:
        print(json.dumps({"error": f"类型{args.type}需要指定--symbol参数"}))
        sys.exit(1)
    
    # 调用对应函数
    result = None
    if args.type == 'kline':
        result = get_futures_daily_sina(args.symbol, args.start, args.end)
    elif args.type == 'calendar':
        result = get_trade_calendar_sina(args.start, args.end)
    elif args.type == 'index':
        result = get_futures_index_sina(args.symbol, args.start, args.end)
    
    # 输出JSON
    print(json.dumps(result, ensure_ascii=False, indent=2))


if __name__ == '__main__':
    # 导入pandas
    try:
        import pandas as pd
    except ImportError:
        print(json.dumps({"error": "pandas未安装，请运行: pip install pandas"}))
        sys.exit(1)
    
    main()