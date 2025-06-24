package db

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type DB struct {
	conn *sql.DB
}

// NewDB creates a connection to Neon PostgreSQL database
func NewDB(connectionString string) (*DB, error) {
	conn, err := sql.Open("pgx", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	// Verify connection
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	// PostgreSQL-compatible table creation
	_, err = conn.Exec(`
		CREATE TABLE IF NOT EXISTS urls (
			id SERIAL PRIMARY KEY,
			url TEXT NOT NULL UNIQUE,
			parent_url TEXT
		)
	`)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create table: %v", err)
	}

	return &DB{conn: conn}, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) StoreLinks(links []string, parentURLs map[string]string) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}

	// PostgreSQL uses ON CONFLICT instead of INSERT OR IGNORE
	stmt, err := tx.Prepare(`
		INSERT INTO urls (url, parent_url) 
		VALUES ($1, $2)
		ON CONFLICT (url) DO NOTHING
	`)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	for _, link := range links {
		parent := parentURLs[link]
		if _, err := stmt.Exec(link, parent); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to insert link %s: %v", link, err)
		}
	}

	return tx.Commit()
}

func (db *DB) ShowStoreLinks() error {
	row, err := db.conn.Query("SELECT id, url, parent_url FROM urls")

	if err != nil {
		return fmt.Errorf("error while querring data :- %v", err)
	}

	defer row.Close()

	fmt.Println("\nStored Links:")
	fmt.Println("ID | URL | Parent URL")
	fmt.Println("----------------------")

	for row.Next() {
		var id int
		var url, parentURL string
		if err := row.Scan(&id, &url, &parentURL); err != nil {
			return fmt.Errorf("error scanning row: %v", err)
		}

		fmt.Printf("%d | %s | %s\n", id, url, parentURL)
	}

	if err := row.Err(); err != nil {
		return fmt.Errorf("error during row iteration: %v", err)
	}

	return nil
}
