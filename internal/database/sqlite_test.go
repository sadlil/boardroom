package database

import (
	"path/filepath"
	"testing"
)

func TestSQLiteDB(t *testing.T) {
	// Use an in-memory database for testing isolation
	// The path file::memory:?cache=shared creates a distinct in-memory DB.
	// However, mattn/go-sqlite3 also supports simply passing an empty string or :memory:
	// To be safe in case of pathing logic, we will use t.TempDir()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := NewSQLiteDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize SQLiteDB: %v", err)
	}
	defer db.Close()

	// Test GetProfile (Empty State)
	profile, err := db.GetProfile()
	if err != nil {
		t.Fatalf("GetProfile failed on pristine DB: %v", err)
	}
	if profile != "" {
		t.Errorf("Expected empty profile, got '%s'", profile)
	}

	// Test SaveProfile
	expectedProfile := `{"role": "CEO", "goals": "Expand business"}`
	err = db.SaveProfile(expectedProfile)
	if err != nil {
		t.Fatalf("SaveProfile failed: %v", err)
	}

	// Test GetProfile (Populated State)
	profile, err = db.GetProfile()
	if err != nil {
		t.Fatalf("GetProfile failed after save: %v", err)
	}
	if profile != expectedProfile {
		t.Errorf("Expected profile '%s', got '%s'", expectedProfile, profile)
	}

	// Test Overwrite Profile
	newProfile := `{"role": "CTO", "goals": "Fix servers"}`
	err = db.SaveProfile(newProfile)
	if err != nil {
		t.Fatalf("SaveProfile (overwrite) failed: %v", err)
	}

	profile, _ = db.GetProfile()
	if profile != newProfile {
		t.Errorf("Expected updated profile '%s', got '%s'", newProfile, profile)
	}
}
