package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

// connectDatabaseMulti creates a database connection with support for PostgreSQL, MySQL, and SQLite
func connectDatabaseMulti() (*bun.DB, error) {
	// Get database URL from config or environment
	dbURL := viper.GetString("database.url")
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		dbURL = "authsome.db" // Default SQLite database
	}

	var sqldb *sql.DB
	var db *bun.DB
	var err error

	// Determine database type from URL
	if strings.HasPrefix(dbURL, "postgres://") || strings.HasPrefix(dbURL, "postgresql://") {
		// PostgreSQL
		connector := pgdriver.NewConnector(pgdriver.WithDSN(dbURL))
		sqldb = sql.OpenDB(connector)
		db = bun.NewDB(sqldb, pgdialect.New())
		if verbose {
			log.Printf("Connected to PostgreSQL")
		}
	} else if strings.HasPrefix(dbURL, "mysql://") {
		// MySQL
		// Remove mysql:// prefix for go-sql-driver
		mysqlDSN := strings.TrimPrefix(dbURL, "mysql://")
		sqldb, err = sql.Open("mysql", mysqlDSN)
		if err != nil {
			return nil, fmt.Errorf("failed to open MySQL connection: %w", err)
		}
		db = bun.NewDB(sqldb, mysqldialect.New())
		if verbose {
			log.Printf("Connected to MySQL")
		}
	} else {
		// SQLite (default)
		// Ensure the directory exists for file-based SQLite
		if dbURL != ":memory:" && !strings.HasPrefix(dbURL, "file:") {
			dir := filepath.Dir(dbURL)
			if dir != "." && dir != "" {
				if err := os.MkdirAll(dir, 0755); err != nil {
					return nil, fmt.Errorf("failed to create database directory: %w", err)
				}
			}
		}

		sqldb, err = sql.Open("sqlite3", dbURL)
		if err != nil {
			return nil, fmt.Errorf("failed to open SQLite connection: %w", err)
		}
		db = bun.NewDB(sqldb, sqlitedialect.New())
		if verbose {
			log.Printf("Connected to SQLite: %s", dbURL)
		}
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
