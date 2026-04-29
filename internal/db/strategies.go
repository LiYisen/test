package db

import (
	"database/sql"
	"fmt"
)

type Strategy struct {
	Name        string         `json:"name"`
	DisplayName string         `json:"display_name"`
	Description string         `json:"description"`
	Enabled     bool           `json:"enabled"`
	Params      []StrategyParam `json:"params"`
}

type StrategyParam struct {
	Name        string  `json:"name"`
	DisplayName string  `json:"display_name"`
	Type        string  `json:"type"`
	Default     float64 `json:"default"`
	Min         float64 `json:"min"`
	Max         float64 `json:"max"`
	Description string  `json:"description"`
	SortOrder   int     `json:"sort_order"`
}

func UpsertStrategy(s Strategy) error {
	return WithTx(func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			INSERT INTO strategies (name, display_name, description, enabled, updated_at)
			VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT(name) DO UPDATE SET
				display_name = excluded.display_name,
				description = excluded.description,
				enabled = excluded.enabled,
				updated_at = CURRENT_TIMESTAMP
		`, s.Name, s.DisplayName, s.Description, s.Enabled)
		if err != nil {
			return fmt.Errorf("写入策略失败: %w", err)
		}

		_, err = tx.Exec(`DELETE FROM strategy_params WHERE strategy_name = ?`, s.Name)
		if err != nil {
			return fmt.Errorf("清除旧参数失败: %w", err)
		}

		stmt, err := tx.Prepare(`
			INSERT INTO strategy_params (strategy_name, param_name, display_name, type, default_value, min_value, max_value, description, sort_order)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			return fmt.Errorf("准备参数插入语句失败: %w", err)
		}
		defer stmt.Close()

		for i, p := range s.Params {
			if _, err := stmt.Exec(s.Name, p.Name, p.DisplayName, p.Type, p.Default, p.Min, p.Max, p.Description, i); err != nil {
				return fmt.Errorf("写入策略参数 %s.%s 失败: %w", s.Name, p.Name, err)
			}
		}

		return nil
	})
}

func GetAllStrategies() ([]Strategy, error) {
	if globalDB == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}
	rows, err := globalDB.Query(`SELECT name, display_name, description, enabled FROM strategies ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var strategies []Strategy
	for rows.Next() {
		var s Strategy
		var enabled int
		if err := rows.Scan(&s.Name, &s.DisplayName, &s.Description, &enabled); err != nil {
			return nil, err
		}
		s.Enabled = enabled == 1
		strategies = append(strategies, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range strategies {
		params, err := GetStrategyParams(strategies[i].Name)
		if err != nil {
			return nil, err
		}
		strategies[i].Params = params
	}

	return strategies, nil
}

func GetStrategyParams(strategyName string) ([]StrategyParam, error) {
	if globalDB == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}
	rows, err := globalDB.Query(`
		SELECT param_name, display_name, type, default_value, min_value, max_value, description, sort_order
		FROM strategy_params WHERE strategy_name = ? ORDER BY sort_order
	`, strategyName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var params []StrategyParam
	for rows.Next() {
		var p StrategyParam
		if err := rows.Scan(&p.Name, &p.DisplayName, &p.Type, &p.Default, &p.Min, &p.Max, &p.Description, &p.SortOrder); err != nil {
			return nil, err
		}
		params = append(params, p)
	}
	return params, rows.Err()
}

func GetStrategyByName(name string) (*Strategy, error) {
	if globalDB == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}
	var s Strategy
	var enabled int
	err := globalDB.QueryRow(`SELECT name, display_name, description, enabled FROM strategies WHERE name = ?`, name).
		Scan(&s.Name, &s.DisplayName, &s.Description, &enabled)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.Enabled = enabled == 1

	params, err := GetStrategyParams(name)
	if err != nil {
		return nil, err
	}
	s.Params = params

	return &s, nil
}

func DeleteStrategy(name string) error {
	return WithTx(func(tx *sql.Tx) error {
		_, err := tx.Exec(`DELETE FROM strategy_params WHERE strategy_name = ?`, name)
		if err != nil {
			return err
		}
		_, err = tx.Exec(`DELETE FROM strategies WHERE name = ?`, name)
		return err
	})
}

func GetConfigMeta(key string) (string, error) {
	if globalDB == nil {
		return "", fmt.Errorf("数据库未初始化")
	}
	var value string
	err := globalDB.QueryRow(`SELECT value FROM config_meta WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func SetConfigMeta(key, value string) error {
	if globalDB == nil {
		return fmt.Errorf("数据库未初始化")
	}
	_, err := globalDB.Exec(`
		INSERT INTO config_meta (key, value, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP
	`, key, value)
	return err
}
