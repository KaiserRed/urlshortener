package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	var storagePath, migrationsPath string

	flag.StringVar(&storagePath, "storage-path", "", "path to the SQLite database")
	flag.StringVar(&migrationsPath, "migrations-path", "", "directory with .sql migrations")
	flag.Parse()

	if storagePath == "" {
		log.Fatal("storage-path is required")
	}
	if migrationsPath == "" {
		log.Fatal("migrations-path is required")
	}

	db, err := sql.Open("sqlite3", storagePath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	if err := ensureMigrationsTable(db); err != nil {
		log.Fatalf("init migrations table: %v", err)
	}

	files, err := filepath.Glob(filepath.Join(migrationsPath, "*.up.sql"))
	if err != nil {
		log.Fatalf("list migrations: %v", err)
	}
	sort.Strings(files)

	for _, path := range files {
		version := versionFromPath(path)

		applied, err := isApplied(db, version)
		if err != nil {
			log.Fatalf("check migration %d: %v", version, err)
		}
		if applied {
			continue
		}

		content, err := os.ReadFile(path)
		if err != nil {
			log.Fatalf("read migration %s: %v", path, err)
		}

		if err := applyMigration(db, version, string(content)); err != nil {
			log.Fatalf("apply migration %s: %v", path, err)
		}

		fmt.Printf("applied %s\n", filepath.Base(path))
	}

	fmt.Println("migrations applied successfully")
}

func ensureMigrationsTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations(
		version INTEGER PRIMARY KEY,
		applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	return err
}

func isApplied(db *sql.DB, version int) (bool, error) {
	var v int
	err := db.QueryRow(`SELECT version FROM schema_migrations WHERE version = ?`, version).Scan(&v)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func applyMigration(db *sql.DB, version int, content string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, stmt := range splitStatements(content) {
		if _, err := tx.Exec(stmt); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(`INSERT INTO schema_migrations(version) VALUES(?)`, version); err != nil {
		return err
	}

	return tx.Commit()
}

func splitStatements(content string) []string {
	var out []string
	for _, part := range strings.Split(content, ";") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func versionFromPath(path string) int {
	base := filepath.Base(path)

	end := 0
	for end < len(base) && base[end] >= '0' && base[end] <= '9' {
		end++
	}

	v, err := strconv.Atoi(base[:end])
	if err != nil {
		return 0
	}
	return v
}
