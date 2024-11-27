package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSQLite(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := NewSQLite(dbPath)
	if err != nil {
		t.Fatalf("Failed to create new database: %v", err)
	}
	defer func() {
		db.Close()
		os.Remove(dbPath)
	}()

	_, err = db.ExecuteWriteQuery(`
		CREATE TABLE IF NOT EXISTS test_table (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	_, err = db.ExecuteWriteQuery(`
		INSERT INTO test_table (name) VALUES (?), (?)
	`, "test1", "test2")
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	results, err := db.ExecuteQuery("SELECT * FROM test_table")
	if err != nil {
		t.Fatalf("Failed to query data: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	if name, ok := results[0]["name"].(string); !ok || name != "test1" {
		t.Errorf("Expected first row name to be 'test1', got %v", results[0]["name"])
	}

	if name, ok := results[1]["name"].(string); !ok || name != "test2" {
		t.Errorf("Expected second row name to be 'test2', got %v", results[1]["name"])
	}
}

func TestSQLiteErrors(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := NewSQLite(dbPath)
	if err != nil {
		t.Fatalf("Failed to create new database: %v", err)
	}
	defer func() {
		db.Close()
		os.Remove(dbPath)
	}()

	_, err = db.ExecuteQuery("SELECT * FROM non_existent_table")
	if err == nil {
		t.Error("Expected error for invalid query, got nil")
	}

	_, err = db.ExecuteWriteQuery("INSERT INTO non_existent_table VALUES (?)", "test")
	if err == nil {
		t.Error("Expected error for invalid write query, got nil")
	}
}

func TestSQLiteDescribeTable(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := NewSQLite(dbPath)
	if err != nil {
		t.Fatalf("Failed to create new database: %v", err)
	}
	defer func() {
		db.Close()
		os.Remove(dbPath)
	}()

	_, err = db.ExecuteWriteQuery(`
		CREATE TABLE IF NOT EXISTS test_table (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	description, err := db.DescribeTable("test_table")
	if err != nil {
		t.Fatalf("Failed to describe table: %v", err)
	}

	expectedDescription := `{"cid":0,"dflt_value":null,"name":"id","notnull":0,"pk":1,"type":"INTEGER"}`
	if description != expectedDescription {
		t.Errorf("Expected description %v, got %v", expectedDescription, description)
	}
}
