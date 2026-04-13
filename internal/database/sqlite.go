package database

import (
	"database/sql"
	"github.com/golang/glog"
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
        CREATE TABLE IF NOT EXISTS user_facts (
            category TEXT PRIMARY KEY,
            value TEXT NOT NULL,
            updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
        );
    `)
	if err != nil {
		return nil, err
	}

	glog.Infoln("SQLite database initialized at", dbPath)
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

// UpsertUserFact inserts or updates a user fact category.
func (s *SQLiteDB) UpsertUserFact(category, value string) error {
	_, err := s.db.Exec(`
        INSERT INTO user_facts (category, value, updated_at) 
        VALUES (?, ?, CURRENT_TIMESTAMP)
        ON CONFLICT(category) DO UPDATE SET 
            value = excluded.value,
            updated_at = excluded.updated_at
    `, category, value)
	return err
}

// GetUserFacts returns all user facts as a map.
func (s *SQLiteDB) GetUserFacts() (map[string]string, error) {
	rows, err := s.db.Query(`SELECT category, value FROM user_facts`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	facts := make(map[string]string)
	for rows.Next() {
		var category, value string
		if err := rows.Scan(&category, &value); err != nil {
			return nil, err
		}
		facts[category] = value
	}
	return facts, nil
}

type SessionRecord struct {
	ID        string
	Prompt    string
	CreatedAt string
}

func (s *SQLiteDB) SaveSession(id, prompt string) error {
	_, err := s.db.Exec(`INSERT INTO sessions (id, prompt) VALUES (?, ?)`, id, prompt)
	return err
}

func (s *SQLiteDB) SaveSessionLog(sessionID, agentRole, content string) error {
	_, err := s.db.Exec(`INSERT INTO session_logs (session_id, agent_role, content) VALUES (?, ?, ?)`, sessionID, agentRole, content)
	return err
}

func (s *SQLiteDB) GetSessions() ([]SessionRecord, error) {
	rows, err := s.db.Query(`SELECT id, prompt, created_at FROM sessions ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []SessionRecord
	for rows.Next() {
		var r SessionRecord
		if err := rows.Scan(&r.ID, &r.Prompt, &r.CreatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, r)
	}
	return sessions, nil
}

func (s *SQLiteDB) GetSessionLogs(sessionID string) (map[string]string, error) {
	rows, err := s.db.Query(`SELECT agent_role, content FROM session_logs WHERE session_id = ?`, sessionID, )
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make(map[string]string)
	for rows.Next() {
		var role, content string
		if err := rows.Scan(&role, &content); err != nil {
			return nil, err
		}
		logs[role] = content
	}
	return logs, nil
}
