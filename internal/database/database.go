package database

import (
	"database/sql"
	"fmt"

	"github.com/driquet/ezbp/internal/boilerplate"
	_ "github.com/mattn/go-sqlite3"
)

// Database defines the interface for boilerplate database operations
type Database interface {
	// GetAllBoilerplates returns all boilerplates as a map with name as key
	GetAllBoilerplates() (map[string]*boilerplate.Boilerplate, error)

	// GetBoilerplateByName returns a specific boilerplate by name
	GetBoilerplateByName(name string) (*boilerplate.Boilerplate, error)

	// CreateBoilerplate creates a new boilerplate
	CreateBoilerplate(boilerplate *boilerplate.Boilerplate) error

	// UpdateBoilerplate updates an existing boilerplate
	UpdateBoilerplate(boilerplate *boilerplate.Boilerplate) error

	// DeleteBoilerplate deletes a boilerplate by name
	DeleteBoilerplate(name string) error

	// IncBoilerplateCount increments the usage count for a boilerplate
	IncBoilerplateCount(name string) error

	// Close closes the database connection
	Close() error
}

// SQLiteDatabase implements the Database interface using SQLite
type SQLiteDatabase struct {
	db *sql.DB
}

// NewSQLiteDatabase creates a new SQLite database connection
func NewSQLiteDatabase(dbPath string) (*SQLiteDatabase, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	sqliteDB := &SQLiteDatabase{db: db}

	// Initialize the database schema
	if err := sqliteDB.initSchema(); err != nil {
		db.Close()
		return nil, err
	}

	return sqliteDB, nil
}

// initSchema creates the boilerplates table if it doesn't exist
func (s *SQLiteDatabase) initSchema() error {
	query := `
	CREATE TABLE IF NOT EXISTS boilerplates (
		name TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		count INTEGER DEFAULT 0
	);`

	_, err := s.db.Exec(query)
	return err
}

// GetAllBoilerplates returns all boilerplates as a map with name as key
func (s *SQLiteDatabase) GetAllBoilerplates() (map[string]*boilerplate.Boilerplate, error) {
	query := "SELECT name, value, count FROM boilerplates ORDER BY name"
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	boilerplates := make(map[string]*boilerplate.Boilerplate)
	for rows.Next() {
		b := &boilerplate.Boilerplate{}
		if err := rows.Scan(&b.Name, &b.Value, &b.Count); err != nil {
			return nil, err
		}
		boilerplates[b.Name] = b
	}

	return boilerplates, rows.Err()
}

// GetBoilerplateByName returns a specific boilerplate by name
func (s *SQLiteDatabase) GetBoilerplateByName(name string) (*boilerplate.Boilerplate, error) {
	query := "SELECT name, value, count FROM boilerplates WHERE name = ?"
	row := s.db.QueryRow(query, name)

	var b boilerplate.Boilerplate
	err := row.Scan(&b.Name, &b.Value, &b.Count)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("unknown boilerplate %q", name)
		}
		return nil, err
	}

	return &b, nil
}

// CreateBoilerplate creates a new boilerplate
func (s *SQLiteDatabase) CreateBoilerplate(bp *boilerplate.Boilerplate) error {
	query := "INSERT INTO boilerplates (name, value, count) VALUES (?, ?, ?)"
	_, err := s.db.Exec(query, bp.Name, bp.Value, bp.Count)
	return err
}

// UpdateBoilerplate updates an existing boilerplate
func (s *SQLiteDatabase) UpdateBoilerplate(bp *boilerplate.Boilerplate) error {
	query := "UPDATE boilerplates SET value = ?, count = ? WHERE name = ?"
	result, err := s.db.Exec(query, bp.Value, bp.Count, bp.Name)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("unknown boilerplate %q", bp.Name)
	}

	return nil
}

// DeleteBoilerplate deletes a boilerplate by name
func (s *SQLiteDatabase) DeleteBoilerplate(name string) error {
	query := "DELETE FROM boilerplates WHERE name = ?"
	result, err := s.db.Exec(query, name)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("unknown boilerplate %q", name)
	}

	return nil
}

// IncBoilerplateCount increments the usage count for a boilerplate
func (s *SQLiteDatabase) IncBoilerplateCount(name string) error {
	query := "UPDATE boilerplates SET count = count + 1 WHERE name = ?"
	result, err := s.db.Exec(query, name)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("unknown boilerplate %q", name)
	}

	return nil
}

// Close closes the database connection
func (s *SQLiteDatabase) Close() error {
	return s.db.Close()
}
