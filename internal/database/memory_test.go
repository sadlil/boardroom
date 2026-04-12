package database

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestVectorMemory(t *testing.T) {
	// Create a temporary directory for Chromem-go database
	tempDir, err := os.MkdirTemp("", "chromem_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // cleanup after test

	dbPath := filepath.Join(tempDir, "vector_mem")

	// Local mock embedding function for offline testing
	mockEmbed := func(ctx context.Context, text string) ([]float32, error) {
		return []float32{0.1, 0.2, 0.3}, nil
	}

	// Test Initialization
	memory, err := NewVectorMemory(dbPath, mockEmbed)
	if err != nil {
		t.Fatalf("Failed to initialize VectorMemory: %v", err)
	}
	if memory == nil {
		t.Fatal("Expected VectorMemory instance, got nil")
	}

	ctx := context.Background()

	// Test AddDocument
	docID := "doc-123"
	content := "The user is concerned about scaling their microservices architecture under high load."
	metadata := map[string]string{
		"category": "technical",
		"priority": "high",
	}

	err = memory.AddDocument(ctx, docID, content, metadata)
	if err != nil {
		t.Fatalf("Failed to add document: %v", err)
	}

	// Add noise document
	memory.AddDocument(ctx, "doc-456", "The new office in London is opening next week.", map[string]string{"category": "hr"})

	// Test Search (using limit 2 because we only added 2 documents)
	query := "microservices scaling load"
	docs, err := memory.Search(ctx, query, 2)
	if err != nil {
		t.Fatalf("Failed to search documents: %v", err)
	}

	if len(docs) == 0 {
		t.Errorf("Expected at least 1 document from search, got 0")
	}

	// The most relevant document should be returned first
	foundRelevant := false
	for _, doc := range docs {
		if doc.ID == docID {
			foundRelevant = true
			if doc.Metadata["category"] != "technical" {
				t.Errorf("Expected metadata 'category' to be 'technical', got '%s'", doc.Metadata["category"])
			}
			break
		}
	}

	if !foundRelevant {
		t.Errorf("Search failed to return the expected relevant document")
	}
}
