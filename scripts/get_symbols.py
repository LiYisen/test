# -*- coding: utf-8 -*-
"""
期货品种列表获取脚本
使用akshare获取国内期货交易所所有品种信息
输出JSON格式数据供Go解析

用法:
    python get_symbols.py
"""

import json
import sys

try:
    import akshare as ak
except ImportError:
    print(json.dumps({"error": "akshare未安装，请运行: pip install akshare"}, ensure_ascii=False))
    sys.exit(1)


def get_pinyin_initials(text):
    """
    获取中文文字的拼音首字母
    简化版本：使用常见字的首字母映射
    """
    pinyin_map = {
        '螺': 'l', '纹': 'w', '钢': 'g',
        '热': 'r', '轧': 'z', '卷': 'j', '板': 'b',
        '铁': 't', '矿': 'k', '石': 's',
        '焦': 'j', '炭': 't', '煤': 'm',
        '黄': 'h', '金': 'j',
        '白': 'b', '银': 'y',
        '铜': 't', '铝': 'l', '锌': 'x', '铅': 'q', '镍': 'n', '锡': 'x',
        '原': 'y', '油': 'y',
        '燃': 'r', '料': 'l',
        '橡': 'x', '胶': 'j',
        '棉': 'm', '花': 'h',
        '白': 'b', '糖': 't',
        '玉': 'y', '米': 'm',
        '大': 'd', '豆': 'd',
        '豆': 'd', '粕': 'p',
        '豆': 'd', '油': 'y',
        '棕': 'z', '榈': 'l',
        '鸡': 'j', '蛋': 'd',
        '生': 's', '猪': 'z',
        '苹': 'p', '果': 'g',
        '红': 'h', '枣': 'z',
        '花': 'h', '生': 's',
        '菜': 'c', '籽': 'z',
        '菜': 'c', '油': 'y',
        '菜': 'c', '粕': 'p',
        '早': 'z', '籼': 'x', '稻': 'd',
        '强': 'q', '麦': 'm',
        '普': 'p', '麦': 'm',
        '棉': 'm', '纱': 's',
        'PTA': 'pta',
        '甲': 'j', '醇': 'c',
        '乙': 'y', '二': 'e', '醇': 'c',
        '尿': 'n', '素': 's',
        '纯': 'c', '碱': 'j',
        '玻': 'b', '璃': 'l',
        '硅': 'g', '铁': 't',
        '锰': 'm', '硅': 'g',
        '线': 'x', '材': 'c',
        '沪': 'h', '深': 's',
        '中': 'z', '证': 'z',
        '五': 'w', '债': 'z',
        '十': 's', '年': 'n',
        '二': 'e', '年': 'n',
        '三': 's', '十': 's',
        '国': 'g', '债': 'z',
        '集': 'j', '运': 'y',
        '欧': 'o', '线': 'x',
        '碳酸锂': 'tsl',
        '工': 'g', '业': 'y',
        '硅': 'g',
        '碳': 't', '酸': 's', '锂': 'l',
    }
    
    result = []
    for char in text:
        if char in pinyin_map:
            result.append(pinyin_map[char])
        elif '\u4e00' <= char <= '\u9fff':
            code = ord(char) - 0x4e00
            if 0 <= code < 399:
                result.append('abcdz'[code // 100])
            elif 400 <= code < 600:
                result.append('efgh'[code // 100 - 4])
            elif 600 <= code < 800:
                result.append('jklm'[code // 100 - 6])
            elif 800 <= code < 900:
                result.append('nopq'[code // 100 - 8])
            elif 900 <= code < 1100:
                result.append('rstw'[code // 100 - 9])
            elif 1100 <= code < 1300:
                result.append('xyz'[code // 100 - 11])
            else:
                result.append(char.lower())
        elif char.isalpha():
            result.append(char.lower())
    return ''.join(result)


def get_all_futures_symbols():
    """
    获取国内期货交易所所有品种信息
    包括：品种代码、品种名称、交易所、拼音首字母
    """
    symbols = []
    seen = set()
    
    exchange_map = {
        "上海期货交易所": "SHFE",
        "大连商品交易所": "DCE",
        "郑州商品交易所": "CZCE",
        "中国金融期货交易所": "CFFEX",
        "上海国际能源交易中心": "INE",
        "广州期货交易所": "GFEX"
    }
    
    try:
        df = ak.futures_display_main_sina()
        
        if df is None or df.empty:
            return {"error": "获取品种列表失败", "symbols": []}
        
        for _, row in df.iterrows():
            symbol = str(row.get('symbol', ''))
            name = str(row.get('name', ''))
            exchange_name = str(row.get('exchange', ''))
            
            if not symbol or not name:
                continue
            
            variety = ''.join([c for c in symbol if c.isalpha()])
            
            if variety in seen:
                continue
            seen.add(variety)
            
            exchange = exchange_map.get(exchange_name, exchange_name)
            
            name = name.replace('连续', '').replace('指数', '').replace('主力', '').strip()
            
            pinyin = get_pinyin_initials(name)
            
            symbols.append({
                "code": variety.upper(),
                "name": name,
                "exchange": exchange,
                "pinyin": pinyin
            })
        
        return {"symbols": symbols}
        
    except Exception as e:
        return {"error": str(e), "symbols": []}


if __name__ == "__main__":
    result = get_all_futures_symbols()
    print(json.dumps(result, ensure_ascii=False))
