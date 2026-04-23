// Package pyexec 封装Go调用Python脚本的逻辑
package pyexec

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// DataType 数据类型
type DataType string

const (
	DataTypeKline     DataType = "kline"     // 期货日K线
	DataTypeCalendar  DataType = "calendar"  // 交易日历
	DataTypeIndex     DataType = "index"     // 期货指数
)

// PyExecutor Python脚本执行器
type PyExecutor struct {
	scriptPath string
	timeout    time.Duration
}

// NewPyExecutor 创建新的Python执行器
func NewPyExecutor(scriptPath string, timeout time.Duration) *PyExecutor {
	if timeout == 0 {
		timeout = 60 * time.Second // 默认60秒超时
	}
	return &PyExecutor{
		scriptPath: scriptPath,
		timeout:    timeout,
	}
}

// RunScript 执行Python脚本并返回JSON结果
func (e *PyExecutor) RunScript(dataType DataType, symbol, startDate, endDate string) ([]byte, error) {
	// 验证脚本路径
	if _, err := os.Stat(e.scriptPath); err != nil {
		return nil, fmt.Errorf("脚本文件不存在: %v", err)
	}

	// 构建命令参数
	args := []string{
		e.scriptPath,
		"--type", string(dataType),
		"--start", startDate,
		"--end", endDate,
	}

	// 如果需要symbol参数
	if symbol != "" && dataType != DataTypeCalendar {
		args = append(args, "--symbol", symbol)
	}

	// 创建命令
	cmd := exec.Command("python", args...)
	
	// 设置超时
	done := make(chan []byte, 1)
	errChan := make(chan error, 1)

	go func() {
		output, err := cmd.CombinedOutput()
		if err != nil {
			errChan <- fmt.Errorf("执行Python脚本失败: %v, 输出: %s", err, string(output))
			return
		}
		done <- output
	}()

	// 等待超时或完成
	select {
	case output := <-done:
		return output, nil
	case err := <-errChan:
		return nil, err
	case <-time.After(e.timeout):
		cmd.Process.Kill()
		return nil, fmt.Errorf("执行Python脚本超时(%v)", e.timeout)
	}
}

// ParseKlineData 解析K线数据JSON
func (e *PyExecutor) ParseKlineData(data []byte) ([]map[string]interface{}, error) {
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %v", err)
	}

	// 检查是否有错误
	if errMsg, ok := result["error"].(string); ok && errMsg != "" {
		return nil, fmt.Errorf("Python脚本错误: %s", errMsg)
	}

	// 提取数据
	dataRaw, ok := result["data"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("数据格式错误: data字段不存在或类型错误")
	}

	// 转换为map切片
	klines := make([]map[string]interface{}, len(dataRaw))
	for i, item := range dataRaw {
		klines[i], ok = item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("数据格式错误: 第%d条数据格式错误", i+1)
		}
	}

	return klines, nil
}

// ParseCalendarData 解析日历数据JSON
func (e *PyExecutor) ParseCalendarData(data []byte) ([]map[string]interface{}, error) {
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %v", err)
	}

	// 检查是否有错误
	if errMsg, ok := result["error"].(string); ok && errMsg != "" {
		return nil, fmt.Errorf("Python脚本错误: %s", errMsg)
	}

	// 提取数据
	dataRaw, ok := result["data"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("数据格式错误: data字段不存在或类型错误")
	}

	// 转换为map切片
	calendar := make([]map[string]interface{}, len(dataRaw))
	for i, item := range dataRaw {
		calendar[i], ok = item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("数据格式错误: 第%d条数据格式错误", i+1)
		}
	}

	return calendar, nil
}

// GetFuturesKLine 获取期货日K线数据
func (e *PyExecutor) GetFuturesKLine(symbol, startDate, endDate string) ([]map[string]interface{}, error) {
	data, err := e.RunScript(DataTypeKline, symbol, startDate, endDate)
	if err != nil {
		return nil, err
	}
	return e.ParseKlineData(data)
}

// GetTradeCalendar 获取交易日历
func (e *PyExecutor) GetTradeCalendar(startDate, endDate string) ([]map[string]interface{}, error) {
	data, err := e.RunScript(DataTypeCalendar, "", startDate, endDate)
	if err != nil {
		return nil, err
	}
	return e.ParseCalendarData(data)
}

// GetFuturesIndex 获取期货指数数据
func (e *PyExecutor) GetFuturesIndex(symbol, startDate, endDate string) ([]map[string]interface{}, error) {
	data, err := e.RunScript(DataTypeIndex, symbol, startDate, endDate)
	if err != nil {
		return nil, err
	}
	return e.ParseKlineData(data)
}

// GetDefaultScriptPath 获取默认脚本路径
func GetDefaultScriptPath() string {
	// 获取当前执行文件的目录
	execPath, _ := os.Executable()
	dir := filepath.Dir(execPath)
	
	// 尝试多个可能的路径
	paths := []string{
		filepath.Join(dir, "scripts", "get_futures_data.py"),
		filepath.Join(dir, "..", "scripts", "get_futures_data.py"),
		filepath.Join(dir, "..", "..", "scripts", "get_futures_data.py"),
	}
	
	for _, p := range paths {
		absPath, _ := filepath.Abs(p)
		if _, err := os.Stat(absPath); err == nil {
			return absPath
		}
	}
	
	// 默认返回相对路径
	return "scripts/get_futures_data.py"
}

// NewDefaultExecutor 创建默认执行器
func NewDefaultExecutor() *PyExecutor {
	return NewPyExecutor(GetDefaultScriptPath(), 60*time.Second)
}