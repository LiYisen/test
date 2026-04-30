package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

var (
	globalDB *sql.DB
	dbMu     sync.RWMutex
)

func InitDB(dbPath string) error {
	if globalDB != nil {
		return nil
	}
	return initDatabase(dbPath)
}

func ResetDB(dbPath string) error {
	if globalDB != nil {
		globalDB.Close()
		globalDB = nil
	}
	return initDatabase(dbPath)
}

func initDatabase(dbPath string) error {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建数据库目录失败: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return fmt.Errorf("打开数据库失败: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("连接数据库失败: %w", err)
	}

	if err := createTables(db); err != nil {
		db.Close()
		return fmt.Errorf("创建表失败: %w", err)
	}

	globalDB = db
	return nil
}

func createTables(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS symbols (
			code TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			exchange TEXT NOT NULL,
			pinyin TEXT NOT NULL DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_symbols_exchange ON symbols(exchange)`,
		`CREATE INDEX IF NOT EXISTS idx_symbols_pinyin ON symbols(pinyin)`,

		`CREATE TABLE IF NOT EXISTS funds (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			start_date TEXT NOT NULL DEFAULT '',
			end_date TEXT NOT NULL DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS fund_positions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			fund_id TEXT NOT NULL,
			symbol TEXT NOT NULL,
			strategy TEXT NOT NULL,
			weight REAL NOT NULL,
			params TEXT NOT NULL DEFAULT '{}',
			sort_order INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY (fund_id) REFERENCES funds(id) ON DELETE CASCADE,
			UNIQUE(fund_id, symbol, strategy)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_fund_positions_fund ON fund_positions(fund_id)`,

		`CREATE TABLE IF NOT EXISTS backtest_results (
			id TEXT PRIMARY KEY,
			symbol TEXT NOT NULL,
			strategy TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			leverage REAL NOT NULL DEFAULT 3.0,
			total_return REAL NOT NULL DEFAULT 0,
			annual_return REAL NOT NULL DEFAULT 0,
			max_drawdown REAL NOT NULL DEFAULT 0,
			max_drawdown_ratio REAL NOT NULL DEFAULT 0,
			win_rate REAL NOT NULL DEFAULT 0,
			profit_loss_ratio REAL NOT NULL DEFAULT 0,
			winning_trades INTEGER NOT NULL DEFAULT 0,
			losing_trades INTEGER NOT NULL DEFAULT 0,
			total_trades INTEGER NOT NULL DEFAULT 0,
			total_win REAL NOT NULL DEFAULT 0,
			total_loss REAL NOT NULL DEFAULT 0,
			sharpe_ratio REAL NOT NULL DEFAULT 0,
			calmar_ratio REAL NOT NULL DEFAULT 0,
			trading_days INTEGER NOT NULL DEFAULT 0,
			final_value REAL NOT NULL DEFAULT 0,
			signals TEXT NOT NULL DEFAULT '[]',
			daily_records TEXT NOT NULL DEFAULT '[]',
			position_returns TEXT NOT NULL DEFAULT '[]',
			state_history TEXT NOT NULL DEFAULT '[]',
			dominant_map TEXT NOT NULL DEFAULT '{}',
			klines TEXT NOT NULL DEFAULT '[]',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_backtest_results_symbol ON backtest_results(symbol)`,
		`CREATE INDEX IF NOT EXISTS idx_backtest_results_strategy ON backtest_results(strategy)`,
		`CREATE INDEX IF NOT EXISTS idx_backtest_results_dates ON backtest_results(start_date, end_date)`,

		`CREATE TABLE IF NOT EXISTS fund_results (
			id TEXT PRIMARY KEY,
			fund_id TEXT NOT NULL,
			fund_name TEXT NOT NULL DEFAULT '',
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			timestamp INTEGER NOT NULL DEFAULT 0,
			total_return REAL NOT NULL DEFAULT 0,
			annual_return REAL NOT NULL DEFAULT 0,
			max_drawdown REAL NOT NULL DEFAULT 0,
			max_drawdown_ratio REAL NOT NULL DEFAULT 0,
			sharpe_ratio REAL NOT NULL DEFAULT 0,
			calmar_ratio REAL NOT NULL DEFAULT 0,
			win_rate REAL NOT NULL DEFAULT 0,
			trading_days INTEGER NOT NULL DEFAULT 0,
			winning_trades INTEGER NOT NULL DEFAULT 0,
			losing_trades INTEGER NOT NULL DEFAULT 0,
			total_trades INTEGER NOT NULL DEFAULT 0,
			daily_records TEXT NOT NULL DEFAULT '[]',
			position_results TEXT NOT NULL DEFAULT '{}',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (fund_id) REFERENCES funds(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_fund_results_fund ON fund_results(fund_id)`,
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("执行建表语句失败 [%s]: %w", stmt[:60], err)
		}
	}

	return nil
}

func GetDB() *sql.DB {
	return globalDB
}

func CloseDB() error {
	if globalDB != nil {
		return globalDB.Close()
	}
	return nil
}

func BeginTx() (*sql.Tx, error) {
	if globalDB == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}
	return globalDB.Begin()
}

func WithTx(fn func(tx *sql.Tx) error) error {
	tx, err := BeginTx()
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func Lock() {
	dbMu.Lock()
}

func Unlock() {
	dbMu.Unlock()
}

func RLock() {
	dbMu.RLock()
}

func RUnlock() {
	dbMu.RUnlock()
}

func GetDefaultDBPath() string {
	return filepath.Join("db", "futures.db")
}
