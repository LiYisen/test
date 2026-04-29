package db

import (
	"database/sql"
	"fmt"
)

type Symbol struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Exchange string `json:"exchange"`
	Pinyin   string `json:"pinyin"`
}

func UpsertSymbol(symbol Symbol) error {
	if globalDB == nil {
		return fmt.Errorf("数据库未初始化")
	}
	_, err := globalDB.Exec(`
		INSERT INTO symbols (code, name, exchange, pinyin, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(code) DO UPDATE SET
			name = excluded.name,
			exchange = excluded.exchange,
			pinyin = excluded.pinyin,
			updated_at = CURRENT_TIMESTAMP
	`, symbol.Code, symbol.Name, symbol.Exchange, symbol.Pinyin)
	return err
}

func UpsertSymbols(symbols []Symbol) error {
	return WithTx(func(tx *sql.Tx) error {
		return UpsertSymbolsTx(tx, symbols)
	})
}

func UpsertSymbolsTx(tx *sql.Tx, symbols []Symbol) error {
	stmt, err := tx.Prepare(`
		INSERT INTO symbols (code, name, exchange, pinyin, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(code) DO UPDATE SET
			name = excluded.name,
			exchange = excluded.exchange,
			pinyin = excluded.pinyin,
			updated_at = CURRENT_TIMESTAMP
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, s := range symbols {
		if _, err := stmt.Exec(s.Code, s.Name, s.Exchange, s.Pinyin); err != nil {
			return fmt.Errorf("写入品种 %s 失败: %w", s.Code, err)
		}
	}
	return nil
}

func GetAllSymbols() ([]Symbol, error) {
	if globalDB == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}
	rows, err := globalDB.Query(`SELECT code, name, exchange, pinyin FROM symbols ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var symbols []Symbol
	for rows.Next() {
		var s Symbol
		if err := rows.Scan(&s.Code, &s.Name, &s.Exchange, &s.Pinyin); err != nil {
			return nil, err
		}
		symbols = append(symbols, s)
	}
	return symbols, rows.Err()
}

func SearchSymbols(query string) ([]Symbol, error) {
	if globalDB == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}
	pattern := "%" + query + "%"
	rows, err := globalDB.Query(`
		SELECT code, name, exchange, pinyin FROM symbols
		WHERE code LIKE ? OR name LIKE ? OR pinyin LIKE ? OR exchange LIKE ?
		ORDER BY code
	`, pattern, pattern, pattern, pattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var symbols []Symbol
	for rows.Next() {
		var s Symbol
		if err := rows.Scan(&s.Code, &s.Name, &s.Exchange, &s.Pinyin); err != nil {
			return nil, err
		}
		symbols = append(symbols, s)
	}
	return symbols, rows.Err()
}

func GetSymbolByCode(code string) (*Symbol, error) {
	if globalDB == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}
	var s Symbol
	err := globalDB.QueryRow(`SELECT code, name, exchange, pinyin FROM symbols WHERE code = ?`, code).
		Scan(&s.Code, &s.Name, &s.Exchange, &s.Pinyin)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func DeleteSymbol(code string) error {
	if globalDB == nil {
		return fmt.Errorf("数据库未初始化")
	}
	_, err := globalDB.Exec(`DELETE FROM symbols WHERE code = ?`, code)
	return err
}

func GetSymbolCount() (int, error) {
	if globalDB == nil {
		return 0, fmt.Errorf("数据库未初始化")
	}
	var count int
	err := globalDB.QueryRow(`SELECT COUNT(*) FROM symbols`).Scan(&count)
	return count, err
}
