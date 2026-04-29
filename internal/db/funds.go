package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

type Fund struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	StartDate   string       `json:"start_date"`
	EndDate     string       `json:"end_date"`
	Positions   []FundPosition `json:"positions"`
}

type FundPosition struct {
	Symbol   string                 `json:"symbol"`
	Strategy string                 `json:"strategy"`
	Weight   float64                `json:"weight"`
	Params   map[string]interface{} `json:"params"`
}

func UpsertFund(fund Fund) error {
	return WithTx(func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			INSERT INTO funds (id, name, description, start_date, end_date, updated_at)
			VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT(id) DO UPDATE SET
				name = excluded.name,
				description = excluded.description,
				start_date = excluded.start_date,
				end_date = excluded.end_date,
				updated_at = CURRENT_TIMESTAMP
		`, fund.ID, fund.Name, fund.Description, fund.StartDate, fund.EndDate)
		if err != nil {
			return fmt.Errorf("写入基金失败: %w", err)
		}

		_, err = tx.Exec(`DELETE FROM fund_positions WHERE fund_id = ?`, fund.ID)
		if err != nil {
			return fmt.Errorf("清除旧持仓失败: %w", err)
		}

		stmt, err := tx.Prepare(`
			INSERT INTO fund_positions (fund_id, symbol, strategy, weight, params, sort_order)
			VALUES (?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			return fmt.Errorf("准备持仓插入语句失败: %w", err)
		}
		defer stmt.Close()

		for i, pos := range fund.Positions {
			paramsJSON, err := json.Marshal(pos.Params)
			if err != nil {
				return fmt.Errorf("序列化持仓参数失败: %w", err)
			}
			if _, err := stmt.Exec(fund.ID, pos.Symbol, pos.Strategy, pos.Weight, string(paramsJSON), i); err != nil {
				return fmt.Errorf("写入持仓 %s 失败: %w", pos.Symbol, err)
			}
		}

		return nil
	})
}

func GetAllFunds() ([]Fund, error) {
	if globalDB == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}
	rows, err := globalDB.Query(`SELECT id, name, description, start_date, end_date FROM funds ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var funds []Fund
	for rows.Next() {
		var f Fund
		if err := rows.Scan(&f.ID, &f.Name, &f.Description, &f.StartDate, &f.EndDate); err != nil {
			return nil, err
		}
		funds = append(funds, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range funds {
		positions, err := GetFundPositions(funds[i].ID)
		if err != nil {
			return nil, err
		}
		funds[i].Positions = positions
	}

	return funds, nil
}

func GetFundByID(fundID string) (*Fund, error) {
	if globalDB == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}
	var f Fund
	err := globalDB.QueryRow(`SELECT id, name, description, start_date, end_date FROM funds WHERE id = ?`, fundID).
		Scan(&f.ID, &f.Name, &f.Description, &f.StartDate, &f.EndDate)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	positions, err := GetFundPositions(fundID)
	if err != nil {
		return nil, err
	}
	f.Positions = positions

	return &f, nil
}

func GetFundPositions(fundID string) ([]FundPosition, error) {
	if globalDB == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}
	rows, err := globalDB.Query(`
		SELECT symbol, strategy, weight, params FROM fund_positions
		WHERE fund_id = ? ORDER BY sort_order
	`, fundID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []FundPosition
	for rows.Next() {
		var pos FundPosition
		var paramsJSON string
		if err := rows.Scan(&pos.Symbol, &pos.Strategy, &pos.Weight, &paramsJSON); err != nil {
			return nil, err
		}
		if paramsJSON != "" && paramsJSON != "{}" {
			pos.Params = make(map[string]interface{})
			if err := json.Unmarshal([]byte(paramsJSON), &pos.Params); err != nil {
				pos.Params = make(map[string]interface{})
			}
		} else {
			pos.Params = make(map[string]interface{})
		}
		positions = append(positions, pos)
	}
	return positions, rows.Err()
}

func DeleteFund(fundID string) error {
	return WithTx(func(tx *sql.Tx) error {
		_, err := tx.Exec(`DELETE FROM fund_positions WHERE fund_id = ?`, fundID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(`DELETE FROM funds WHERE id = ?`, fundID)
		return err
	})
}
