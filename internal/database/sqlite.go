package database

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/golang/glog"

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

// UpsertUserFact appends a new fact to the given category. The value column
// stores a JSON array of timestamped entries so facts accumulate over time
// rather than being silently overwritten by each new Scribe extraction.
//
// Schema: user_facts(category TEXT PK, value TEXT JSON array, updated_at)
func (s *SQLiteDB) UpsertUserFact(category, value string) error {
	// Read the existing value (if any)
	var existing string
	err := s.db.QueryRow(`SELECT value FROM user_facts WHERE category = ?`, category).Scan(&existing)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	// Parse existing entries or start a new array
	var entries []map[string]string
	if existing != "" {
		// Try parsing as JSON array (new format)
		if jsonErr := json.Unmarshal([]byte(existing), &entries); jsonErr != nil {
			// Legacy plain-text value — migrate it into the array format
			entries = []map[string]string{{"value": existing, "at": "legacy"}}
		}
	}

	// Deduplicate: skip if the exact same value is already the latest entry
	if len(entries) > 0 && entries[len(entries)-1]["value"] == value {
		return nil
	}

	// Append the new entry with a timestamp
	entries = append(entries, map[string]string{
		"value": value,
		"at":    time.Now().UTC().Format(time.RFC3339),
	})

	// Re-serialize and write back
	encoded, err := json.Marshal(entries)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
        INSERT INTO user_facts (category, value, updated_at) 
        VALUES (?, ?, CURRENT_TIMESTAMP)
        ON CONFLICT(category) DO UPDATE SET 
            value = excluded.value,
            updated_at = excluded.updated_at
    `, category, string(encoded))
	return err
}

func (s *SQLiteDB) DeleteUserFact(category string) error {
	_, err := s.db.Exec(`DELETE FROM user_facts WHERE category = ?`, category)
	return err
}

// GetUserFacts returns all user facts as a map, with each category's value
// being the latest entry from the accumulated JSON array.
func (s *SQLiteDB) GetUserFacts() (map[string]string, error) {
	rows, err := s.db.Query(`SELECT category, value FROM user_facts`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	facts := make(map[string]string)
	for rows.Next() {
		var category, rawValue string
		if err := rows.Scan(&category, &rawValue); err != nil {
			return nil, err
		}

		// Try to parse as JSON array (new accumulated format)
		var entries []map[string]string
		if json.Unmarshal([]byte(rawValue), &entries) == nil && len(entries) > 0 {
			// Return the latest entry's value
			facts[category] = entries[len(entries)-1]["value"]
		} else {
			// Legacy plain-text value — return as-is
			facts[category] = rawValue
		}
	}
	return facts, nil
}

type SessionRecord struct {
	ID             string
	Prompt         string
	CreatedAt      string
	VerdictPreview string // First ~150 chars of the decider output (if available)
}

func (s *SQLiteDB) SaveSession(id, prompt string) error {
	_, err := s.db.Exec(`INSERT INTO sessions (id, prompt) VALUES (?, ?)`, id, prompt)
	return err
}

func (s *SQLiteDB) SaveSessionLog(sessionID, agentRole, content string) error {
	_, err := s.db.Exec(`INSERT INTO session_logs (session_id, agent_role, content) VALUES (?, ?, ?)`, sessionID, agentRole, content)
	return err
}

// GetSessions returns paginated sessions with optional keyword search and a
// verdict preview from the decider's session log.
func (s *SQLiteDB) GetSessions(search string, limit, offset int) ([]SessionRecord, error) {
	if limit <= 0 {
		limit = 20
	}

	query := `
		SELECT s.id, s.prompt, s.created_at, COALESCE(SUBSTR(l.content, 1, 150), '') as verdict
		FROM sessions s
		LEFT JOIN session_logs l ON l.session_id = s.id AND l.agent_role = 'decider'
	`
	var args []interface{}

	if search != "" {
		query += " WHERE s.prompt LIKE ?"
		args = append(args, "%"+search+"%")
	}

	query += " GROUP BY s.id ORDER BY s.created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []SessionRecord
	for rows.Next() {
		var r SessionRecord
		if err := rows.Scan(&r.ID, &r.Prompt, &r.CreatedAt, &r.VerdictPreview); err != nil {
			return nil, err
		}
		sessions = append(sessions, r)
	}
	return sessions, nil
}

// GetSessionCount returns the total number of sessions, optionally filtered by search.
func (s *SQLiteDB) GetSessionCount(search string) (int, error) {
	query := `SELECT COUNT(*) FROM sessions`
	var args []interface{}
	if search != "" {
		query += " WHERE prompt LIKE ?"
		args = append(args, "%"+search+"%")
	}
	var count int
	err := s.db.QueryRow(query, args...).Scan(&count)
	return count, err
}

// DeleteSession removes a session and its associated logs.
func (s *SQLiteDB) DeleteSession(id string) error {
	_, err := s.db.Exec(`DELETE FROM session_logs WHERE session_id = ?`, id)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	return err
}

func (s *SQLiteDB) GetSessionLogs(sessionID string) (map[string]string, error) {
	rows, err := s.db.Query(`SELECT agent_role, content FROM session_logs WHERE session_id = ?`, sessionID)
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
