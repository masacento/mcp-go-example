package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

type SQLite struct {
	db     *sql.DB
	dbPath string
}

func NewSQLite(path string) (*SQLite, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	return &SQLite{
		db:     db,
		dbPath: path,
	}, nil
}

func (s *SQLite) Call(name string, query string) (map[string]any, error) {
	switch name {
	case "read-query":
		if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "SELECT") {
			return nil, fmt.Errorf("only SELECT queries are allowed for read-query")
		}
		results, err := s.ExecuteQuery(query)
		if err != nil {
			return nil, err
		}
		js, _ := json.Marshal(results)
		return map[string]any{"content": []map[string]any{{"type": "text", "text": string(js)}}}, nil

	case "write-query":
		if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "SELECT") {
			return nil, fmt.Errorf("SELECT queries are not allowed for write-query")
		}
		affected, err := s.ExecuteWriteQuery(query)
		if err != nil {
			return nil, err
		}
		return map[string]any{"content": fmt.Sprintf("Affected rows: %d", affected)}, nil

	case "create-table":
		if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "CREATE TABLE") {
			return nil, fmt.Errorf("only CREATE TABLE statements are allowed")
		}
		_, err := s.ExecuteWriteQuery(query)
		if err != nil {
			return nil, err
		}
		return map[string]any{"content": "Table created successfully"}, nil

	case "list-tables":
		tables, err := s.ListTables()
		if err != nil {
			return nil, err
		}
		return map[string]any{"content": []map[string]any{{"type": "text", "text": strings.Join(tables, ", ")}}}, nil

	case "describe-table":
		description, err := s.DescribeTable(query)
		if err != nil {
			return nil, err
		}
		return map[string]any{"content": []map[string]any{{"type": "text", "text": description}}}, nil
	}

	return nil, fmt.Errorf("unknown call: %s", name)
}

func (s *SQLite) ExecuteQuery(query string, params ...any) ([]map[string]any, error) {
	rows, err := s.db.Query(query, params...)
	if err != nil {
		return nil, fmt.Errorf("query execution error: %w", err)
	}
	defer rows.Close()

	// カラム名の取得
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	// 結果の格納用スライス
	var results []map[string]any

	// 各行のスキャン用の変数を準備
	values := make([]any, len(columns))
	valuePtrs := make([]any, len(columns))
	for i := range columns {
		valuePtrs[i] = &values[i]
	}

	// 行のスキャンと結果の構築
	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("row scan error: %w", err)
		}

		row := make(map[string]any)
		for i, col := range columns {
			row[col] = values[i]
		}
		results = append(results, row)
	}

	return results, nil
}

func (s *SQLite) ExecuteWriteQuery(query string, params ...any) (int64, error) {
	result, err := s.db.Exec(query, params...)
	if err != nil {
		return 0, fmt.Errorf("write query execution error: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}

	return affected, nil
}

func (s *SQLite) Close() error {
	return s.db.Close()
}

func (s *SQLite) ListTables() ([]string, error) {
	query := "SELECT name FROM sqlite_master WHERE type='table'"
	results, err := s.ExecuteQuery(query)
	if err != nil {
		return nil, err
	}
	var tables []string
	for _, result := range results {
		tables = append(tables, result["name"].(string))
	}
	return tables, nil
}

func (s *SQLite) DescribeTable(table string) (string, error) {
	query := fmt.Sprintf("PRAGMA table_info(%s)", table)
	results, err := s.ExecuteQuery(query)
	if err != nil {
		return "", err
	}
	if len(results) == 0 {
		return "", nil
	}
	js, _ := json.Marshal(results[0])
	return string(js), nil
}
