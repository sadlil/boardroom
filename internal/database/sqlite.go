package database

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteDB struct {
	db *sql.DB
}

func NewSQLiteDB(dbPath string) (*SQLiteDB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Create tables
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS sessions (
            id TEXT PRIMARY KEY,
            prompt TEXT NOT NULL,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP
        );
        CREATE TABLE IF NOT EXISTS session_logs (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            session_id TEXT NOT NULL,
            agent_role TEXT NOT NULL,
            content TEXT NOT NULL,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY(session_id) REFERENCES sessions(id)
        );
        CREATE TABLE IF NOT EXISTS user_profile (
            id INTEGER PRIMARY KEY CHECK (id = 1),
            profile_data TEXT NOT NULL,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
        );
    `)
	if err != nil {
		return nil, err
	}

	log.Println("SQLite database initialized at", dbPath)
	return &SQLiteDB{db: db}, nil
}

func (s *SQLiteDB) Close() error {
	return s.db.Close()
}

// GetProfile retrieves the single user profile (if any).
func (s *SQLiteDB) GetProfile() (string, error) {
	var profileData string
	err := s.db.QueryRow(`SELECT profile_data FROM user_profile WHERE id = 1`).Scan(&profileData)
	if err == sql.ErrNoRows {
		return "", nil // Return empty string if no profile exists
	}
	if err != nil {
		return "", err
	}
	return profileData, nil
}

// SaveProfile inserts or updates the single user profile.
func (s *SQLiteDB) SaveProfile(profileData string) error {
	_, err := s.db.Exec(`
        INSERT INTO user_profile (id, profile_data, updated_at) 
        VALUES (1, ?, CURRENT_TIMESTAMP)
        ON CONFLICT(id) DO UPDATE SET 
            profile_data = excluded.profile_data,
            updated_at = excluded.updated_at
    `, profileData)
	return err
}

