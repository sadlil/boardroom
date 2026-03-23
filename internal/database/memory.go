package database

import (
	"context"
	"log"

	"github.com/philippgille/chromem-go"
)

type VectorMemory struct {
	collection *chromem.Collection
}

func NewVectorMemory(dbPath string) (*VectorMemory, error) {
	// Persistent vector database instance
	db, err := chromem.NewPersistentDB(dbPath, false)
	if err != nil {
		return nil, err
	}

	// Create or get collection using the default embedding function
	collection, err := db.GetOrCreateCollection("boardroom_memory", nil, nil)
	if err != nil {
		return nil, err
	}

	log.Println("Chromem-go vector memory initialized")
	return &VectorMemory{collection: collection}, nil
}

func (m *VectorMemory) AddDocument(ctx context.Context, id, content string, metadata map[string]string) error {
	doc := chromem.Document{
		ID:       id,
		Metadata: metadata,
		Content:  content,
	}
	// We pass simply an array of docs
	return m.collection.AddDocuments(ctx, []chromem.Document{doc}, 1)
}

func (m *VectorMemory) Search(ctx context.Context, query string, limit int) ([]chromem.Document, error) {
	res, err := m.collection.Query(ctx, query, limit, nil, nil)
	if err != nil {
		return nil, err
	}

	docs := make([]chromem.Document, 0, len(res))
	for _, r := range res {
		docs = append(docs, chromem.Document{
			ID:        r.ID,
			Metadata:  r.Metadata,
			Embedding: r.Embedding,
			Content:   r.Content,
		})
	}
	return docs, nil
}
